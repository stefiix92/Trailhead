package backends

import (
	"context"
	"errors"
	"time"
)

var ErrUnsupportedSource = errors.New("unsupported source")

// SearchQuery is passed to all backends. Fields not used by a backend are ignored.
type SearchQuery struct {
	Query string
	Limit int
	// Start and End bound the time range (UTC). Zero Start/End is interpreted per backend
	// (Loki defaults to [now-1h, now] when End is set but Start is zero, etc.).
	Start time.Time
	End   time.Time
	// Loki: full LogQL. Required for LokiBackend.
	LokiLogQL string
	// File: when true, only lines that look like errors are returned.
	FilterErrors bool
}

type Line struct {
	Timestamp string
	Message   string
}

type Backend interface {
	Search(ctx context.Context, q SearchQuery) ([]Line, error)
}
