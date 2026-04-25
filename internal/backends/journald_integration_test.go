package backends

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestJournaldBackend_Search_integration(t *testing.T) {
	if os.Getenv("TRAILHEAD_RUN_JOURNALD_IT") != "1" {
		t.Skip("set TRAILHEAD_RUN_JOURNALD_IT=1 to run")
	}
	if runtime.GOOS != "linux" {
		t.Skip("journald integration test is linux-only")
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		t.Skip("journalctl not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b := &JournaldBackend{}
	_, err := b.Search(ctx, SearchQuery{
		Query:        "",
		Limit:        5,
		Start:        time.Now().Add(-5 * time.Minute),
		End:          time.Now(),
		FilterErrors: false,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
}

