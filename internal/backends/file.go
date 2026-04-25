package backends

import (
	"bufio"
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
)

type FileBackend struct {
	path string
}

func NewFileBackend(path string) *FileBackend {
	return &FileBackend{path: path}
}

func (b *FileBackend) Search(ctx context.Context, q SearchQuery) ([]Line, error) {
	if b.path == "" {
		return nil, errors.New("filePath is required for source=file")
	}
	f, err := os.Open(b.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	needle := strings.ToLower(q.Query)
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}

	var out []Line
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}

		msg := sc.Text()
		low := strings.ToLower(msg)
		if q.FilterErrors && !looksErrorLine(low) {
			continue
		}
		if needle == "" || strings.Contains(low, needle) {
			out = append(out, Line{Message: msg})
			if len(out) >= limit {
				break
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

var reErrorWord = regexp.MustCompile(`(?i)\b(error|errors|errored|exception|panic|fatal|err)\b`)

func looksErrorLine(low string) bool {
	return reErrorWord.MatchString(low)
}
