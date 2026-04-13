package sync

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"UnifiedBookmarks-Desktop/internal/browser"
)

// WriteResult holds the result of a sync operation
type WriteResult struct {
	Browser string `json:"browser"`
	Profile string `json:"profile"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Written int    `json:"written"`
	Folders int    `json:"folders"`
}

var nextID int

func getNextID() string {
	nextID++
	return strconv.Itoa(nextID)
}

func chromeNow() string {
	// Chrome epoch: 1601-01-01 → microseconds
	const epochDiffUS = 11644473600 * 1_000_000
	nowUS := time.Now().UnixMicro() + epochDiffUS
	return strconv.FormatInt(nowUS, 10)
}

// WriteBookmarks builds a Chrome Bookmarks JSON and writes it atomically.
// This REPLACES all existing bookmarks (clean sync — fixes the duplicate issue).
func WriteBookmarks(bookmarks []browser.Bookmark, bookmarksPath string, dryRun bool) (*WriteResult, error) {
	nextID = 3 // 0=root, 1=bar, 2=other, 3=synced reserved

	barChildren, folderCount := buildChromeTree(bookmarks)

	roots := map[string]interface{}{
		"bookmark_bar": map[string]interface{}{
			"children":       barChildren,
			"date_added":     chromeNow(),
			"date_last_used": "0",
			"date_modified":  chromeNow(),
			"id":             "1",
			"name":           "Bookmarks bar",
			"type":           "folder",
		},
		"other": map[string]interface{}{
			"children":       []interface{}{},
			"date_added":     chromeNow(),
			"date_last_used": "0",
			"date_modified":  chromeNow(),
			"id":             "2",
			"name":           "Other bookmarks",
			"type":           "folder",
		},
		"synced": map[string]interface{}{
			"children":       []interface{}{},
			"date_added":     chromeNow(),
			"date_last_used": "0",
			"date_modified":  chromeNow(),
			"id":             "3",
			"name":           "Mobile bookmarks",
			"type":           "folder",
		},
	}

	data := map[string]interface{}{
		"roots":   roots,
		"version": 1,
	}

	if dryRun {
		return &WriteResult{
			Status:  "dry-run",
			Written: len(bookmarks),
			Folders: folderCount,
		}, nil
	}

	// Marshal with indent first (this is what Chrome expects to read)
	output, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return nil, err
	}

	// Compute checksum on the FINAL output bytes — Chrome validates this
	checksum := fmt.Sprintf("%x", md5.Sum(output))

	// Re-insert checksum and re-marshal so the final file has the correct checksum
	data["checksum"] = checksum
	output, err = json.MarshalIndent(data, "", "   ")
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(bookmarksPath)
	tmpFile := filepath.Join(dir, "Bookmarks.tmp.ub")
	if err := os.WriteFile(tmpFile, output, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tmpFile, bookmarksPath); err != nil {
		os.Remove(tmpFile)
		return nil, fmt.Errorf("failed to rename: %w", err)
	}

	// Also overwrite Bookmarks.bak so Chrome cannot restore the old version.
	// If .bak is stale/missing, Chrome may detect checksum mismatch and roll back.
	bakPath := bookmarksPath + ".bak"
	_ = os.WriteFile(bakPath, output, 0644) // best-effort, ignore error

	return &WriteResult{
		Status:  "success",
		Written: len(bookmarks),
		Folders: folderCount,
	}, nil
}

type treeNode struct {
	bookmarks []browser.Bookmark
	children  map[string]*treeNode
}

func buildChromeTree(bookmarks []browser.Bookmark) ([]interface{}, int) {
	root := &treeNode{children: map[string]*treeNode{}}
	folderCount := 0

	for _, bm := range bookmarks {
		parts := splitCategory(bm.Category)
		node := root
		for _, part := range parts {
			if node.children == nil {
				node.children = map[string]*treeNode{}
			}
			if _, ok := node.children[part]; !ok {
				node.children[part] = &treeNode{children: map[string]*treeNode{}}
				folderCount++
			}
			node = node.children[part]
		}
		node.bookmarks = append(node.bookmarks, bm)
	}

	return convertNode(root), folderCount
}

func convertNode(node *treeNode) []interface{} {
	var children []interface{}

	// Sort folder names
	var names []string
	for name := range node.children {
		names = append(names, name)
	}
	sortStrings(names)

	for _, name := range names {
		child := node.children[name]
		children = append(children, map[string]interface{}{
			"children":       convertNode(child),
			"date_added":     chromeNow(),
			"date_last_used": "0",
			"date_modified":  chromeNow(),
			"id":             getNextID(),
			"name":           name,
			"type":           "folder",
		})
	}

	for _, bm := range node.bookmarks {
		dateAdded := bm.DateAdded
		if dateAdded == "" {
			dateAdded = chromeNow()
		}
		children = append(children, map[string]interface{}{
			"date_added":     dateAdded,
			"date_last_used": "0",
			"id":             getNextID(),
			"name":           bm.Title,
			"type":           "url",
			"url":            bm.URL,
		})
	}

	return children
}

func splitCategory(cat string) []string {
	if cat == "" {
		cat = "Uncategorized"
	}
	var parts []string
	for _, p := range strings.Split(cat, "/") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return []string{"Uncategorized"}
	}
	return parts
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// SyncToProfile writes bookmarks back to a specific profile, replacing existing bookmarks
func SyncToProfile(bookmarks []browser.Bookmark, profile browser.Profile) (*WriteResult, error) {
	result, err := WriteBookmarks(bookmarks, profile.BookmarksPath, false)
	if err != nil {
		return nil, err
	}
	result.Browser = profile.BrowserLabel
	result.Profile = profile.DisplayName
	return result, nil
}
