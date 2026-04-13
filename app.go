package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"UnifiedBookmarks-Desktop/internal/browser"
	"UnifiedBookmarks-Desktop/internal/config"
	"UnifiedBookmarks-Desktop/internal/engine"
	bsync "UnifiedBookmarks-Desktop/internal/sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct — Wails bindings
type App struct {
	ctx       context.Context
	cfg       *config.Config
	profiles  []browser.Profile
	bookmarks []browser.Bookmark
	backupDir string
	cacheDir  string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = config.Load()

	// Set backup dir
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	a.backupDir = a.cfg.BackupDir
	if a.backupDir == "" {
		a.backupDir = filepath.Join(base, "backups")
	}
	// Fallback for development (wails dev uses a temp exe, not build/bin/)
	if _, err := os.Stat(a.backupDir); os.IsNotExist(err) {
		if cwd, err := os.Getwd(); err == nil {
			// Prefer build/bin/backups if it exists (mirrors production layout)
			candidate := filepath.Join(cwd, "build", "bin", "backups")
			if _, err2 := os.Stat(candidate); err2 == nil {
				a.backupDir = candidate
			} else {
				a.backupDir = filepath.Join(cwd, "backups")
			}
		}
	}

	// Cache dir: sibling of backups
	a.cacheDir = filepath.Join(filepath.Dir(a.backupDir), "cache")
	if _, err := os.Stat(a.cacheDir); os.IsNotExist(err) {
		if cwd, err := os.Getwd(); err == nil {
			// Prefer build/bin/cache if it exists (mirrors production layout)
			candidate := filepath.Join(cwd, "build", "bin", "cache")
			if _, err2 := os.Stat(candidate); err2 == nil {
				a.cacheDir = candidate
			} else {
				a.cacheDir = filepath.Join(cwd, "cache")
			}
		}
	}
	os.MkdirAll(a.cacheDir, 0755)
}

// ── In-app page fetcher ───────────────────────────────
// FetchPageHTML fetches a URL server-side, strips X-Frame-Options /
// Content-Security-Policy, injects a <base> tag, and returns the HTML.
// The frontend sets iframe srcdoc to avoid any networking inside WebView2.

// FetchPageResult is the return type for FetchPageHTML.
type FetchPageResult struct {
	HTML  string `json:"html"`
	Error string `json:"error"`
}

func (a *App) FetchPageHTML(rawURL string) FetchPageResult {
	targetURL, err := url.Parse(rawURL)
	if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		return FetchPageResult{Error: "invalid url"}
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return FetchPageResult{Error: err.Error()}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return FetchPageResult{Error: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchPageResult{Error: err.Error()}
	}

	// Inject <base> tag so relative URLs (images, CSS, JS) resolve correctly
	base := targetURL.Scheme + "://" + targetURL.Host + "/"
	baseTag := []byte(`<base href="` + base + `">`)
	injected := false
	for _, tag := range [][]byte{[]byte("<head>"), []byte("<HEAD>")} {
		if idx := bytes.Index(body, tag); idx >= 0 {
			ins := idx + len(tag)
			body = append(body[:ins], append(baseTag, body[ins:]...)...)
			injected = true
			break
		}
	}
	if !injected {
		body = append(baseTag, body...)
	}

	return FetchPageResult{HTML: string(body)}
}

// ── Scan ──────────────────────────────────────────────

type ScanResult struct {
	Profiles []browser.Profile `json:"profiles"`
	OS       string            `json:"os"`
}

func (a *App) ScanProfiles() (*ScanResult, error) {
	profiles, err := browser.ScanProfiles()
	if err != nil {
		return nil, err
	}
	a.profiles = profiles
	return &ScanResult{
		Profiles: profiles,
		OS:       runtime.GOOS,
	}, nil
}

// ── Folder Tree ───────────────────────────────────────

// FolderNode is a recursive tree node representing a bookmark folder.
type FolderNode struct {
	Name     string       `json:"name"`
	Path     string       `json:"path"`
	Count    int          `json:"count"`
	URLs     []string     `json:"urls"`     // direct bookmark URLs at this exact folder path
	Children []FolderNode `json:"children"`
}

// ProfileTree groups a profile's folder tree with its metadata.
type ProfileTree struct {
	Browser      string       `json:"browser"`
	BrowserLabel string       `json:"browserLabel"`
	ProfileDir   string       `json:"profileDir"`
	DisplayName  string       `json:"displayName"`
	TotalCount   int          `json:"totalCount"`
	Roots        []FolderNode `json:"roots"`
}

// GetFolderTree builds a hierarchical tree showing each profile's OWN bookmarks
// (read directly from disk). This is separate from a.bookmarks (the unified deduped set)
// so the sidebar always shows all 9 profiles with their original folder structure.
func (a *App) GetFolderTree() []ProfileTree {
	browserLabels := map[string]string{
		"chrome": "Google Chrome",
		"edge":   "Microsoft Edge",
	}

	var result []ProfileTree
	for _, p := range a.profiles {
		if !p.HasBookmarks {
			continue
		}
		bms, err := browser.ReadBookmarks(p)
		if err != nil {
			bms = nil
		}

		pathItems := map[string][]string{}
		for _, bm := range bms {
			fp := bm.FolderPath
			if fp == "" {
				fp = "Uncategorized"
			}
			pathItems[fp] = append(pathItems[fp], bm.URL)
		}

		label := browserLabels[p.Browser]
		if label == "" {
			label = p.Browser
		}

		result = append(result, ProfileTree{
			Browser:      p.Browser,
			BrowserLabel: label,
			ProfileDir:   p.ProfileDir,
			DisplayName:  p.DisplayName,
			TotalCount:   len(bms),
			Roots:        buildTreeFromPaths(pathItems),
		})
	}
	return result
}

// buildTreeFromPaths converts a map of "A/B/C" path → []URL into a hierarchical tree.
func buildTreeFromPaths(pathItems map[string][]string) []FolderNode {
	type node struct {
		name     string
		urls     []string
		children map[string]*node
	}

	roots := map[string]*node{}

	for path, urls := range pathItems {
		parts := strings.Split(path, "/")
		current := roots
		for i, part := range parts {
			if _, ok := current[part]; !ok {
				current[part] = &node{name: part, children: map[string]*node{}}
			}
			if i == len(parts)-1 {
				current[part].urls = append(current[part].urls, urls...)
			}
			current = current[part].children
		}
	}

	var toSlice func(m map[string]*node, prefix string) []FolderNode
	toSlice = func(m map[string]*node, prefix string) []FolderNode {
		result := []FolderNode{}
		for _, n := range m {
			fullPath := n.name
			if prefix != "" {
				fullPath = prefix + "/" + n.name
			}
			children := toSlice(n.children, fullPath)
			totalCount := len(n.urls)
			for _, c := range children {
				totalCount += c.Count
			}
			result = append(result, FolderNode{
				Name:     n.name,
				Path:     fullPath,
				Count:    totalCount,
				URLs:     n.urls,
				Children: children,
			})
		}
		// Sort by name
		for i := 1; i < len(result); i++ {
			for j := i; j > 0 && result[j].Name < result[j-1].Name; j-- {
				result[j], result[j-1] = result[j-1], result[j]
			}
		}
		return result
	}

	return toSlice(roots, "")
}

// ── Collect ───────────────────────────────────────────

type CollectResult struct {
	TotalRaw        int `json:"totalRaw"`
	TotalDeduped    int `json:"totalDeduped"`
	DuplicatesURL   int `json:"duplicatesUrl"`
	DuplicatesFuzzy int `json:"duplicatesFuzzy"`
}

func (a *App) CollectBookmarks() (*CollectResult, error) {
	if len(a.profiles) == 0 {
		return nil, fmt.Errorf("no profiles scanned yet — click Scan first")
	}

	raw, err := browser.ReadAllBookmarks(a.profiles)
	if err != nil {
		return nil, err
	}

	// Deduplicate within each profile separately so every profile appears in the tree.
	// Cross-profile dedup would collapse all synced bookmarks into one profile.
	type profileKey struct{ browser, profileDir string }
	groups := map[profileKey][]browser.Bookmark{}
	order := []profileKey{}
	for _, bm := range raw {
		k := profileKey{bm.Browser, bm.ProfileDir}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		groups[k] = append(groups[k], bm)
	}

	var kept []browser.Bookmark
	var totalURLDupes, totalFuzzyDupes int
	for _, k := range order {
		deduped, ud, fd := engine.Deduplicate(groups[k])
		kept = append(kept, deduped...)
		totalURLDupes += ud
		totalFuzzyDupes += fd
	}

	// Final cross-profile dedup: same URL may appear in multiple profiles (e.g. synced bookmarks).
	// Keep the entry with the best category (non-empty wins) or the first occurrence.
	crossDeduped, crossURLDupes, _ := engine.Deduplicate(kept)
	totalURLDupes += crossURLDupes
	a.bookmarks = crossDeduped

	// Set category from folder path if not set
	for i := range a.bookmarks {
		if a.bookmarks[i].Category == "" && a.bookmarks[i].FolderPath != "" {
			a.bookmarks[i].Category = a.bookmarks[i].FolderPath
		}
	}

	return &CollectResult{
		TotalRaw:        len(raw),
		TotalDeduped:    len(crossDeduped),
		DuplicatesURL:   totalURLDupes,
		DuplicatesFuzzy: totalFuzzyDupes,
	}, nil
}

// ── Get / Set Bookmarks ───────────────────────────────

func (a *App) GetBookmarks() []browser.Bookmark {
	return a.bookmarks
}

func (a *App) SetBookmarks(bookmarks []browser.Bookmark) {
	a.bookmarks = bookmarks
}

func (a *App) GetBookmarkCount() int {
	return len(a.bookmarks)
}

// ── Analyze (AI) ──────────────────────────────────────

func (a *App) Analyze() (*engine.CategorizeResult, error) {
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks loaded — Collect first")
	}

	// Active prompt profile takes priority over cfg.SystemPrompt (ai-prompt.txt)
	cfg := *a.cfg
	if content := a.getActiveProfileContent(); content != "" {
		cfg.SystemPrompt = content
	}

	// Inject folder structure constraints from saved settings into the prompt.
	// These override / append to whatever the prompt profile says.
	fs := a.loadFolderSettings()
	constraint := fmt.Sprintf(
		"\n\n[FOLDER STRUCTURE RULES — OVERRIDE]\n"+
			"Max folder depth: %d levels.\n"+
			"Min bookmarks per folder: %d — if fewer, merge into parent.\n"+
			"Max bookmarks per folder: %d — if more, split into meaningful subcategories.",
		fs.MaxDepth, fs.MinFolderItems, fs.MaxFolderItems,
	)
	if fs.SmartRenamePrefix {
		constraint += "\n\n[SMART RENAME — OVERRIDE]\n" +
			"Prepend a short type prefix in square brackets to each bookmark title based on its URL and content type.\n" +
			"Use these standard prefixes (add more as needed):\n" +
			"[PBI] Power BI reports/dashboards (app.powerbi.com, powerbi.microsoft.com)\n" +
			"[SP] SharePoint documents/sites (sharepoint.com)\n" +
			"[TEAMS] Microsoft Teams (teams.microsoft.com)\n" +
			"[D365] Dynamics 365 (dynamics.com, crm.dynamics.com)\n" +
			"[AZ] Azure portal/services (portal.azure.com, azure.microsoft.com)\n" +
			"[DOC] Google Docs (docs.google.com/document)\n" +
			"[SHEET] Google Sheets (docs.google.com/spreadsheets)\n" +
			"[SLIDE] Google Slides (docs.google.com/presentation)\n" +
			"[GD] Google Drive (drive.google.com)\n" +
			"[GH] GitHub repos/issues (github.com)\n" +
			"[YT] YouTube videos (youtube.com)\n" +
			"[CF] Cloudflare (dash.cloudflare.com, cloudflare.com)\n" +
			"[NOTION] Notion pages (notion.so)\n" +
			"[FIGMA] Figma designs (figma.com)\n" +
			"[XLS] Excel/spreadsheet files (.xlsx, onedrive, excel online)\n" +
			"Keep the original title after the prefix. Example: '[PBI] CHI Dashboard - Power BI'"
	}
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt) + constraint

	result, err := engine.Categorize(a.ctx, a.bookmarks, &cfg, func(info engine.ProgressInfo) {
		wailsRuntime.EventsEmit(a.ctx, "analyze:progress", info)
	})
	if err != nil {
		return nil, err
	}

	a.bookmarks = result.Bookmarks
	applyFolderPostProcessing(a.bookmarks, fs)

	// ── Persist cache for LoadLastAnalysis ──────────────
	a.saveAnalysisCache(result)

	return result, nil
}

// AnalysisCacheMeta is returned by GetLastAnalysisMeta so the UI can show
// when the last analysis was done without loading the full dataset.
type AnalysisCacheMeta struct {
	Exists    bool   `json:"exists"`
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
	Model     string `json:"model"`
	Tokens    int    `json:"tokens"`
}

type analysisCacheFile struct {
	Meta      AnalysisCacheMeta  `json:"meta"`
	Bookmarks []browser.Bookmark `json:"bookmarks"`
	Profiles  []browser.Profile  `json:"profiles"`
}

func (a *App) cacheFilePath() string {
	return filepath.Join(a.cacheDir, "last-analysis.json")
}

func (a *App) saveAnalysisCache(result *engine.CategorizeResult) {
	os.MkdirAll(a.cacheDir, 0755)
	cache := analysisCacheFile{
		Meta: AnalysisCacheMeta{
			Exists:    true,
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			Count:     len(result.Bookmarks),
			Model:     result.Model,
			Tokens:    result.TotalTokens,
		},
		Bookmarks: result.Bookmarks,
		Profiles:  a.profiles,
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	os.WriteFile(a.cacheFilePath(), data, 0644)
}

// GetLastAnalysisMeta returns metadata about the cached analysis (no bookmarks).
func (a *App) GetLastAnalysisMeta() AnalysisCacheMeta {
	data, err := os.ReadFile(a.cacheFilePath())
	if err != nil {
		return AnalysisCacheMeta{Exists: false}
	}
	var cache analysisCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return AnalysisCacheMeta{Exists: false}
	}
	cache.Meta.Exists = true
	return cache.Meta
}

// LoadLastAnalysisResult is the return type for LoadLastAnalysis, including restored profiles.
type LoadLastAnalysisResult struct {
	Bookmarks   []browser.Bookmark `json:"bookmarks"`
	Profiles    []browser.Profile  `json:"profiles"`
	Model       string             `json:"model"`
	TotalTokens int                `json:"totalTokens"`
}

// LoadLastAnalysis loads the cached analysis result into memory and returns it.
func (a *App) LoadLastAnalysis() (*LoadLastAnalysisResult, error) {
	data, err := os.ReadFile(a.cacheFilePath())
	if err != nil {
		return nil, fmt.Errorf("no cached analysis found")
	}
	var cache analysisCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("corrupt cache file: %w", err)
	}
	if len(cache.Bookmarks) == 0 {
		return nil, fmt.Errorf("cache is empty")
	}
	a.bookmarks = cache.Bookmarks
	if len(cache.Profiles) > 0 {
		a.profiles = cache.Profiles
	}
	return &LoadLastAnalysisResult{
		Bookmarks:   cache.Bookmarks,
		Profiles:    cache.Profiles,
		Model:       cache.Meta.Model,
		TotalTokens: cache.Meta.Tokens,
	}, nil
}

// ── Sync ──────────────────────────────────────────────

// ProfileDiff shows what would change for a profile on sync
type ProfileDiff struct {
	Browser       string             `json:"browser"`
	BrowserLabel  string             `json:"browserLabel"`
	ProfileDir    string             `json:"profileDir"`
	DisplayName   string             `json:"displayName"`
	BeforeCount   int                `json:"beforeCount"`
	AfterCount    int                `json:"afterCount"`
	Added         int                `json:"added"`
	Removed       int                `json:"removed"`
	Unchanged     int                `json:"unchanged"`
	AddedSample   []browser.Bookmark `json:"addedSample"`
	RemovedSample []browser.Bookmark `json:"removedSample"`
}

type SyncPreviewResult struct {
	Diffs       []ProfileDiff `json:"diffs"`
	TotalBefore int           `json:"totalBefore"`
	TotalAfter  int           `json:"totalAfter"`
}

// GetSyncPreview compares current bookmarks with each profile's existing bookmarks
func (a *App) GetSyncPreview() (*SyncPreviewResult, error) {
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks loaded")
	}

	// Build URL set for the unified bookmarks
	afterURLs := make(map[string]bool, len(a.bookmarks))
	for _, b := range a.bookmarks {
		afterURLs[b.URL] = true
	}

	result := &SyncPreviewResult{
		TotalAfter: len(a.bookmarks),
	}

	for _, profile := range a.profiles {
		if !profile.HasBookmarks {
			continue
		}

		existing, err := browser.ReadBookmarks(profile)
		if err != nil {
			// If can't read, show as all-new
			result.Diffs = append(result.Diffs, ProfileDiff{
				Browser:      profile.Browser,
				BrowserLabel: profile.BrowserLabel,
				ProfileDir:   profile.ProfileDir,
				DisplayName:  profile.DisplayName,
				BeforeCount:  0,
				AfterCount:   len(a.bookmarks),
				Added:        len(a.bookmarks),
				Removed:      0,
				Unchanged:    0,
			})
			continue
		}

		// Build URL set for existing bookmarks
		beforeURLs := make(map[string]bool, len(existing))
		for _, b := range existing {
			beforeURLs[b.URL] = true
		}

		// Calculate diff
		added := 0
		removed := 0
		unchanged := 0

		for u := range afterURLs {
			if beforeURLs[u] {
				unchanged++
			} else {
				added++
			}
		}
		for u := range beforeURLs {
			if !afterURLs[u] {
				removed++
			}
		}

		// Collect samples (max 5 each)
		var addedSample []browser.Bookmark
		for _, b := range a.bookmarks {
			if !beforeURLs[b.URL] {
				addedSample = append(addedSample, b)
				if len(addedSample) >= 5 {
					break
				}
			}
		}

		var removedSample []browser.Bookmark
		for _, b := range existing {
			if !afterURLs[b.URL] {
				removedSample = append(removedSample, b)
				if len(removedSample) >= 5 {
					break
				}
			}
		}

		result.TotalBefore += len(existing)
		result.Diffs = append(result.Diffs, ProfileDiff{
			Browser:       profile.Browser,
			BrowserLabel:  profile.BrowserLabel,
			ProfileDir:    profile.ProfileDir,
			DisplayName:   profile.DisplayName,
			BeforeCount:   len(existing),
			AfterCount:    len(a.bookmarks),
			Added:         added,
			Removed:       removed,
			Unchanged:     unchanged,
			AddedSample:   addedSample,
			RemovedSample: removedSample,
		})
	}

	return result, nil
}

// BookmarkDiffRow is a single bookmark with its diff status for the Commander view
type BookmarkDiffRow struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	Domain     string `json:"domain"`
	FolderPath string `json:"folderPath"`
	Category   string `json:"category"`
	Status     string `json:"status"` // "added" | "removed" | "unchanged"
}

// ProfileDiffDetail contains full before/after lists for one profile
type ProfileDiffDetail struct {
	Browser      string            `json:"browser"`
	BrowserLabel string            `json:"browserLabel"`
	ProfileDir   string            `json:"profileDir"`
	DisplayName  string            `json:"displayName"`
	Before       []BookmarkDiffRow `json:"before"`
	After        []BookmarkDiffRow `json:"after"`
	Added        int               `json:"added"`
	Removed      int               `json:"removed"`
	Unchanged    int               `json:"unchanged"`
}

// enrichWithCachedCategories applies LLM categories from the saved cache to a.bookmarks,
// then flattens any leaf folders that contain fewer than minFolderItems bookmarks.
func (a *App) enrichWithCachedCategories() {
	data, err := os.ReadFile(a.cacheFilePath())
	if err != nil {
		return
	}
	var cache analysisCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return
	}
	catMap := make(map[string]string, len(cache.Bookmarks))
	for _, b := range cache.Bookmarks {
		if b.Category != "" {
			catMap[b.URL] = b.Category
		}
	}
	for i := range a.bookmarks {
		if cat, ok := catMap[a.bookmarks[i].URL]; ok {
			a.bookmarks[i].Category = cat
		}
	}
	fs := a.loadFolderSettings()
	applyFolderPostProcessing(a.bookmarks, fs)
}

// enforceMaxDepth truncates any bookmark Category that has more path segments than
// maxDepth. e.g. with maxDepth=2, "Work/MSFT/Teams/Channels" → "Work/MSFT".
// maxDepth <= 0 means no limit.
func enforceMaxDepth(bookmarks []browser.Bookmark, maxDepth int) {
	if maxDepth <= 0 {
		return
	}
	for i, b := range bookmarks {
		parts := strings.Split(b.Category, "/")
		if len(parts) > maxDepth {
			bookmarks[i].Category = strings.Join(parts[:maxDepth], "/")
		}
	}
}

// applyFolderPostProcessing runs the full folder-shaping pipeline in the correct order:
//  1. enforceMaxDepth — truncate paths beyond the depth limit (may consolidate many items)
//  2. splitFatFolders — split oversized folders into domain sub-groups
//  3. flattenThinFolders — merge thin folders back up (runs AFTER split to clean up domain
//     groups that end up with fewer than minItems entries)
//  4. applySmartPrefixes — optional rule-based title prefix pass
func applyFolderPostProcessing(bookmarks []browser.Bookmark, fs FolderSettings) {
	enforceMaxDepth(bookmarks, fs.MaxDepth)
	splitFatFolders(bookmarks, fs.MaxFolderItems)
	flattenThinFolders(bookmarks, fs.MinFolderItems)
	if fs.SmartRenamePrefix {
		applySmartPrefixes(bookmarks)
	}
}

// flattenThinFolders promotes bookmarks out of any leaf path that has fewer than
// minItems bookmarks — those items move up to the parent path.
// e.g. "01_WORK_MSFT/Learning_Dynamics365" with 1 item → "01_WORK_MSFT"
// Repeats until no further changes (handles cascading thin parents).
func flattenThinFolders(bookmarks []browser.Bookmark, minItems int) {
	for {
		// Count items per full category path
		counts := make(map[string]int)
		for _, b := range bookmarks {
			counts[b.Category]++
		}

		changed := false
		for i, b := range bookmarks {
			cat := b.Category
			sep := strings.LastIndex(cat, "/")
			if sep < 0 {
				continue // already at root level, cannot flatten further
			}
			if counts[cat] < minItems {
				bookmarks[i].Category = cat[:sep]
				changed = true
			}
		}
		if !changed {
			break
		}
	}
}

// splitFatFolders splits any folder that holds more than maxItems bookmarks into
// domain-based sub-groups (e.g. "01_WORK_MSFT/Reports" → "01_WORK_MSFT/Reports/PowerBI",
// "01_WORK_MSFT/Reports/SharePoint", …). Falls back to alphabetical A-G/H-Z buckets
// when domain grouping still leaves oversized groups.
func splitFatFolders(bookmarks []browser.Bookmark, maxItems int) {
	if maxItems <= 0 {
		return
	}
	// Collect indices per category
	idxBycat := make(map[string][]int)
	for i, b := range bookmarks {
		idxBycat[b.Category] = append(idxBycat[b.Category], i)
	}
	for cat, indices := range idxBycat {
		if len(indices) <= maxItems {
			continue
		}
		// Group by simplified domain label
		domainGroup := make(map[string][]int)
		for _, i := range indices {
			label := domainLabel(bookmarks[i].URL)
			domainGroup[label] = append(domainGroup[label], i)
		}
		// Only use domain grouping if it actually splits the folder meaningfully
		if len(domainGroup) > 1 {
			for label, idxs := range domainGroup {
				sub := cat + "/" + label
				for _, i := range idxs {
					bookmarks[i].Category = sub
				}
			}
			continue
		}
		// Fallback: split alphabetically by first letter of title into chunks
		for chunk := 0; chunk*maxItems < len(indices); chunk++ {
			start := chunk * maxItems
			end := start + maxItems
			if end > len(indices) {
				end = len(indices)
			}
			bucket := fmt.Sprintf("%s/Part_%02d", cat, chunk+1)
			for _, i := range indices[start:end] {
				bookmarks[i].Category = bucket
			}
		}
	}
}

// domainLabel returns a short capitalized label for a URL's domain,
// e.g. "powerbi.microsoft.com" → "PowerBI", "sharepoint.com" → "SharePoint".
func domainLabel(rawURL string) string {
	s := rawURL
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	// Strip www. and port
	s = strings.TrimPrefix(s, "www.")
	if i := strings.Index(s, ":"); i > 0 {
		s = s[:i]
	}
	// Use the second-level domain part as label (e.g. "powerbi" from "powerbi.microsoft.com")
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		// prefer the most specific subdomain that isn't generic
		for _, p := range parts {
			if p != "com" && p != "net" && p != "org" && p != "io" &&
				p != "microsoft" && p != "google" && p != "www" && len(p) > 2 {
				return strings.Title(strings.ToLower(p))
			}
		}
		return strings.Title(strings.ToLower(parts[0]))
	}
	if s == "" {
		return "Other"
	}
	return strings.Title(strings.ToLower(s))
}

// applySmartPrefixes adds a type prefix to bookmark titles that don't already have one,
// based on URL pattern matching. This is a best-effort rule-based pass that runs AFTER
// the AI has already renamed titles — it won't overwrite an existing [XXX] prefix.
func applySmartPrefixes(bookmarks []browser.Bookmark) {
	type rule struct {
		prefix  string
		matches []string // substrings matched against full URL (case-insensitive)
	}
	rules := []rule{
		{"[PBI]", []string{"app.powerbi.com", "powerbi.microsoft.com"}},
		{"[SP]", []string{"sharepoint.com"}},
		{"[TEAMS]", []string{"teams.microsoft.com"}},
		{"[D365]", []string{"dynamics.com", "crm.dynamics.com"}},
		{"[AZ]", []string{"portal.azure.com", "azure.microsoft.com"}},
		{"[DOC]", []string{"docs.google.com/document"}},
		{"[SHEET]", []string{"docs.google.com/spreadsheets", ".xlsx", "excel.office.com"}},
		{"[SLIDE]", []string{"docs.google.com/presentation"}},
		{"[GD]", []string{"drive.google.com"}},
		{"[GH]", []string{"github.com"}},
		{"[YT]", []string{"youtube.com", "youtu.be"}},
		{"[CF]", []string{"dash.cloudflare.com"}},
		{"[NOTION]", []string{"notion.so", "notion.com"}},
		{"[FIGMA]", []string{"figma.com"}},
		{"[LOOM]", []string{"loom.com"}},
	}

	for i, bm := range bookmarks {
		// Skip if title already has a bracket prefix like [PBI] or [DOC]
		if len(bm.Title) > 0 && bm.Title[0] == '[' {
			continue
		}
		urlLower := strings.ToLower(bm.URL)
		for _, r := range rules {
			for _, m := range r.matches {
				if strings.Contains(urlLower, m) {
					bookmarks[i].Title = r.prefix + " " + bm.Title
					goto nextBookmark
				}
			}
		}
	nextBookmark:
	}
}

// sortedBookmarks returns a copy of bookmarks sorted by Category then Title (A-Z).
// Used when SortAlphaInFolder is enabled so Chrome folder items appear alphabetically.
func sortedBookmarks(bookmarks []browser.Bookmark) []browser.Bookmark {
	sorted := make([]browser.Bookmark, len(bookmarks))
	copy(sorted, bookmarks)
	// Simple insertion sort — stable, in-place on the copy
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0; j-- {
			ki := sorted[j].Category + "\x00" + strings.ToLower(sorted[j].Title)
			kj := sorted[j-1].Category + "\x00" + strings.ToLower(sorted[j-1].Title)
			if ki < kj {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			} else {
				break
			}
		}
	}
	return sorted
}

// profileKeyOf returns the composite key "browser:profileDir" for a profile.
func profileKeyOf(browser, profileDir string) string {
	return browser + ":" + profileDir
}

// GetProfileDiffDetail returns the full before/after bookmark lists for one profile.
// profileKey is a composite "browser:profileDir" string (e.g. "chrome:Default").
func (a *App) GetProfileDiffDetail(profileKey string) (*ProfileDiffDetail, error) {
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks loaded — run collect + analyze first")
	}

	// Ensure LLM categories are applied even if only CollectBookmarks was run
	a.enrichWithCachedCategories()

	var target *browser.Profile
	for _, p := range a.profiles {
		if profileKeyOf(p.Browser, p.ProfileDir) == profileKey {
			pp := p
			target = &pp
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("profile not found: %s", profileKey)
	}

	// Build URL set for unified (after) bookmarks
	afterURLSet := make(map[string]bool, len(a.bookmarks))
	for _, b := range a.bookmarks {
		afterURLSet[b.URL] = true
	}

	// Read current profile bookmarks
	existing, err := browser.ReadBookmarks(*target)
	if err != nil {
		existing = nil
	}

	// Build URL set for before
	beforeURLSet := make(map[string]bool, len(existing))
	for _, b := range existing {
		beforeURLSet[b.URL] = true
	}

	domainOf := func(rawURL string) string {
		s := rawURL
		if i := strings.Index(s, "://"); i >= 0 {
			s = s[i+3:]
		}
		if i := strings.Index(s, "/"); i >= 0 {
			s = s[:i]
		}
		if strings.HasPrefix(s, "www.") {
			s = s[4:]
		}
		return s
	}

	// Build BEFORE list (current profile's bookmarks + their status)
	before := make([]BookmarkDiffRow, 0, len(existing))
	for _, b := range existing {
		status := "unchanged"
		if !afterURLSet[b.URL] {
			status = "removed"
		}
		before = append(before, BookmarkDiffRow{
			Title:      b.Title,
			URL:        b.URL,
			Domain:     domainOf(b.URL),
			FolderPath: b.FolderPath,
			Status:     status,
		})
	}

	// Build AFTER list (unified bookmarks + their status for this profile).
	// Deduplicate by URL across profiles — the same URL in multiple profiles
	// should appear only once in the unified target set.
	afterSeen := make(map[string]bool, len(a.bookmarks))
	after := make([]BookmarkDiffRow, 0)
	for _, b := range a.bookmarks {
		if afterSeen[b.URL] {
			continue
		}
		afterSeen[b.URL] = true
		status := "unchanged"
		if !beforeURLSet[b.URL] {
			status = "added"
		}
		after = append(after, BookmarkDiffRow{
			Title:      b.Title,
			URL:        b.URL,
			Domain:     domainOf(b.URL),
			FolderPath: b.FolderPath,
			Category:   b.Category,
			Status:     status,
		})
	}

	added, removed, unchanged := 0, 0, 0
	for _, r := range before {
		if r.Status == "removed" {
			removed++
		} else {
			unchanged++
		}
	}
	for _, r := range after {
		if r.Status == "added" {
			added++
		}
	}

	return &ProfileDiffDetail{
		Browser:      target.Browser,
		BrowserLabel: target.BrowserLabel,
		ProfileDir:   target.ProfileDir,
		DisplayName:  target.DisplayName,
		Before:       before,
		After:        after,
		Added:        added,
		Removed:      removed,
		Unchanged:    unchanged,
	}, nil
}

// SyncToSelectedProfiles syncs only to the profiles whose profileDir is in the provided list.
func (a *App) SyncToSelectedProfiles(profileDirs []string) (*SyncResponse, error) {
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks to sync")
	}

	bms := a.bookmarks
	if fs := a.loadFolderSettings(); fs.SortAlphaInFolder {
		bms = sortedBookmarks(bms)
	}

	// profileDirs contains composite "browser:profileDir" keys
	selected := make(map[string]bool, len(profileDirs))
	for _, d := range profileDirs {
		selected[d] = true
	}

	var results []*bsync.WriteResult
	var backups []*bsync.Snapshot

	for _, profile := range a.profiles {
		if !profile.HasBookmarks || !selected[profileKeyOf(profile.Browser, profile.ProfileDir)] {
			continue
		}
		snap, err := bsync.BackupBeforeSync(profile, a.backupDir)
		if err != nil {
			results = append(results, &bsync.WriteResult{
				Browser: profile.BrowserLabel,
				Profile: profile.DisplayName,
				Status:  "error",
				Reason:  fmt.Sprintf("backup failed: %s", err),
			})
			continue
		}
		backups = append(backups, snap)

		res, err := bsync.SyncToProfile(bms, profile)
		if err != nil {
			results = append(results, &bsync.WriteResult{
				Browser: profile.BrowserLabel,
				Profile: profile.DisplayName,
				Status:  "error",
				Reason:  err.Error(),
			})
			continue
		}
		results = append(results, res)
	}

	return &SyncResponse{Results: results, Backups: backups}, nil
}

type SyncResponse struct {
	Results []*bsync.WriteResult `json:"results"`
	Backups []*bsync.Snapshot    `json:"backups"`
}

func (a *App) SyncToAllProfiles() (*SyncResponse, error) {
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks to sync")
	}

	var results []*bsync.WriteResult
	var backups []*bsync.Snapshot

	for _, profile := range a.profiles {
		if !profile.HasBookmarks {
			continue
		}

		// Create backup before sync
		snap, err := bsync.BackupBeforeSync(profile, a.backupDir)
		if err != nil {
			results = append(results, &bsync.WriteResult{
				Browser: profile.BrowserLabel,
				Profile: profile.DisplayName,
				Status:  "error",
				Reason:  fmt.Sprintf("backup failed: %s", err),
			})
			continue
		}
		backups = append(backups, snap)

		// Clean sync: REPLACE all bookmarks (not append!)
		res, err := bsync.SyncToProfile(a.bookmarks, profile)
		if err != nil {
			results = append(results, &bsync.WriteResult{
				Browser: profile.BrowserLabel,
				Profile: profile.DisplayName,
				Status:  "error",
				Reason:  err.Error(),
			})
			continue
		}
		results = append(results, res)
	}

	return &SyncResponse{Results: results, Backups: backups}, nil
}

func (a *App) SyncToProfile(profileIndex int) (*SyncResponse, error) {
	if profileIndex < 0 || profileIndex >= len(a.profiles) {
		return nil, fmt.Errorf("invalid profile index: %d", profileIndex)
	}
	if len(a.bookmarks) == 0 {
		return nil, fmt.Errorf("no bookmarks to sync")
	}

	profile := a.profiles[profileIndex]

	snap, err := bsync.BackupBeforeSync(profile, a.backupDir)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	res, err := bsync.SyncToProfile(a.bookmarks, profile)
	if err != nil {
		return nil, err
	}

	return &SyncResponse{
		Results: []*bsync.WriteResult{res},
		Backups: []*bsync.Snapshot{snap},
	}, nil
}

// ── Backup / Restore ──────────────────────────────────

func (a *App) ListSnapshots() ([]bsync.Snapshot, error) {
	return bsync.ListSnapshots(a.backupDir)
}

func (a *App) RestoreSnapshot(snapshotID string, profileIndex int) error {
	if profileIndex < 0 || profileIndex >= len(a.profiles) {
		return fmt.Errorf("invalid profile index")
	}

	snapshots, err := bsync.ListSnapshots(a.backupDir)
	if err != nil {
		return err
	}

	var targetSnap *bsync.Snapshot
	for _, s := range snapshots {
		if s.ID == snapshotID {
			targetSnap = &s
			break
		}
	}
	if targetSnap == nil {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	return bsync.RestoreSnapshot(targetSnap.FilePath, a.profiles[profileIndex])
}

func (a *App) DeleteSnapshot(snapshotID string) error {
	snapshots, err := bsync.ListSnapshots(a.backupDir)
	if err != nil {
		return err
	}
	for _, s := range snapshots {
		if s.ID == snapshotID {
			return bsync.DeleteSnapshot(s.FilePath)
		}
	}
	return fmt.Errorf("snapshot not found")
}

// ForceBackupAll immediately backups all scanned profiles without waiting for sync.
func (a *App) ForceBackupAll() ([]bsync.Snapshot, error) {
	if len(a.profiles) == 0 {
		return nil, fmt.Errorf("no profiles scanned yet — click Scan first")
	}
	var snaps []bsync.Snapshot
	var lastErr error
	for _, p := range a.profiles {
		if !p.HasBookmarks {
			continue
		}
		snap, err := bsync.BackupBeforeSync(p, a.backupDir)
		if err != nil {
			lastErr = err
			continue
		}
		snaps = append(snaps, *snap)
	}
	if len(snaps) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return snaps, nil
}

// ── Export ─────────────────────────────────────────────

func (a *App) ExportBookmarks() (string, error) {
	if len(a.bookmarks) == 0 {
		return "", fmt.Errorf("no bookmarks to export")
	}
	data, err := json.MarshalIndent(a.bookmarks, "", "  ")
	if err != nil {
		return "", err
	}

	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Bookmarks",
		DefaultFilename: "bookmarks.json",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// ── Config ────────────────────────────────────────────

type AppConfig struct {
	Model     string `json:"model"`
	BatchSize int    `json:"batchSize"`
	HasAPIKey bool   `json:"hasApiKey"`
	BackupDir string `json:"backupDir"`
}

func (a *App) GetConfig() AppConfig {
	return AppConfig{
		Model:     a.cfg.OpenAIModel,
		BatchSize: a.cfg.BatchSize,
		HasAPIKey: a.cfg.OpenAIKey != "",
		BackupDir: a.backupDir,
	}
}

// ── AI Config ────────────────────────────────────────

type AIConfig struct {
	Endpoint     string `json:"endpoint"`
	APIKey       string `json:"apiKey"`
	Model        string `json:"model"`
	SystemPrompt string `json:"systemPrompt"`
}

func (a *App) GetAIConfig() AIConfig {
	return AIConfig{
		Endpoint:     a.cfg.OpenAIBase,
		APIKey:       a.cfg.OpenAIKey,
		Model:        a.cfg.OpenAIModel,
		SystemPrompt: a.cfg.SystemPrompt,
	}
}

func (a *App) SaveAIConfig(cfg AIConfig) error {
	// Update in-memory config
	a.cfg.OpenAIBase = strings.TrimSpace(cfg.Endpoint)
	a.cfg.OpenAIKey = strings.TrimSpace(cfg.APIKey)
	a.cfg.OpenAIModel = strings.TrimSpace(cfg.Model)
	a.cfg.SystemPrompt = cfg.SystemPrompt

	// Save custom prompt to ai-prompt.txt (handles multiline)
	exe, _ := os.Executable()
	promptPath := filepath.Join(filepath.Dir(exe), "ai-prompt.txt")
	prompt := strings.TrimSpace(cfg.SystemPrompt)
	if prompt == "" {
		os.Remove(promptPath)
	} else {
		os.WriteFile(promptPath, []byte(prompt), 0644)
	}

	// Also set env vars so engine picks them up
	os.Setenv("OPENAI_BASE_URL", a.cfg.OpenAIBase)
	os.Setenv("OPENAI_API_KEY", a.cfg.OpenAIKey)
	os.Setenv("OPENAI_MODEL", a.cfg.OpenAIModel)

	// Persist to .env file next to the executable
	envPath := filepath.Join(filepath.Dir(exe), ".env")
	return writeEnvFile(envPath, a.cfg)
}

// ── Prompt Profiles ──────────────────────────────────────

type PromptProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	IsBuiltin   bool   `json:"isBuiltin"`
}

type promptProfilesFile struct {
	ActiveID string          `json:"activeId"`
	Custom   []PromptProfile `json:"custom"`
}

var builtinProfiles = []PromptProfile{
	{
		ID: "default", Name: "Default (PARA method)",
		Description: "Built-in PARA classifier. No extra instructions.",
		Content:     "", IsBuiltin: true,
	},
	{
		ID: "flat-10", Name: "Flat — max 10 categories",
		Description: "Broad top-level folders only, max 2 levels deep.",
		Content:     "Organize into AT MOST 10 top-level categories. Use broad names: Dev, Design, Learning, News, Work, Tools, Finance, Health, Entertainment, Other. Never go deeper than 2 levels. Merge niche topics into the nearest broad category.",
		IsBuiltin:   true,
	},
	{
		ID: "pkm-vi", Name: "PKM Vietnamese — 5 root folders",
		Description: "Theo ngữ cảnh MSFT, Inno Mountain, Freelance, Tech Stack, Life.",
		Content:     "[CẤU TRÚC]\nPhân loại vào đúng 5 thư mục gốc:\n01_WORK_MSFT — Microsoft, Dynamics 365, LCS, Azure DevOps, quản lý team.\n02_INNO_MOUNTAIN — Sản phẩm AI (DocuAI, InnoTranslator), Astro, Cloudflare, Proxmox, Linux.\n03_FREELANCE_CLIENTS — Phân theo tên khách hàng. Nếu không rõ → _Unsorted_Freelance.\n04_TECH_STACK — Tài liệu kỹ thuật dùng chung: Frontend, Backend, Infrastructure, AI.\n05_LIFE_HEALTH — Sức khỏe, ăn kiêng, thể thao, sở thích cá nhân.\n\n[QUY TẮC]\nMax 2 sub-levels. Smart naming: [Website] - [Mô tả ngắn]. Đánh dấu localhost/sandbox là archive.",
		IsBuiltin:   true,
	},
	{
		ID: "tech-dev", Name: "Tech Developer Focus",
		Description: "Grouped by dev stack: Frontend, Backend, DevOps, AI/ML, Learning.",
		Content:     "Organize technical bookmarks by stack: Frontend (HTML/CSS/JS/React/Vue), Backend (Node/Python/Go/API), DevOps (Docker/K8s/CI/Cloud), AI-ML (models/tools/papers), Mobile (iOS/Android/RN/Flutter), Learning (docs/tutorials/courses), Tools (utilities/extensions), and Other for non-tech. Max 2 levels.",
		IsBuiltin:   true,
	},
}

func (a *App) promptProfilesPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "prompt-profiles.json")
}

func (a *App) loadPromptStore() promptProfilesFile {
	var pf promptProfilesFile
	if data, err := os.ReadFile(a.promptProfilesPath()); err == nil {
		json.Unmarshal(data, &pf)
	}
	if pf.ActiveID == "" {
		pf.ActiveID = "default"
	}
	return pf
}

func (a *App) savePromptStore(pf promptProfilesFile) error {
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.promptProfilesPath(), data, 0644)
}

func (a *App) getActiveProfileContent() string {
	pf := a.loadPromptStore()
	for _, p := range builtinProfiles {
		if p.ID == pf.ActiveID {
			return p.Content
		}
	}
	for _, p := range pf.Custom {
		if p.ID == pf.ActiveID {
			return p.Content
		}
	}
	return ""
}

func (a *App) GetPromptProfiles() map[string]interface{} {
	pf := a.loadPromptStore()
	all := make([]PromptProfile, 0, len(builtinProfiles)+len(pf.Custom))
	all = append(all, builtinProfiles...)
	all = append(all, pf.Custom...)
	return map[string]interface{}{"profiles": all, "activeId": pf.ActiveID}
}

func (a *App) SaveCustomProfiles(custom []PromptProfile) error {
	pf := a.loadPromptStore()
	// Only keep non-builtin profiles
	var filtered []PromptProfile
	for _, p := range custom {
		if !p.IsBuiltin {
			filtered = append(filtered, p)
		}
	}
	pf.Custom = filtered
	return a.savePromptStore(pf)
}

func (a *App) SetActivePromptID(id string) error {
	pf := a.loadPromptStore()
	pf.ActiveID = id
	return a.savePromptStore(pf)
}

// ── Folder Structure Settings ─────────────────────────────

// FolderSettings controls the folder hierarchy injected into the AI prompt
// and the post-processing flattening/splitting/renaming passes.
type FolderSettings struct {
	MaxDepth          int  `json:"maxDepth"`          // max folder nesting levels (e.g. 3)
	MinFolderItems    int  `json:"minFolderItems"`    // folders with fewer items get merged up (e.g. 3)
	MaxFolderItems    int  `json:"maxFolderItems"`    // folders with more items get split into sub-groups (e.g. 30)
	SmartRenamePrefix bool `json:"smartRenamePrefix"` // prepend type prefix [PBI], [DOC] etc. to bookmark titles
	SortAlphaInFolder bool `json:"sortAlphaInFolder"` // sort bookmarks A-Z within each folder on sync
}

func (a *App) folderSettingsPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "folder-settings.json")
}

func (a *App) loadFolderSettings() FolderSettings {
	fs := FolderSettings{MaxDepth: 3, MinFolderItems: 3, MaxFolderItems: 30} // defaults
	if data, err := os.ReadFile(a.folderSettingsPath()); err == nil {
		json.Unmarshal(data, &fs)
	}
	if fs.MaxDepth < 1 { fs.MaxDepth = 1 }
	if fs.MinFolderItems < 1 { fs.MinFolderItems = 1 }
	if fs.MaxFolderItems < 1 { fs.MaxFolderItems = 999 }
	return fs
}

func (a *App) GetFolderSettings() FolderSettings {
	return a.loadFolderSettings()
}

func (a *App) SaveFolderSettings(fs FolderSettings) error {
	if fs.MaxDepth < 1 {
		fs.MaxDepth = 1
	}
	if fs.MinFolderItems < 1 {
		fs.MinFolderItems = 1
	}
	if fs.MaxFolderItems < 1 {
		fs.MaxFolderItems = 999
	}
	data, err := json.MarshalIndent(fs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.folderSettingsPath(), data, 0644)
}

// ── Test AI ───────────────────────────────────────────────

type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Model   string `json:"model"`
}

func (a *App) TestAIConnection(cfg AIConfig) TestResult {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	apiKey := strings.TrimSpace(cfg.APIKey)
	model := strings.TrimSpace(cfg.Model)

	if endpoint == "" || apiKey == "" || model == "" {
		return TestResult{OK: false, Message: "endpoint, api key and model are required"}
	}

	url := strings.TrimRight(endpoint, "/") + "/chat/completions"
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "Say OK"},
		},
		"max_completion_tokens": 5,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return TestResult{OK: false, Message: "invalid endpoint: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("api-key", apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return TestResult{OK: false, Message: "connection failed: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if e, ok := errBody["error"]; ok {
			if em, ok := e.(map[string]interface{}); ok {
				if m, ok := em["message"].(string); ok {
					msg += ": " + m
				}
			}
		}
		return TestResult{OK: false, Message: msg}
	}

	return TestResult{OK: true, Message: "connection successful", Model: model}
}

// writeEnvFile persists AI config to .env
func writeEnvFile(path string, cfg *config.Config) error {
	// Read existing .env to preserve non-AI keys
	existing := make(map[string]string)
	var orderedKeys []string

	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				existing[key] = val
				orderedKeys = append(orderedKeys, key)
			}
		}
	}

	// Update AI keys
	aiKeys := map[string]string{
		"OPENAI_BASE_URL": cfg.OpenAIBase,
		"OPENAI_API_KEY":  cfg.OpenAIKey,
		"OPENAI_MODEL":    cfg.OpenAIModel,
	}
	for k, v := range aiKeys {
		if _, ok := existing[k]; !ok {
			orderedKeys = append(orderedKeys, k)
		}
		existing[k] = v
	}

	var buf strings.Builder
	for _, k := range orderedKeys {
		buf.WriteString(k + "=" + existing[k] + "\n")
	}

	return os.WriteFile(path, []byte(buf.String()), 0600)
}
