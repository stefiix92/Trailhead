package backends

import "testing"

func TestSplitDockerTimestamp_parsesRFC3339Nano(t *testing.T) {
	ts, msg := splitDockerTimestamp("2026-04-25T09:01:02.123456789Z hello world")
	if ts != "2026-04-25T09:01:02.123456789Z" {
		t.Fatalf("unexpected ts: %q", ts)
	}
	if msg != "hello world" {
		t.Fatalf("unexpected msg: %q", msg)
	}
}

func TestSplitDockerTimestamp_parsesRFC3339(t *testing.T) {
	ts, msg := splitDockerTimestamp("2026-04-25T09:01:02Z hi")
	if ts != "2026-04-25T09:01:02Z" {
		t.Fatalf("unexpected ts: %q", ts)
	}
	if msg != "hi" {
		t.Fatalf("unexpected msg: %q", msg)
	}
}

func TestSplitDockerTimestamp_fallsBackOnNonTimestamp(t *testing.T) {
	ts, msg := splitDockerTimestamp("not-a-timestamp hello")
	if ts != "" {
		t.Fatalf("expected empty ts, got %q", ts)
	}
	if msg != "not-a-timestamp hello" {
		t.Fatalf("expected original line, got %q", msg)
	}
}

