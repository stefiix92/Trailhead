package backends

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDockerBackend_Search_integration(t *testing.T) {
	if os.Getenv("TRAILHEAD_RUN_DOCKER_IT") != "1" {
		t.Skip("set TRAILHEAD_RUN_DOCKER_IT=1 to run")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := "trailhead-it-backend"
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()

	run := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "busybox", "sh", "-c",
		`echo "ERROR timeout A"; echo "INFO ok"; echo "ERROR timeout B"; sleep 60`,
	)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("docker run: %v: %s", err, strings.TrimSpace(string(out)))
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	b := &DockerBackend{Container: name}
	res, err := b.Search(ctx, SearchQuery{
		Query:        "timeout",
		Limit:        10,
		Start:        time.Time{},
		End:          time.Time{},
		FilterErrors: true,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(res), res)
	}
	if !strings.Contains(res[0].Message, "timeout") || !strings.Contains(res[1].Message, "timeout") {
		t.Fatalf("unexpected messages: %+v", res)
	}
}

