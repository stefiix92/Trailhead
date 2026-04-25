package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

	// This method does not exist yet (RED): once implemented it should return the stored line.
	lines, missing, err := s.GetLinesByID(ctx, res.LineIDs)
	if err != nil {
		t.Fatalf("get_lines_by_id: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing IDs, got %v", missing)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Message != "ERROR boom" {
		t.Fatalf("unexpected message: %q", lines[0].Message)
	}
	if lines[0].LineID != res.LineIDs[0] {
		t.Fatalf("unexpected line id: %q", lines[0].LineID)
	}
}

func TestSession_GetLinesByID_reportsMissingIDs(t *testing.T) {
	ctx := context.Background()
	s := NewSession()

	lines, missing, err := s.GetLinesByID(ctx, []string{"log:9999"})
	if err != nil {
		t.Fatalf("get_lines_by_id: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
	if len(missing) != 1 || missing[0] != "log:9999" {
		t.Fatalf("expected missing log:9999, got %v", missing)
	}
}

