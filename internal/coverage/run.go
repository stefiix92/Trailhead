package coverage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ToolArgs struct {
	// Packages are passed to `go test`. Default is ["./..."].
	Packages []string `json:"packages,omitempty"`
	// TopFiles bounds per-file results in the output.
	TopFiles int `json:"top_files,omitempty"`
}

type ToolResult struct {
	Query   map[string]any       `json:"query"`
	Summary CoverProfileSummary  `json:"summary"`
}

func Run(ctx context.Context, args ToolArgs) (ToolResult, error) {
	pkgs := args.Packages
	if len(pkgs) == 0 {
		pkgs = []string{"./..."}
	}
	topFiles := args.TopFiles
	if topFiles <= 0 {
		topFiles = 10
	}

	dir, err := os.MkdirTemp("", "trailhead-cover-*")
	if err != nil {
		return ToolResult{}, err
	}
	defer os.RemoveAll(dir)

	profilePath := filepath.Join(dir, "cover.out")
	cmdArgs := append([]string{"test", "-coverprofile=" + profilePath}, pkgs...)
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{}, fmt.Errorf("go test failed: %w\n%s", err, string(out))
	}

	profile, err := os.ReadFile(profilePath)
	if err != nil {
		return ToolResult{}, err
	}

	summary, err := ParseCoverProfile(profile, ParseOptions{TopFiles: topFiles})
	if err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Query: map[string]any{
			"packages":  pkgs,
			"top_files": topFiles,
		},
		Summary: summary,
	}, nil
}

