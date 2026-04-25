package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/michalstefanec/trailhead/internal/backends"
	"github.com/michalstefanec/trailhead/internal/cluster"
	"github.com/michalstefanec/trailhead/internal/config"
	"github.com/michalstefanec/trailhead/internal/lineid"
)

// Session holds per-MCP connection state: line store, line IDs, and last cluster mapping.
type Session struct {
	cfg *config.Config

	mu         sync.Mutex
	allocators map[string]*lineid.Allocator
	lines      map[string]SearchMatch
	evictOrder []string

	// last summarize: cluster_id -> line IDs (in discovery order)
	clusterToLines map[string][]string
}

// NewSession creates a session with default settings (suitable for tests).
func NewSession() *Session {
	return NewSessionWithConfig(&config.Config{MaxSessionLines: config.DefaultMaxSessionLines})
}

// NewSessionWithConfig attaches Loki and eviction limits from the server.
func NewSessionWithConfig(cfg *config.Config) *Session {
	if cfg == nil {
		cfg = &config.Config{MaxSessionLines: config.DefaultMaxSessionLines}
	} else {
		c2 := *cfg
		if c2.MaxSessionLines <= 0 {
			c2.MaxSessionLines = config.DefaultMaxSessionLines
		}
		cfg = &c2
	}
	return &Session{
		cfg:            cfg,
		allocators:     newAllocators(),
		lines:          map[string]SearchMatch{},
		clusterToLines: map[string][]string{},
	}
}

func newAllocators() map[string]*lineid.Allocator {
	return map[string]*lineid.Allocator{
		"file": lineid.NewAllocator("file"),
		"loki": lineid.NewAllocator("loki"),
	}
}

func (s *Session) putLine(m SearchMatch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lines[m.LineID]; !ok {
		s.evictOrder = append(s.evictOrder, m.LineID)
	}
	s.lines[m.LineID] = m
	s.trimLocked()
}

func (s *Session) putLinesUnlocked(ms []SearchMatch) {
	for _, m := range ms {
		if _, ok := s.lines[m.LineID]; !ok {
			s.evictOrder = append(s.evictOrder, m.LineID)
		}
		s.lines[m.LineID] = m
	}
	s.trimLocked()
}

func (s *Session) trimLocked() {
	max := s.cfg.MaxSessionLines
	if max <= 0 {
		return
	}
	if len(s.lines) <= max {
		return
	}
	// evict oldest ~10% over the cap
	overflow := len(s.lines) - max
	if overflow < 100 {
		overflow = 100
	}
	if overflow > len(s.evictOrder) {
		overflow = len(s.evictOrder)
	}
	for i := 0; i < overflow && len(s.evictOrder) > 0; i++ {
		id := s.evictOrder[0]
		s.evictOrder = s.evictOrder[1:]
		delete(s.lines, id)
	}
}

type SearchArgs struct {
	Query    string `json:"query"`
	Limit    int    `json:"limit"`
	Source   string `json:"source"`
	FilePath string `json:"filePath"`
	// Loki (requires TRAILHEAD_LOKI_URL in server env)
	LokiLogQL          string `json:"lokiLogql"`
	LokiService        string `json:"lokiService"`
	LokiStreamSelector string `json:"lokiStreamSelector"`
	Since              string `json:"since"`
	Start              string `json:"start"`
	End                string `json:"end"`
	Until              string `json:"until"`
	FilterErrors       bool   `json:"filterErrors"`
}

// SearchMatch is one citeable line.
type SearchMatch struct {
	LineID    string `json:"line_id"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

// SearchResult is a bounded search response.
type SearchResult struct {
	Query   map[string]any `json:"query"`
	Count   int            `json:"count"`
	Matches []SearchMatch  `json:"matches"`
	LineIDs []string       `json:"line_ids"`
}

// Search runs a backend search, assigns source-prefixed line IDs, and stores matches.
func (s *Session) Search(ctx context.Context, args SearchArgs) (SearchResult, error) {
	src := args.Source
	if src == "" {
		src = "file"
	}
	lines, bq, err := s.queryBackend(ctx, src, args, false)
	if err != nil {
		return SearchResult{}, err
	}
	ms := s.registerLines(src, lines)
	return SearchResult{
		Query:   bq,
		Count:   len(ms),
		Matches: ms,
		LineIDs: lineIDsOf(ms),
	}, nil
}

func (s *Session) registerLines(source string, lines []backends.Line) []SearchMatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	alloc := s.allocators[source]
	if alloc == nil {
		alloc = lineid.NewAllocator(source)
		s.allocators[source] = alloc
	}
	out := make([]SearchMatch, 0, len(lines))
	for _, l := range lines {
		id := alloc.Next()
		m := SearchMatch{
			LineID:    id,
			Timestamp: l.Timestamp,
			Message:   l.Message,
			Source:    source,
		}
		out = append(out, m)
	}
	s.putLinesUnlocked(out)
	return out
}

func lineIDsOf(ms []SearchMatch) []string {
	ids := make([]string, len(ms))
	for i := range ms {
		ids[i] = ms[i].LineID
	}
	return ids
}

func (s *Session) queryBackend(ctx context.Context, source string, args SearchArgs, forErrors bool) ([]backends.Line, map[string]any, error) {
	start, end, err := parseTimeWindow(args.Since, args.Start, args.End, args.Until)
	if err != nil {
		return nil, nil, err
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	bq := map[string]any{
		"query":  args.Query,
		"limit":  limit,
		"source": source,
		"start":  start.Format(time.RFC3339),
		"end":    end.Format(time.RFC3339),
	}
	switch source {
	case "file":
		if args.FilePath == "" {
			return nil, bq, fmt.Errorf("filePath is required for source=file")
		}
		bq["filePath"] = args.FilePath
		b := backends.NewFileBackend(args.FilePath)
		var lines []backends.Line
		lines, err = b.Search(ctx, backends.SearchQuery{
			Query:        args.Query,
			Limit:        limit,
			Start:        start,
			End:          end,
			FilterErrors: forErrors || args.FilterErrors,
		})
		if err != nil {
			return nil, bq, err
		}
		return lines, bq, nil
	case "loki":
		if s.cfg == nil || !s.cfg.LokiConfigured() {
			return nil, bq, fmt.Errorf("loki: server is not configured (set TRAILHEAD_LOKI_URL)")
		}
		var logql string
		if args.LokiLogQL != "" {
			logql = args.LokiLogQL
		} else {
			var e error
			if forErrors {
				logql, e = buildLokiErrorLogQL(args.LokiService, args.LokiStreamSelector, args.Query)
			} else {
				logql, e = buildLokiSearchLogQL(args.LokiService, args.LokiStreamSelector, args.Query)
			}
			if e != nil {
				return nil, bq, e
			}
		}
		bq["lokiLogql"] = logql
		lb := &backends.LokiBackend{
			BaseURL: s.cfg.LokiBaseURL,
			OrgID:   s.cfg.LokiOrgID,
			Bearer:  s.cfg.LokiBearer,
		}
		lines, err := lb.Search(ctx, backends.SearchQuery{
			Query:     args.Query,
			Limit:     limit,
			Start:     start,
			End:       end,
			LokiLogQL: logql,
		})
		if err != nil {
			return nil, bq, err
		}
		return lines, bq, nil
	default:
		return nil, bq, backends.ErrUnsupportedSource
	}
}

// GetLinesByIDArgs resolves stored lines by id.
type GetLinesByIDArgs struct {
	LineIDs []string `json:"line_ids"`
}

// GetLinesByIDResult is the get_lines_by_id tool payload.
type GetLinesByIDResult struct {
	Query          map[string]any `json:"query"`
	Count          int            `json:"count"`
	Lines          []SearchMatch  `json:"lines"`
	Missing        []string       `json:"missing_line_ids"`
	PartialEvicted bool           `json:"partial_evicted,omitempty"`
}

// GetLinesByID returns full lines for ids still present in the session.
func (s *Session) GetLinesByID(_ context.Context, ids []string) (GetLinesByIDResult, error) {
	if len(ids) == 0 {
		return GetLinesByIDResult{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SearchMatch, 0, len(ids))
	missing := make([]string, 0)
	for _, id := range ids {
		if l, ok := s.lines[id]; ok {
			out = append(out, l)
		} else {
			missing = append(missing, id)
		}
	}
	return GetLinesByIDResult{
		Query: map[string]any{
			"line_ids": ids,
		},
		Count:          len(out),
		Lines:          out,
		Missing:        missing,
		PartialEvicted: len(missing) > 0,
	}, nil
}

// --- summarize_errors ---

// SummarizeErrorsArgs defines error summarization and clustering.
type SummarizeErrorsArgs struct {
	Source             string `json:"source"`
	FilePath           string `json:"filePath"`
	LokiService        string `json:"lokiService"`
	LokiStreamSelector string `json:"lokiStreamSelector"`
	LokiLogQL          string `json:"lokiLogql"`
	Since              string `json:"since"`
	Start              string `json:"start"`
	End                string `json:"end"`
	Until              string `json:"until"`
	MaxLines           int    `json:"maxLines"`
}

// SummarizeErrorsResult is structured aggregate + evidence for clusters.
type SummarizeErrorsResult struct {
	Query               map[string]any     `json:"query"`
	Total               int                `json:"total_lines"`
	Clusters            []SummarizeCluster `json:"clusters"`
	Coverage            float64            `json:"coverage_fraction"`
	TooSmall            bool               `json:"too_small_for_confident_clustering"`
	RepresentativeLimit int                `json:"representative_per_cluster_max"`
}

// SummarizeCluster is one error cluster.
type SummarizeCluster struct {
	ClusterID      string   `json:"cluster_id"`
	Size           int      `json:"size"`
	Signature      string   `json:"signature"`
	LineIDs        []string `json:"line_ids"`
	Representative []string `json:"representative_line_ids"`
	Score          float64  `json:"cohesion_score,omitempty"`
}

// SummarizeErrors fetches error-shaped lines, clusters them, and stores line IDs.
func (s *Session) SummarizeErrors(ctx context.Context, args SummarizeErrorsArgs) (SummarizeErrorsResult, error) {
	src := args.Source
	if src == "" {
		src = "file"
	}
	maxL := args.MaxLines
	if maxL <= 0 {
		maxL = 2000
	}
	sa := SearchArgs{
		Source:             src,
		FilePath:           args.FilePath,
		LokiService:        args.LokiService,
		LokiStreamSelector: args.LokiStreamSelector,
		LokiLogQL:          args.LokiLogQL,
		Since:              args.Since,
		Start:              args.Start,
		End:                args.End,
		Until:              args.Until,
		Query:              "",
		Limit:              maxL,
	}
	lines, bq, err := s.queryBackend(ctx, src, sa, true)
	if err != nil {
		return SummarizeErrorsResult{}, err
	}
	msgs := make([]string, len(lines))
	for i, l := range lines {
		msgs[i] = l.Message
	}
	result := SummarizeErrorsResult{
		Query:               bq,
		Total:               len(msgs),
		RepresentativeLimit: 5,
	}
	if len(msgs) == 0 {
		result.TooSmall = true
		s.mu.Lock()
		s.clusterToLines = map[string][]string{}
		s.mu.Unlock()
		return result, nil
	}
	if len(msgs) < 2 {
		result.TooSmall = true
		result.Coverage = 1
		if len(msgs) == 1 {
			ids := s.registerLines(src, lines)
			rep := lineIDsOf(ids)
			result.Clusters = []SummarizeCluster{{
				ClusterID:      "c1",
				Size:           1,
				Signature:      msgs[0],
				LineIDs:        rep,
				Representative: rep,
			}}
			s.mu.Lock()
			s.clusterToLines = map[string][]string{"c1": result.Clusters[0].LineIDs}
			s.mu.Unlock()
		}
		return result, nil
	}

	groups := cluster.ClusterTFIDF(msgs, 12)
	s.mu.Lock()
	s.clusterToLines = make(map[string][]string)
	s.mu.Unlock()

	clusters := make([]SummarizeCluster, 0, len(groups))
	var covered int
	for _, g := range groups {
		subLines := make([]backends.Line, 0, len(g.MemberIndex))
		for _, mi := range g.MemberIndex {
			subLines = append(subLines, lines[mi])
		}
		ids := s.registerLines(src, subLines)
		covered += len(ids)
		repN := 5
		if repN > len(ids) {
			repN = len(ids)
		}
		rep := lineIDsOf(ids)[:repN]
		cid := g.ID
		clusters = append(clusters, SummarizeCluster{
			ClusterID:      cid,
			Size:           g.Size,
			Signature:      g.Signature,
			LineIDs:        lineIDsOf(ids),
			Representative: rep,
			Score:          g.Score,
		})
		s.mu.Lock()
		s.clusterToLines[cid] = lineIDsOf(ids)
		s.mu.Unlock()
	}
	result.Clusters = clusters
	if result.Total > 0 {
		result.Coverage = float64(covered) / float64(result.Total)
	}
	return result, nil
}

// --- sample_cluster ---

// SampleClusterArgs requests more samples from a known cluster.
type SampleClusterArgs struct {
	ClusterID string `json:"cluster_id"`
	N         int    `json:"n"`
}

// SampleClusterResult is representative lines from a cluster.
type SampleClusterResult struct {
	Query          map[string]any `json:"query"`
	ClusterID      string         `json:"cluster_id"`
	Count          int            `json:"count"`
	LineIDs        []string       `json:"line_ids"`
	Representative []SearchMatch  `json:"representative_lines"`
	Note           string         `json:"note,omitempty"`
}

// SampleCluster returns up to n line records from a cluster produced by the last summarize_errors.
func (s *Session) SampleCluster(ctx context.Context, args SampleClusterArgs) (SampleClusterResult, error) {
	if args.ClusterID == "" {
		return SampleClusterResult{}, fmt.Errorf("cluster_id is required")
	}
	n := args.N
	if n <= 0 {
		n = 5
	}
	if n > 200 {
		n = 200
	}
	s.mu.Lock()
	ids, ok := s.clusterToLines[args.ClusterID]
	s.mu.Unlock()
	if !ok {
		return SampleClusterResult{}, fmt.Errorf("unknown cluster %q: run summarize_errors first in this session", args.ClusterID)
	}
	if len(ids) < n {
		n = len(ids)
	}
	pick := ids[:n]
	res, err := s.GetLinesByID(ctx, pick)
	if err != nil {
		return SampleClusterResult{}, err
	}
	return SampleClusterResult{
		Query: map[string]any{
			"cluster_id": args.ClusterID,
			"n":          n,
		},
		ClusterID:      args.ClusterID,
		Count:          n,
		LineIDs:        pick,
		Representative: res.Lines,
		Note:           "Cited evidence uses session line_ids; call get_lines_by_id to resolve full text if any were evicted.",
	}, nil
}

// --- diff_error_rate ---

// DiffErrorRateArgs compares error counts in two time windows.
type DiffErrorRateArgs struct {
	Source             string `json:"source"`
	FilePath           string `json:"filePath"`
	LokiService        string `json:"lokiService"`
	LokiStreamSelector string `json:"lokiStreamSelector"`
	// window A (current) and B (baseline), each specified with since (relative to "end" of that window) or explicit start/end
	ACurrentSince  string `json:"a_since"` // e.g. 1h
	BBaselineSince string `json:"b_since"` // e.g. 1h, measured ending at a_offset before A's start
	// AEnd and BEnd as RFC3339: if empty, A_end = now, B_end = A_start (back-to-back) when using offsets
	AEnd   string `json:"a_end"`
	AStart string `json:"a_start"`
	BEnd   string `json:"b_end"`
	BStart string `json:"b_start"`
}

// DiffErrorRateResult compares two windows.
type DiffErrorRateResult struct {
	Query        map[string]any `json:"query"`
	WindowA      map[string]any `json:"window_a"`
	WindowB      map[string]any `json:"window_b"`
	CountA       int            `json:"count_a"`
	CountB       int            `json:"count_b"`
	RatioBToA    float64        `json:"ratio_b_to_a"`
	DeltaAminusB int            `json:"delta_a_minus_b"`
}

// DiffErrorRate runs two error-scoped fetches and compares counts.
func (s *Session) DiffErrorRate(ctx context.Context, args DiffErrorRateArgs) (DiffErrorRateResult, error) {
	src := args.Source
	if src == "" {
		src = "file"
	}
	if src == "file" {
		return DiffErrorRateResult{}, fmt.Errorf("diff_error_rate is time-bounded: use source=loki (set TRAILHEAD_LOKI_URL) or compare using summarize_errors on file logs")
	}
	if args.ACurrentSince == "" && args.AStart == "" {
		return DiffErrorRateResult{}, fmt.Errorf("set a_since or a_start/a_end for the current window")
	}
	// Default baseline: same duration as A, immediately before A
	aStart, aEnd, err := parseTimeWindow(args.ACurrentSince, args.AStart, args.AEnd, "")
	if err != nil {
		return DiffErrorRateResult{}, err
	}
	var bStart, bEnd time.Time
	if args.BStart != "" || args.BEnd != "" {
		bStart, bEnd, err = parseTimeWindow(args.BBaselineSince, args.BStart, args.BEnd, "")
		if err != nil {
			return DiffErrorRateResult{}, err
		}
	} else {
		dur := aEnd.Sub(aStart)
		bEnd = aStart
		bStart = bEnd.Add(-dur)
	}
	limit := 10_000
	countA, err := s.countErrors(ctx, src, args, aStart, aEnd, limit)
	if err != nil {
		return DiffErrorRateResult{}, err
	}
	countB, err := s.countErrors(ctx, src, args, bStart, bEnd, limit)
	if err != nil {
		return DiffErrorRateResult{}, err
	}
	ra := float64(0)
	if countA > 0 {
		ra = float64(countB) / float64(countA)
	} else {
		ra = 0
	}
	return DiffErrorRateResult{
		Query: map[string]any{
			"source":  src,
			"a_since": args.ACurrentSince,
			"b_since": args.BBaselineSince,
		},
		WindowA: map[string]any{
			"start": aStart.Format(time.RFC3339),
			"end":   aEnd.Format(time.RFC3339),
		},
		WindowB: map[string]any{
			"start": bStart.Format(time.RFC3339),
			"end":   bEnd.Format(time.RFC3339),
		},
		CountA:       countA,
		CountB:       countB,
		RatioBToA:    ra,
		DeltaAminusB: countA - countB,
	}, nil
}

func (s *Session) countErrors(ctx context.Context, source string, args DiffErrorRateArgs, start, end time.Time, cap int) (int, error) {
	sa := SearchArgs{
		Source:             source,
		FilePath:           args.FilePath,
		LokiService:        args.LokiService,
		LokiStreamSelector: args.LokiStreamSelector,
		FilterErrors:       true,
		Limit:              cap,
	}
	sa.Since = ""
	sa.Start = start.Format(time.RFC3339)
	sa.End = end.Format(time.RFC3339)
	lines, _, err := s.queryBackend(ctx, source, sa, true)
	if err != nil {
		return 0, err
	}
	return len(lines), nil
}

// --- correlated_events ---

// CorrelatedEventsArgs fetches a bounded window around a timestamp, plus optional deploy markers.
type CorrelatedEventsArgs struct {
	Around             string `json:"around"`
	Window             string `json:"window"`
	Source             string `json:"source"`
	FilePath           string `json:"filePath"`
	LokiService        string `json:"lokiService"`
	LokiStreamSelector string `json:"lokiStreamSelector"`
	Limit              int    `json:"limit"`
}

// CorrelatedEventsResult is nearby lines plus markers.
type CorrelatedEventsResult struct {
	Query   map[string]any `json:"query"`
	Window  map[string]any `json:"window"`
	Markers []Marker       `json:"markers,omitempty"`
	Events  []SearchMatch  `json:"events"`
	LineIDs []string       `json:"line_ids"`
}

// Marker is an external time-bounded label (e.g. deploy) from file or request.
type Marker struct {
	Timestamp time.Time `json:"timestamp"`
	Label     string    `json:"label"`
}

type markerFile struct {
	MTime string `json:"time"`
	Label string `json:"label"`
}

// CorrelatedEvents returns log lines in [around-window, around+window] and optional markers.
func (s *Session) CorrelatedEvents(ctx context.Context, args CorrelatedEventsArgs) (CorrelatedEventsResult, error) {
	if strings.TrimSpace(args.Around) == "" {
		return CorrelatedEventsResult{}, fmt.Errorf("around (RFC3339) is required")
	}
	around, err := time.Parse(time.RFC3339, args.Around)
	if err != nil {
		return CorrelatedEventsResult{}, fmt.Errorf("around: %w", err)
	}
	wd := 2 * time.Minute
	if w := strings.TrimSpace(args.Window); w != "" {
		d, perr := time.ParseDuration(w)
		if perr != nil {
			return CorrelatedEventsResult{}, fmt.Errorf("window: %w", perr)
		}
		wd = d
	}
	half := wd / 2
	start := around.Add(-half)
	end := around.Add(half)
	src := args.Source
	if src == "" {
		if s.cfg != nil && s.cfg.LokiConfigured() {
			src = "loki"
		} else {
			src = "file"
		}
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 100
	}
	sa := SearchArgs{
		Source:             src,
		FilePath:           args.FilePath,
		LokiService:        args.LokiService,
		LokiStreamSelector: args.LokiStreamSelector,
		Limit:              limit,
		Since:              "",
		Start:              start.Format(time.RFC3339),
		End:                end.Format(time.RFC3339),
		Query:              "",
	}
	lines, bq, err := s.queryBackend(ctx, src, sa, false)
	if err != nil {
		return CorrelatedEventsResult{}, err
	}
	// for file, queryBackend already filtered by time? File ignores time - we filter in app
	if src == "file" {
		lines, err = filterFileLinesByTime(lines, start, end)
		if err != nil {
			return CorrelatedEventsResult{}, err
		}
		if len(lines) > limit {
			lines = lines[:limit]
		}
	}
	ms := s.registerLines(src, lines)
	markers, err := loadMarkers(s.cfg, start, end)
	if err != nil {
		return CorrelatedEventsResult{}, err
	}
	return CorrelatedEventsResult{
		Query: bq,
		Window: map[string]any{
			"start":    start.Format(time.RFC3339),
			"end":      end.Format(time.RFC3339),
			"around":   around.Format(time.RFC3339),
			"halfspan": (wd / 2).String(),
		},
		Markers: markers,
		Events:  ms,
		LineIDs: lineIDsOf(ms),
	}, nil
}

func filterFileLinesByTime(in []backends.Line, start, end time.Time) ([]backends.Line, error) {
	// if timestamps are empty, keep all
	var out []backends.Line
	for _, l := range in {
		if l.Timestamp == "" {
			out = append(out, l)
			continue
		}
		t, err := time.Parse(time.RFC3339, l.Timestamp)
		if err != nil {
			out = append(out, l)
			continue
		}
		if (t.Equal(start) || t.After(start)) && t.Before(end) {
			out = append(out, l)
		}
	}
	return out, nil
}

func loadMarkers(cfg *config.Config, start, end time.Time) ([]Marker, error) {
	if cfg == nil || strings.TrimSpace(cfg.MarkersFile) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(cfg.MarkersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw []markerFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var out []Marker
	for _, m := range raw {
		if m.MTime == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, m.MTime)
		if err != nil {
			continue
		}
		if (t.Equal(start) || t.After(start)) && t.Before(end) {
			out = append(out, Marker{Timestamp: t, Label: m.Label})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out, nil
}
