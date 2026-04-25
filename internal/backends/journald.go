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

// JournaldBackend reads logs via journalctl.
// It is bounded and intended for single-host / systemd-based deployments.
type JournaldBackend struct {
	// Unit is the systemd unit (e.g. nginx.service). Optional.
	Unit string
	// Identifier maps to journalctl's SYSLOG_IDENTIFIER match. Optional.
	Identifier string
}

func (b *JournaldBackend) Search(ctx context.Context, q SearchQuery) ([]Line, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}

	args := []string{
		"--output=short-iso",
		"--no-pager",
	}
	if u := strings.TrimSpace(b.Unit); u != "" {
		args = append(args, "--unit", u)
	}
	// journalctl supports matching by field assignment like SYSLOG_IDENTIFIER=foo.
	if id := strings.TrimSpace(b.Identifier); id != "" {
		if !looksLikeJournalIdentifier(id) {
			return nil, errors.New("journald: journaldIdentifier must not contain whitespace or '='")
		}
		args = append(args, fmt.Sprintf("SYSLOG_IDENTIFIER=%s", id))
	}
	if !q.Start.IsZero() {
		args = append(args, "--since", q.Start.UTC().Format(time.RFC3339Nano))
	}
	if !q.End.IsZero() {
		args = append(args, "--until", q.End.UTC().Format(time.RFC3339Nano))
	}

	// Ask journalctl for more than we return, because we still apply substring/error heuristics.
	tail := limit * 50
	if tail < 500 {
		tail = 500
	}
	if tail > 5000 {
		tail = 5000
	}
	args = append(args, "-n", fmt.Sprintf("%d", tail))

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := truncateForError(out, 1200)
		if msg != "" {
			return nil, fmt.Errorf("journalctl: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("journalctl: %w", err)
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
		ts, msg, ok := splitJournalctlShortISO(raw)
		if !ok {
			// Fallback: treat whole line as message.
			ts = ""
			msg = raw
		}
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

func splitJournalctlShortISO(line string) (timestamp string, message string, ok bool) {
	// journalctl --output=short-iso:
	// 2026-04-25T07:01:23+0200 hostname unit[pid]: message...
	// We take the first whitespace-separated token as timestamp if it parses in common layouts.
	i := strings.IndexByte(line, ' ')
	if i <= 0 {
		return "", "", false
	}
	ts := line[:i]
	rest := strings.TrimSpace(line[i+1:])
	if rest == "" {
		return "", "", false
	}
	if isJournalctlShortISO(ts) {
		return ts, rest, true
	}
	return "", "", false
}

func isJournalctlShortISO(ts string) bool {
	// journalctl uses +HHMM (no colon) by default.
	if _, err := time.Parse("2006-01-02T15:04:05-0700", ts); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339, ts); err == nil {
		return true
	}
	return false
}

func looksLikeJournalIdentifier(s string) bool {
	if strings.ContainsAny(s, " \t\r\n=") {
		return false
	}
	return true
}

