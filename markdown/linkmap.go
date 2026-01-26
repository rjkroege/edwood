package markdown

// LinkMap maps positions in rendered content to URLs for links.
type LinkMap struct {
	entries []LinkEntry
}

// LinkEntry tracks a link's position in rendered content and its URL.
type LinkEntry struct {
	Start int    // Rune position of link start in rendered content
	End   int    // Rune position of link end (exclusive) in rendered content
	URL   string // The link target URL
}

// NewLinkMap creates an empty LinkMap.
func NewLinkMap() *LinkMap {
	return &LinkMap{}
}

// Add registers a link from start to end (exclusive) with the given URL.
func (lm *LinkMap) Add(start, end int, url string) {
	lm.entries = append(lm.entries, LinkEntry{
		Start: start,
		End:   end,
		URL:   url,
	})
}

// URLAt returns the URL if pos is within a link, or empty string if not.
// The range is [start, end) - start is inclusive, end is exclusive.
func (lm *LinkMap) URLAt(pos int) string {
	for _, e := range lm.entries {
		if pos >= e.Start && pos < e.End {
			return e.URL
		}
	}
	return ""
}
