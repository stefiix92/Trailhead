package query

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stefiix92/Trailhead/internal/backends"
)

func TestSession_GetLinesByID_returnsStoredSearchLines(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	p := filepath.Join(dir, "demo.log")
	if err := os.WriteFile(p, []byte("hello\nERROR boom\nhello again\n"), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := NewSession()
	res, err := s.Search(ctx, SearchArgs{
		Source:   "file",
		FilePath: p,
		Query:    "error",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Count != 1 {
		t.Fatalf("expected 1 match, got %d", res.Count)
	}
	if len(res.LineIDs) != 1 {
		t.Fatalf("expected 1 line id, got %d", len(res.LineIDs))
	}

	g, err := s.GetLinesByID(ctx, res.LineIDs)
	if err != nil {
		t.Fatalf("get_lines_by_id: %v", err)
	}
	if len(g.Missing) != 0 {
		t.Fatalf("expected no missing IDs, got %v", g.Missing)
	}
	if len(g.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(g.Lines))
	}
	if g.Lines[0].Message != "ERROR boom" {
		t.Fatalf("unexpected message: %q", g.Lines[0].Message)
	}
	if g.Lines[0].LineID != res.LineIDs[0] {
		t.Fatalf("unexpected line id: %q", g.Lines[0].LineID)
	}
}

func TestSession_GetLinesByID_reportsMissingIDs(t *testing.T) {
	ctx := context.Background()
	s := NewSession()

	g, err := s.GetLinesByID(ctx, []string{"file:9999"})
	if err != nil {
		t.Fatalf("get_lines_by_id: %v", err)
	}
	if len(g.Lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(g.Lines))
	}
	if len(g.Missing) != 1 || g.Missing[0] != "file:9999" {
		t.Fatalf("expected missing file:9999, got %v", g.Missing)
	}
}

func TestSession_SummarizeErrors_clusters(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := filepath.Join(dir, "e.log")
	content := "ERROR: timeout A\nERROR: timeout B\nFATAL: db down\n"
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	s := NewSession()
	res, err := s.SummarizeErrors(ctx, SummarizeErrorsArgs{
		Source:   "file",
		FilePath: p,
		MaxLines: 100,
	})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if res.Total != 3 {
		t.Fatalf("expected 3 error lines, got %d", res.Total)
	}
	if len(res.Clusters) < 1 {
		t.Fatalf("expected clusters, got 0")
	}
	// sample first cluster
	cid := res.Clusters[0].ClusterID
	sr, err := s.SampleCluster(ctx, SampleClusterArgs{ClusterID: cid, N: 3})
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	if sr.Count < 1 {
		t.Fatalf("expected samples")
	}
}

func TestSession_registerLines_prefixesBySource(t *testing.T) {
	s := NewSession()
	ms := s.registerLines("docker", []backends.Line{{Message: "hello"}})
	if len(ms) != 1 {
		t.Fatalf("expected 1, got %d", len(ms))
	}
	if !strings.HasPrefix(ms[0].LineID, "docker:") {
		t.Fatalf("expected docker: prefix, got %q", ms[0].LineID)
	}

	ms2 := s.registerLines("journald", []backends.Line{{Message: "hello2"}})
	if len(ms2) != 1 {
		t.Fatalf("expected 1, got %d", len(ms2))
	}
	if !strings.HasPrefix(ms2[0].LineID, "journald:") {
		t.Fatalf("expected journald: prefix, got %q", ms2[0].LineID)
	}
}
