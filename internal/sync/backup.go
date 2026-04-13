package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"UnifiedBookmarks-Desktop/internal/browser"
)

// Snapshot represents a saved backup snapshot
type Snapshot struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Browser   string `json:"browser"`
	Profile   string `json:"profile"`
	Count     int    `json:"count"`
	FilePath  string `json:"filePath"`
	SizeBytes int64  `json:"sizeBytes"`
}

// SnapshotDetail includes the actual bookmarks
type SnapshotDetail struct {
	Snapshot
	Bookmarks []browser.Bookmark `json:"bookmarks"`
}

const maxBackups = 20

// BackupBeforeSync creates a timestamped backup of the bookmarks file
func BackupBeforeSync(profile browser.Profile, backupDir string) (*Snapshot, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create backup dir: %w", err)
	}

	// Read current bookmarks to get count
	data, err := os.ReadFile(profile.BookmarksPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read bookmarks file: %w", err)
	}

	// Count bookmarks
	var parsed struct {
		Roots map[string]json.RawMessage `json:"roots"`
	}
	bmCount := 0
	if json.Unmarshal(data, &parsed) == nil {
		for _, raw := range parsed.Roots {
			bmCount += countBookmarksInJSON(raw)
		}
	}

	now := time.Now()
	ts := now.Format("20060102_150405")
	safeName := fmt.Sprintf("%s_%s_%s_%s.json",
		profile.Browser, profile.ProfileDir,
		sanitize(profile.DisplayName), ts)
	destPath := filepath.Join(backupDir, safeName)

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, fmt.Errorf("cannot write backup: %w", err)
	}

	// Also create in-place .bak
	bakPath := profile.BookmarksPath + ".bak"
	os.WriteFile(bakPath, data, 0644)

	// Rotate
	rotateBackups(backupDir, profile.Browser, profile.ProfileDir)

	info, _ := os.Stat(destPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	return &Snapshot{
		ID:        ts,
		Timestamp: now.Format("2006-01-02 15:04:05"),
		Browser:   profile.Browser,     // lowercase key (chrome/edge) for consistency with ListSnapshots
		Profile:   profile.DisplayName, // e.g. "Minh C Ng"
		Count:     bmCount,
		FilePath:  destPath,
		SizeBytes: size,
	}, nil
}

// ListSnapshots returns all available backup snapshots
func ListSnapshots(backupDir string) ([]Snapshot, error) {
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []Snapshot{}, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var snapshots []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := entry.Name()
		info, _ := entry.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}

		// Parse filename: browser_profileDir_sanitizedDisplayName_YYYYMMDD_HHMMSS.json
		// Timestamp is ALWAYS the last 15 chars (YYYYMMDD_HHMMSS). Parse from the right
		// to avoid ambiguity: sanitize() replaces spaces/special chars with '_', so
		// SplitN can't reliably find the timestamp by splitting from the left.
		nameNoExt := strings.TrimSuffix(name, ".json")
		browserName := "unknown"
		profileName := name // fallback: raw filename
		tsFormatted := ""

		const tsLen = 15 // "20060102_150405"
		if len(nameNoExt) > tsLen+1 {
			if t, err := time.Parse("20060102_150405", nameNoExt[len(nameNoExt)-tsLen:]); err == nil {
				tsFormatted = t.Format("2006-01-02 15:04:05")
				// strip "_YYYYMMDD_HHMMSS" from right
				rest := nameNoExt[:len(nameNoExt)-tsLen-1]
				// rest = "browser_profileDir[_sanitizedDisplayName]"
				if idx := strings.Index(rest, "_"); idx > 0 {
					browserName = rest[:idx]
					afterBrowser := rest[idx+1:]
					// Chrome profileDir may have spaces ("Profile 1") but no underscores.
					// sanitizedDisplayName uses underscores for spaces.
					// So the first "_" in afterBrowser separates profileDir from display name.
					if idx2 := strings.Index(afterBrowser, "_"); idx2 > 0 {
						profileDir := afterBrowser[:idx2]
						display := strings.TrimSpace(strings.ReplaceAll(afterBrowser[idx2+1:], "_", " "))
						if display != "" {
							profileName = display // prefer reconstructed display name
						} else {
							profileName = profileDir
						}
					} else {
						profileName = afterBrowser
					}
				}
			}
		}

		// Count bookmarks in backup
		fullPath := filepath.Join(backupDir, name)
		count := countBookmarksInFile(fullPath)

		snapshots = append(snapshots, Snapshot{
			ID:        name,
			Timestamp: tsFormatted,
			Browser:   browserName,
			Profile:   profileName,
			Count:     count,
			FilePath:  fullPath,
			SizeBytes: size,
		})
	}

	// Sort by newest first
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp > snapshots[j].Timestamp
	})

	return snapshots, nil
}

// RestoreSnapshot restores a backup to a specific browser profile
func RestoreSnapshot(snapshotPath string, targetProfile browser.Profile) error {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("cannot read snapshot: %w", err)
	}

	// Validate it's valid JSON
	var test map[string]interface{}
	if err := json.Unmarshal(data, &test); err != nil {
		return fmt.Errorf("invalid bookmark JSON: %w", err)
	}

	// Atomic write
	dir := filepath.Dir(targetProfile.BookmarksPath)
	tmpFile := filepath.Join(dir, "Bookmarks.tmp.restore")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := os.Rename(tmpFile, targetProfile.BookmarksPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("cannot rename: %w", err)
	}

	return nil
}

// DeleteSnapshot removes a backup file
func DeleteSnapshot(snapshotPath string) error {
	return os.Remove(snapshotPath)
}

func countBookmarksInFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var parsed struct {
		Roots map[string]json.RawMessage `json:"roots"`
	}
	if json.Unmarshal(data, &parsed) != nil {
		return 0
	}
	count := 0
	for _, raw := range parsed.Roots {
		count += countBookmarksInJSON(raw)
	}
	return count
}

func countBookmarksInJSON(raw json.RawMessage) int {
	var node struct {
		Type     string            `json:"type"`
		Children []json.RawMessage `json:"children"`
	}
	if json.Unmarshal(raw, &node) != nil {
		return 0
	}
	if node.Type == "url" {
		return 1
	}
	count := 0
	for _, child := range node.Children {
		count += countBookmarksInJSON(child)
	}
	return count
}

func rotateBackups(backupDir, browserKey, profileDir string) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	prefix := browserKey + "_" + profileDir + "_"
	var matches []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".json") {
			matches = append(matches, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	if len(matches) <= maxBackups {
		return
	}

	for _, old := range matches[maxBackups:] {
		os.Remove(filepath.Join(backupDir, old))
	}
}

func sanitize(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// IsBrowserRunning checks if Chrome/Edge lock files exist
func IsBrowserRunning(userDataDir string) bool {
	locks := []string{"lockfile", "SingletonLock", "SingletonCookie"}
	for _, lf := range locks {
		if _, err := os.Stat(filepath.Join(userDataDir, lf)); err == nil {
			return true
		}
	}
	return false
}
