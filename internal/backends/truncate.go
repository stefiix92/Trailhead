package backends

import "strings"

func truncateForError(b []byte, n int) string {
	if n <= 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
