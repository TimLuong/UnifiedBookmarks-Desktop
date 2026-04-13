package engine

import (
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"UnifiedBookmarks-Desktop/internal/browser"
)

// Deduplicate removes duplicate bookmarks (URL normalization + fuzzy title matching)
func Deduplicate(bookmarks []browser.Bookmark) (kept []browser.Bookmark, urlDupes, fuzzyDupes int) {
	type entry struct {
		bm    browser.Bookmark
		normU string
	}

	var entries []entry
	for _, bm := range bookmarks {
		entries = append(entries, entry{bm: bm, normU: normalizeURL(bm.URL)})
	}

	seen := map[string]int{} // normURL -> index in kept
	for _, e := range entries {
		if idx, ok := seen[e.normU]; ok {
			// Keep earlier dateAdded
			if e.bm.DateAdded != "" && (kept[idx].DateAdded == "" || e.bm.DateAdded < kept[idx].DateAdded) {
				kept[idx] = e.bm
			}
			urlDupes++
			continue
		}
		seen[e.normU] = len(kept)
		kept = append(kept, e.bm)
	}

	// Fuzzy title matching on same domain
	var final []browser.Bookmark
	removed := map[int]bool{}
	for i := 0; i < len(kept); i++ {
		if removed[i] {
			continue
		}
		for j := i + 1; j < len(kept); j++ {
			if removed[j] {
				continue
			}
			if extractDomain(kept[i].URL) != extractDomain(kept[j].URL) {
				continue
			}
			t1, t2 := kept[i].Title, kept[j].Title
			if utf8.RuneCountInString(t1) < 8 || utf8.RuneCountInString(t2) < 8 {
				continue
			}
			if levenshteinRatio(t1, t2) > 0.85 {
				removed[j] = true
				fuzzyDupes++
			}
		}
		final = append(final, kept[i])
	}

	// Add non-removed items
	for j := range kept {
		if removed[j] && j > 0 {
			// already filtered
		}
	}

	return final, urlDupes, fuzzyDupes
}

func normalizeURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.ToLower(rawURL)
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	path := strings.TrimRight(u.Path, "/")
	return host + path + "?" + u.RawQuery
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	return strings.TrimPrefix(host, "www.")
}

func levenshteinRatio(a, b string) float64 {
	ra := []rune(strings.ToLower(a))
	rb := []rune(strings.ToLower(b))
	maxLen := len(ra)
	if len(rb) > maxLen {
		maxLen = len(rb)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshtein(ra, rb)
	return 1.0 - float64(dist)/float64(maxLen)
}

func levenshtein(a, b []rune) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// SortBookmarks sorts bookmarks by category then title
func SortBookmarks(bookmarks []browser.Bookmark) {
	sort.Slice(bookmarks, func(i, j int) bool {
		if bookmarks[i].Category != bookmarks[j].Category {
			return bookmarks[i].Category < bookmarks[j].Category
		}
		return bookmarks[i].Title < bookmarks[j].Title
	})
}
