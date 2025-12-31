package parse

import (
	"regexp"
	"strings"
)

// Extract returns candidate reference tokens from a decl-like plaintext file.
// We deliberately avoid relying on quotes: tokens may be unquoted.
// We keep only tokens containing '/' because canonical resource names do.
var tokenRe = regexp.MustCompile(`[A-Za-z0-9_./-]+`)

func Extract(text string) []string {
	raw := tokenRe.FindAllString(text, -1)
	out := make([]string, 0, len(raw))

	for _, t := range raw {
		if !strings.Contains(t, "/") {
			continue
		}
		t = strings.Trim(t, "\"';,(){}[]")
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}
