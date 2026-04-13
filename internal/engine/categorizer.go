package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"UnifiedBookmarks-Desktop/internal/browser"
	"UnifiedBookmarks-Desktop/internal/config"
)

// ProgressInfo sent during categorization
type ProgressInfo struct {
	Batch      int    `json:"batch"`
	Total      int    `json:"total"`
	Tokens     int    `json:"tokens"`
	BatchItems int    `json:"batchItems"`
	Elapsed    string `json:"elapsed"`
	Message    string `json:"message"`
}

// CategorizeResult holds the final result
type CategorizeResult struct {
	Bookmarks   []browser.Bookmark `json:"bookmarks"`
	TotalTokens int                `json:"totalTokens"`
	TotalBatch  int                `json:"totalBatches"`
	Model       string             `json:"model"`
}

const systemPrompt = `You are a bookmark organizer and PKM classifier. You receive browser bookmarks with their current folder path, title, and URL.

Your job:
1. Review each bookmark's CURRENT folder path — respect the user's existing organization when it makes sense
2. Consolidate and clean up categories — merge similar/redundant folders, fix inconsistent naming
3. Assign a clean category path using "/" as separator (e.g., "Dev/Tools", "News/Vietnam")
4. Classify each bookmark using the PARA method (Personal Knowledge Management)
5. Suggest 1-3 short lowercase tags per bookmark

Category rules:
- KEEP the user's existing folder structure when it's well-organized
- Merge duplicated or near-identical folders
- Normalize naming: concise English, Title Case, max 2-3 levels deep
- Confidence: 0.9+ if keeping original path, 0.7-0.9 if moved, <0.7 if guessing
- If a URL is unclear, use "Uncategorized"

PARA classification:
- "project": actively used for a current goal with a deadline
- "area": ongoing responsibility without end date
- "resource": reference/learning material not actively used right now
- "archive": completed/inactive content
- null: if unsure

paraContext: a short name grouping related bookmarks.
tags: 1-3 short lowercase tags per bookmark.

Return ONLY valid JSON:
{"bookmarks":[{"url":"...","category":"...","confidence":0.95,"paraType":"resource","paraContext":"Web Development","tags":["tool","dev"]}, ...]}`

type aiRequest struct {
	Model          string      `json:"model"`
	Messages       []aiMessage `json:"messages"`
	Temperature    float64     `json:"temperature"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type aiResult struct {
	Bookmarks []struct {
		URL         string   `json:"url"`
		Category    string   `json:"category"`
		Confidence  float64  `json:"confidence"`
		ParaType    string   `json:"paraType"`
		ParaContext string   `json:"paraContext"`
		Tags        []string `json:"tags"`
	} `json:"bookmarks"`
}

// Categorize sends bookmarks to the AI for categorization with progress callback
func Categorize(ctx context.Context, bookmarks []browser.Bookmark, cfg *config.Config, onProgress func(ProgressInfo)) (*CategorizeResult, error) {
	if cfg.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = len(bookmarks) // 0 = unlimited: send all in one request
	}

	// Split into batches
	var batches [][]browser.Bookmark
	for i := 0; i < len(bookmarks); i += batchSize {
		end := i + batchSize
		if end > len(bookmarks) {
			end = len(bookmarks)
		}
		batches = append(batches, bookmarks[i:end])
	}

	if onProgress != nil {
		onProgress(ProgressInfo{Message: fmt.Sprintf("Processing %d bookmark(s) in %d batch(es)...", len(bookmarks), len(batches))})
	}

	categoryMap := map[string]catEntry{}
	totalTokens := 0

	for i, batch := range batches {
		start := time.Now()
		results, tokens, err := categorizeBatch(ctx, batch, cfg)
		elapsed := time.Since(start).Seconds()

		if err != nil {
			if onProgress != nil {
				onProgress(ProgressInfo{Message: fmt.Sprintf("Batch %d failed: %s", i+1, err.Error())})
			}
			continue
		}

		totalTokens += tokens
		for _, r := range results {
			paraType := ""
			switch r.ParaType {
			case "project", "area", "resource", "archive":
				paraType = r.ParaType
			}
			var tags []string
			for _, t := range r.Tags {
				t = strings.ToLower(strings.TrimSpace(t))
				if t != "" {
					tags = append(tags, t)
				}
			}
			categoryMap[r.URL] = catEntry{
				Category:    r.Category,
				Confidence:  r.Confidence,
				ParaType:    paraType,
				ParaContext: r.ParaContext,
				Tags:        tags,
			}
		}

		if onProgress != nil {
			onProgress(ProgressInfo{
				Batch:      i + 1,
				Total:      len(batches),
				Tokens:     totalTokens,
				BatchItems: len(batch),
				Elapsed:    fmt.Sprintf("%.1f", elapsed),
			})
		}
	}

	// Merge results
	result := make([]browser.Bookmark, len(bookmarks))
	for i, bm := range bookmarks {
		result[i] = bm
		if entry, ok := categoryMap[bm.URL]; ok {
			result[i].Category = entry.Category
			result[i].Confidence = entry.Confidence
			result[i].ParaType = entry.ParaType
			result[i].ParaContext = entry.ParaContext
			if len(entry.Tags) > 0 {
				result[i].Tags = entry.Tags
			}
		} else {
			result[i].Category = "Uncategorized"
		}
	}

	return &CategorizeResult{
		Bookmarks:   result,
		TotalTokens: totalTokens,
		TotalBatch:  len(batches),
		Model:       cfg.OpenAIModel,
	}, nil
}

type catEntry struct {
	Category    string
	Confidence  float64
	ParaType    string
	ParaContext string
	Tags        []string
}

func categorizeBatch(ctx context.Context, batch []browser.Bookmark, cfg *config.Config) ([]struct {
	URL         string   `json:"url"`
	Category    string   `json:"category"`
	Confidence  float64  `json:"confidence"`
	ParaType    string   `json:"paraType"`
	ParaContext string   `json:"paraContext"`
	Tags        []string `json:"tags"`
}, int, error) {

	// If custom prompt is set, prepend it to the built-in prompt so
	// the JSON output schema is always preserved (custom rules + required format).
	sysPrompt := systemPrompt
	if cfg.SystemPrompt != "" {
		sysPrompt = cfg.SystemPrompt + "\n\n" + systemPrompt
	}

	userPrompt := buildUserPrompt(batch)

	reqBody := aiRequest{
		Model:       cfg.OpenAIModel,
		Temperature: 0.2,
		Messages: []aiMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	reqBody.ResponseFormat.Type = "json_object"

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, err
		}

		baseURL := strings.TrimRight(cfg.OpenAIBase, "/")
		endpoint := baseURL + "/chat/completions"

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)
		// Azure OpenAI uses api-key header
		req.Header.Set("api-key", cfg.OpenAIKey)

		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(math.Pow(2, float64(attempt-1))) * time.Second)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
			time.Sleep(time.Duration(math.Pow(2, float64(attempt-1))) * time.Second)
			continue
		}

		if err != nil {
			lastErr = err
			continue
		}

		var aiResp aiResponse
		if err := json.Unmarshal(respBody, &aiResp); err != nil {
			lastErr = err
			continue
		}

		if len(aiResp.Choices) == 0 {
			lastErr = fmt.Errorf("no choices in response")
			continue
		}

		content := aiResp.Choices[0].Message.Content
		var parsed aiResult
		if err := json.Unmarshal([]byte(content), &parsed); err != nil {
			// Try bare array
			var arr []struct {
				URL         string   `json:"url"`
				Category    string   `json:"category"`
				Confidence  float64  `json:"confidence"`
				ParaType    string   `json:"paraType"`
				ParaContext string   `json:"paraContext"`
				Tags        []string `json:"tags"`
			}
			if err2 := json.Unmarshal([]byte(content), &arr); err2 == nil {
				return arr, aiResp.Usage.TotalTokens, nil
			}
			lastErr = err
			continue
		}

		return parsed.Bookmarks, aiResp.Usage.TotalTokens, nil
	}

	return nil, 0, fmt.Errorf("all %d attempts failed: %w", cfg.MaxRetries, lastErr)
}

func buildUserPrompt(bookmarks []browser.Bookmark) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Organize these %d bookmarks. Their current folder paths are shown in brackets:\n\n", len(bookmarks)))
	for i, bm := range bookmarks {
		folder := bm.FolderPath
		if folder == "" {
			folder = "Unknown"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] \"%s\" | %s\n", i+1, folder, bm.Title, bm.URL))
	}
	return sb.String()
}
