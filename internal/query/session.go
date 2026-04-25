package query

import (
	"context"
	"sync"

	"github.com/michalstefanec/trailhead/internal/backends"
	"github.com/michalstefanec/trailhead/internal/lineid"
)

type Session struct {
	ids *lineid.Allocator

	mu    sync.RWMutex
	lines map[string]SearchMatch // line_id -> stored match (message + metadata)
}

func NewSession() *Session {
	return &Session{
		ids: lineid.NewAllocator("log"),
		lines: map[string]SearchMatch{},
	}
}

type SearchArgs struct {
	Query    string `json:"query"`
	Limit    int    `json:"limit"`
	Source   string `json:"source"`
	FilePath string `json:"filePath"`
}

type SearchMatch struct {
	LineID    string `json:"line_id"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

type SearchResult struct {
	Query   map[string]any  `json:"query"`
	Count   int             `json:"count"`
	Matches []SearchMatch   `json:"matches"`
	LineIDs []string        `json:"line_ids"`
}

type GetLinesByIDArgs struct {
	LineIDs []string `json:"line_ids"`
}

type GetLinesByIDResult struct {
	Query     map[string]any `json:"query"`
	Count     int            `json:"count"`
	Lines     []SearchMatch  `json:"lines"`
	Missing   []string       `json:"missing_line_ids"`
}

func (s *Session) Search(ctx context.Context, args SearchArgs) (SearchResult, error) {
	src := args.Source
	if src == "" {
		src = "file"
	}

	var b backends.Backend
	switch src {
	case "file":
		b = backends.NewFileBackend(args.FilePath)
	default:
		return SearchResult{}, backends.ErrUnsupportedSource
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}

	lines, err := b.Search(ctx, backends.SearchQuery{Query: args.Query, Limit: limit})
	if err != nil {
		return SearchResult{}, err
	}

	matches := make([]SearchMatch, 0, len(lines))
	lineIDs := make([]string, 0, len(lines))
	for _, l := range lines {
		id := s.ids.Next()
		lineIDs = append(lineIDs, id)
		m := SearchMatch{
			LineID:    id,
			Timestamp: l.Timestamp,
			Message:   l.Message,
			Source:    src,
		}
		matches = append(matches, m)

		s.mu.Lock()
		s.lines[id] = m
		s.mu.Unlock()
	}

	return SearchResult{
		Query: map[string]any{
			"query":  args.Query,
			"limit":  limit,
			"source": src,
		},
		Count:   len(matches),
		Matches: matches,
		LineIDs: lineIDs,
	}, nil
}

func (s *Session) GetLinesByID(_ context.Context, ids []string) ([]SearchMatch, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SearchMatch, 0, len(ids))
	missing := make([]string, 0)
	for _, id := range ids {
		if l, ok := s.lines[id]; ok {
			out = append(out, l)
		} else {
			missing = append(missing, id)
		}
	}
	return out, missing, nil
}

