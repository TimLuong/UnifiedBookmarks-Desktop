package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Bookmark represents a single parsed bookmark
type Bookmark struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	FolderPath  string   `json:"folderPath"`
	RootSection string   `json:"rootSection"`
	Browser     string   `json:"browser"`
	ProfileDir  string   `json:"profileDir"`
	DisplayName string   `json:"displayName"`
	DateAdded   string   `json:"dateAdded"`
	Category    string   `json:"category"`
	Confidence  float64  `json:"confidence"`
	ParaType    string   `json:"paraType"`
	ParaContext string   `json:"paraContext"`
	Tags        []string `json:"tags"`
}

// Profile represents a discovered browser profile
type Profile struct {
	Browser       string `json:"browser"`
	BrowserLabel  string `json:"browserLabel"`
	ProfileDir    string `json:"profileDir"`
	DisplayName   string `json:"displayName"`
	BookmarksPath string `json:"bookmarksPath"`
	HasBookmarks  bool   `json:"hasBookmarks"`
	UserDataDir   string `json:"userDataDir"`
}

type browserConfig struct {
	Label       string
	UserDataDir string
}

func getBrowserConfigs() map[string]browserConfig {
	if runtime.GOOS != "windows" {
		return map[string]browserConfig{}
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	return map[string]browserConfig{
		"chrome": {
			Label:       "Google Chrome",
			UserDataDir: filepath.Join(localAppData, "Google", "Chrome", "User Data"),
		},
		"edge": {
			Label:       "Microsoft Edge",
			UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge", "User Data"),
		},
	}
}

// ScanProfiles discovers all browser profiles with bookmarks
func ScanProfiles() ([]Profile, error) {
	configs := getBrowserConfigs()
	var profiles []Profile

	for key, cfg := range configs {
		found, err := scanBrowser(key, cfg)
		if err != nil {
			continue
		}
		profiles = append(profiles, found...)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Browser != profiles[j].Browser {
			return profiles[i].Browser < profiles[j].Browser
		}
		return profiles[i].ProfileDir < profiles[j].ProfileDir
	})

	return profiles, nil
}

func scanBrowser(key string, cfg browserConfig) ([]Profile, error) {
	if _, err := os.Stat(cfg.UserDataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s not found", cfg.Label)
	}

	entries, err := os.ReadDir(cfg.UserDataDir)
	if err != nil {
		return nil, err
	}

	localState := readLocalState(cfg.UserDataDir)
	var profiles []Profile

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isProfileDir(name) {
			continue
		}

		bmPath := filepath.Join(cfg.UserDataDir, name, "Bookmarks")
		exists := fileExists(bmPath)

		displayName := getDisplayName(localState, name)

		profiles = append(profiles, Profile{
			Browser:       key,
			BrowserLabel:  cfg.Label,
			ProfileDir:    name,
			DisplayName:   displayName,
			BookmarksPath: bmPath,
			HasBookmarks:  exists,
			UserDataDir:   cfg.UserDataDir,
		})
	}

	return profiles, nil
}

func isProfileDir(name string) bool {
	if name == "Default" {
		return true
	}
	if len(name) > 8 && name[:8] == "Profile " {
		for _, c := range name[8:] {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

func readLocalState(userDataDir string) map[string]interface{} {
	data, err := os.ReadFile(filepath.Join(userDataDir, "Local State"))
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

func getDisplayName(localState map[string]interface{}, profileDir string) string {
	if localState == nil {
		return profileDir
	}
	profile, ok := localState["profile"].(map[string]interface{})
	if !ok {
		return profileDir
	}
	infoCache, ok := profile["info_cache"].(map[string]interface{})
	if !ok {
		return profileDir
	}
	info, ok := infoCache[profileDir].(map[string]interface{})
	if !ok {
		return profileDir
	}
	if name, ok := info["name"].(string); ok && name != "" {
		return name
	}
	return profileDir
}

// ReadBookmarks reads and parses all bookmarks from a profile
func ReadBookmarks(profile Profile) ([]Bookmark, error) {
	data, err := os.ReadFile(profile.BookmarksPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read bookmarks file: %w", err)
	}

	var root struct {
		Roots map[string]json.RawMessage `json:"roots"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	rootLabels := map[string]string{
		"bookmark_bar": "Bookmark Bar",
		"other":        "Other Bookmarks",
		"synced":       "Mobile Bookmarks",
	}

	var bookmarks []Bookmark
	for section, raw := range root.Roots {
		var node chromeNode
		if err := json.Unmarshal(raw, &node); err != nil {
			continue
		}
		label := rootLabels[section]
		if label == "" {
			label = section
		}
		walkNode(&node, "", section, profile, &bookmarks)
	}

	return bookmarks, nil
}

type chromeNode struct {
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	URL       string       `json:"url"`
	ID        string       `json:"id"`
	DateAdded string       `json:"date_added"`
	Children  []chromeNode `json:"children"`
}

func walkNode(node *chromeNode, folderPath, rootSection string, profile Profile, acc *[]Bookmark) {
	if node == nil {
		return
	}

	if node.Type == "url" {
		if node.URL == "" || strings.HasPrefix(node.URL, "javascript:") {
			return
		}
		title := strings.TrimSpace(node.Name)
		if title == "" {
			title = node.URL
		}
		*acc = append(*acc, Bookmark{
			ID:          node.ID,
			Title:       title,
			URL:         node.URL,
			FolderPath:  folderPath,
			RootSection: rootSection,
			Browser:     profile.Browser,
			ProfileDir:  profile.ProfileDir,
			DisplayName: profile.DisplayName,
			DateAdded:   node.DateAdded,
			Tags:        []string{},
		})
	} else if node.Type == "folder" {
		nextPath := folderPath
		if folderPath != "" {
			nextPath = folderPath + "/" + node.Name
		} else {
			nextPath = node.Name
		}
		for i := range node.Children {
			walkNode(&node.Children[i], nextPath, rootSection, profile, acc)
		}
	}
}

// ReadAllBookmarks reads from all profiles
func ReadAllBookmarks(profiles []Profile) ([]Bookmark, error) {
	var all []Bookmark
	for _, p := range profiles {
		if !p.HasBookmarks {
			continue
		}
		bms, err := ReadBookmarks(p)
		if err != nil {
			continue
		}
		all = append(all, bms...)
	}
	return all, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
