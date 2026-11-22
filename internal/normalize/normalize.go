package normalize

import (
	"strings"
	"unicode"
	"golang.org/x/text/unicode/norm"
)

var prefixes = []string{
	"thành phố", "thị trấn", "thị xã",
	"huyện", "phường", "quận", "tỉnh", "xã",
	"tp", "tx", "tt",
	"h", "p", "q", "x",
}

// Name normalizes a Vietnamese place name
func Name(s string) string {
	// Step 1: NFC
	s = norm.NFC.String(s)
	// Step 2: trim + collapse whitespace
	s = strings.TrimSpace(s)
	s = collapseSpaces(s)
	// Step 3: lowercase
	s = strings.ToLower(s)
	// Step 4: strip one leading prefix
	s = stripPrefix(s)
	return s
}

func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

func stripPrefix(s string) string {
	for _, prefix := range prefixes {
		if !strings.HasPrefix(s, prefix) {
			continue
		}
		rest := s[len(prefix):]
		if len(rest) == 0 {
			// The entire string is just the prefix — don't strip
			return s
		}
		// Check: rest must start with dot or space
		if rest[0] == '.' {
			// Skip dot and optional following spaces
			rest = rest[1:]
			rest = strings.TrimLeft(rest, " ")
			if rest == "" {
				return s // nothing after prefix+dot
			}
			return rest
		}
		if rest[0] == ' ' {
			rest = strings.TrimLeft(rest, " ")
			if rest == "" {
				return s // nothing after prefix+space
			}
			return rest
		}
		// Not followed by dot or space — not a valid prefix
	}
	return s
}
