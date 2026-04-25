package query

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSession_Search_Golden_file(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	p := filepath.Join(dir, "demo.log")
	content := "" +
		"info hello\n" +
		"ERROR timeout A\n" +
		"warn ok\n" +
		"ERROR timeout B\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := NewSession()
	res, err := s.Search(ctx, SearchArgs{
		Source:   "file",
		FilePath: p,
		Query:    "error",
		Limit:    10,
		Start:    "2026-04-25T10:00:00Z",
		End:      "2026-04-25T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// Normalize temp-path noise for golden stability.
	if res.Query != nil {
		if _, ok := res.Query["filePath"]; ok {
			res.Query["filePath"] = "<tempfile>"
		}
	}
	assertGoldenJSON(t, filepath.Join("testdata", "session_search_file.json"), res)
}

func TestSession_SummarizeErrors_Golden_file(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	p := filepath.Join(dir, "errs.log")
	// Use distinct sizes so TF-IDF cluster sorting stays stable.
	content := "" +
		"ERROR timeout A\n" +
		"ERROR timeout B\n" +
		"ERROR timeout C\n" +
		"FATAL db down\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := NewSession()
	res, err := s.SummarizeErrors(ctx, SummarizeErrorsArgs{
		Source:   "file",
		FilePath: p,
		MaxLines: 100,
		Start:    "2026-04-25T10:00:00Z",
		End:      "2026-04-25T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}

	// Normalize temp-path noise for golden stability.
	if res.Query != nil {
		if _, ok := res.Query["filePath"]; ok {
			res.Query["filePath"] = "<tempfile>"
		}
	}
	assertGoldenJSON(t, filepath.Join("testdata", "session_summarize_errors_file.json"), res)
}

func assertGoldenJSON(t *testing.T, relPath string, v any) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	baseDir := filepath.Dir(thisFile)
	absPath := filepath.Join(baseDir, relPath)

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(absPath, append(b, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	gb, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", absPath, err)
	}

	var want any
	if err := json.Unmarshal(gb, &want); err != nil {
		t.Fatalf("unmarshal golden %s: %v", absPath, err)
	}

	var got any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal current: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("golden mismatch for %s (set UPDATE_GOLDEN=1 to update)", absPath)
	}
}

