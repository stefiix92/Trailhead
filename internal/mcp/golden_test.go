package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/michalstefanec/trailhead/internal/config"
)

func TestToolsList_Golden_default(t *testing.T) {
	s := newServer(&config.Config{})
	got := map[string]any{"tools": s.toolList()}
	assertGoldenJSON(t, filepath.Join("testdata", "tools_list.json"), got)
}

func TestToolsList_Golden_devtools(t *testing.T) {
	s := newServer(&config.Config{DevTools: true})
	got := map[string]any{"tools": s.toolList()}
	assertGoldenJSON(t, filepath.Join("testdata", "tools_list_devtools.json"), got)
}

func assertGoldenJSON(t *testing.T, relPath string, v any) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	baseDir := filepath.Dir(thisFile)
	absPath := filepath.Join(baseDir, relPath)

	// Normalize Go maps/ints into JSON-types (map[string]any, []any, float64, etc.)
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

