package backends

import "testing"

func TestSplitJournalctlShortISO_parses(t *testing.T) {
	ts, msg, ok := splitJournalctlShortISO("2026-04-25T07:01:23+0200 host unit[1]: boom")
	if !ok {
		t.Fatalf("expected ok")
	}
	if ts != "2026-04-25T07:01:23+0200" {
		t.Fatalf("unexpected ts: %q", ts)
	}
	if msg != "host unit[1]: boom" {
		t.Fatalf("unexpected msg: %q", msg)
	}
}

func TestSplitJournalctlShortISO_rejectsInvalid(t *testing.T) {
	_, _, ok := splitJournalctlShortISO("not-a-time host msg")
	if ok {
		t.Fatalf("expected not ok")
	}
}

func TestLooksLikeJournalIdentifier(t *testing.T) {
	if !looksLikeJournalIdentifier("nginx") {
		t.Fatalf("expected nginx ok")
	}
	if looksLikeJournalIdentifier("has space") {
		t.Fatalf("expected whitespace rejected")
	}
	if looksLikeJournalIdentifier("has=eq") {
		t.Fatalf("expected '=' rejected")
	}
}

