package query

import (
	"testing"
	"time"
)

func TestParseTimeWindow_startAndEndRFC3339(t *testing.T) {
	start, end, err := parseTimeWindow("", "2026-04-25T10:00:00Z", "2026-04-25T11:00:00Z", "")
	if err != nil {
		t.Fatalf("parseTimeWindow: %v", err)
	}
	if start.Format(time.RFC3339) != "2026-04-25T10:00:00Z" {
		t.Fatalf("unexpected start: %s", start.Format(time.RFC3339))
	}
	if end.Format(time.RFC3339) != "2026-04-25T11:00:00Z" {
		t.Fatalf("unexpected end: %s", end.Format(time.RFC3339))
	}
}

func TestParseTimeWindow_startRFC3339_requiresEndAfterStart(t *testing.T) {
	_, _, err := parseTimeWindow("", "2026-04-25T11:00:00Z", "2026-04-25T11:00:00Z", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTimeWindow_endUsesEndOverUntil(t *testing.T) {
	_, end, err := parseTimeWindow("", "", "2026-04-25T11:00:00Z", "2026-04-25T09:00:00Z")
	if err != nil {
		t.Fatalf("parseTimeWindow: %v", err)
	}
	if end.Format(time.RFC3339) != "2026-04-25T11:00:00Z" {
		t.Fatalf("unexpected end: %s", end.Format(time.RFC3339))
	}
}

func TestParseTimeWindow_untilSetsEnd(t *testing.T) {
	_, end, err := parseTimeWindow("", "", "", "2026-04-25T11:00:00Z")
	if err != nil {
		t.Fatalf("parseTimeWindow: %v", err)
	}
	if end.Format(time.RFC3339) != "2026-04-25T11:00:00Z" {
		t.Fatalf("unexpected end: %s", end.Format(time.RFC3339))
	}
}

func TestParseTimeWindow_sinceDerivesStartFromEnd(t *testing.T) {
	start, end, err := parseTimeWindow("30m", "", "2026-04-25T11:00:00Z", "")
	if err != nil {
		t.Fatalf("parseTimeWindow: %v", err)
	}
	if end.Format(time.RFC3339) != "2026-04-25T11:00:00Z" {
		t.Fatalf("unexpected end: %s", end.Format(time.RFC3339))
	}
	if start.Format(time.RFC3339) != "2026-04-25T10:30:00Z" {
		t.Fatalf("unexpected start: %s", start.Format(time.RFC3339))
	}
}

func TestParseTimeWindow_sinceMustBePositive(t *testing.T) {
	_, _, err := parseTimeWindow("-1s", "", "2026-04-25T11:00:00Z", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTimeWindow_sinceCannotBeZero(t *testing.T) {
	_, _, err := parseTimeWindow("0s", "", "2026-04-25T11:00:00Z", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTimeWindow_defaultSinceIsOneHour(t *testing.T) {
	start, end, err := parseTimeWindow("", "", "2026-04-25T11:00:00Z", "")
	if err != nil {
		t.Fatalf("parseTimeWindow: %v", err)
	}
	if start.Add(time.Hour) != end {
		t.Fatalf("expected start=end-1h, got start=%s end=%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	}
}

func TestParseTimeWindow_invalidRFC3339ErrorsAreAttributed(t *testing.T) {
	_, _, err := parseTimeWindow("", "not-a-time", "2026-04-25T11:00:00Z", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

