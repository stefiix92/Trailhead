package backends

import (
	"testing"
)

func FuzzSplitDockerTimestamp_neverPanics(f *testing.F) {
	f.Add("2026-04-25T09:01:02Z hello")
	f.Add("not-a-time hello")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = splitDockerTimestamp(s)
	})
}

func FuzzSplitJournalctlShortISO_neverPanics(f *testing.F) {
	f.Add("2026-04-25T07:01:23+0200 host unit[1]: boom")
	f.Add("not-a-time host msg")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		_, _, _ = splitJournalctlShortISO(s)
	})
}

