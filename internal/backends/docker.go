package backends

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DockerBackend reads logs via the local docker CLI.
// It is intentionally bounded (tail + limit) to preserve "query, don't dump".
type DockerBackend struct {
	// Exactly one of Container or ComposeService should be set.
	Container       string
	ComposeService  string
	ComposeProject  string
	ComposeFilePath string
}

func (b *DockerBackend) Search(ctx context.Context, q SearchQuery) ([]Line, error) {
	container := strings.TrimSpace(b.Container)
	if container == "" {
		if strings.TrimSpace(b.ComposeService) == "" {
			return nil, errors.New("docker: set dockerContainer or dockerComposeService")
		}
		cid, err := b.resolveComposeContainer(ctx)
		if err != nil {
			return nil, err
		}
		container = cid
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}

	args := []string{"logs", "--timestamps"}
	if !q.Start.IsZero() {
		args = append(args, "--since", q.Start.UTC().Format(time.RFC3339Nano))
	}
	if !q.End.IsZero() {
		args = append(args, "--until", q.End.UTC().Format(time.RFC3339Nano))
	}

	// Bound the amount of data we pull from docker itself.
	// We still apply our own substring + error filtering after.
	tail := limit * 50
	if tail < 500 {
		tail = 500
	}
	if tail > 5000 {
		tail = 5000
	}
	args = append(args, "--tail", fmt.Sprintf("%d", tail))
	args = append(args, container)

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := truncateForError(out, 1200)
		if msg != "" {
			return nil, fmt.Errorf("docker logs: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("docker logs: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(q.Query))
	var res []Line
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}
		raw := sc.Text()
		ts, msg := splitDockerTimestamp(raw)
		low := strings.ToLower(msg)
		if q.FilterErrors && !looksErrorLine(low) {
			continue
		}
		if needle == "" || strings.Contains(low, needle) {
			res = append(res, Line{Timestamp: ts, Message: msg})
			if len(res) >= limit {
				break
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (b *DockerBackend) resolveComposeContainer(ctx context.Context) (string, error) {
	svc := strings.TrimSpace(b.ComposeService)
	if svc == "" {
		return "", errors.New("docker compose: dockerComposeService is empty")
	}
	args := []string{"compose"}
	if p := strings.TrimSpace(b.ComposeProject); p != "" {
		args = append(args, "--project-name", p)
	}
	if f := strings.TrimSpace(b.ComposeFilePath); f != "" {
		args = append(args, "-f", f)
	}
	args = append(args, "ps", "-q", svc)

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := truncateForError(out, 1200)
		if msg != "" {
			return "", fmt.Errorf("docker compose ps: %w: %s", err, msg)
		}
		return "", fmt.Errorf("docker compose ps: %w", err)
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		id := strings.TrimSpace(sc.Text())
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("docker compose ps: no container found for service %q", svc)
}

func splitDockerTimestamp(line string) (timestamp string, message string) {
	// docker logs --timestamps prepends RFC3339-ish timestamp, then a space.
	// If it doesn't look like that, keep timestamp empty.
	i := strings.IndexByte(line, ' ')
	if i <= 0 {
		return "", line
	}
	ts := line[:i]
	msg := strings.TrimSpace(line[i+1:])
	if _, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return ts, msg
	}
	if _, err := time.Parse(time.RFC3339, ts); err == nil {
		return ts, msg
	}
	return "", line
}

