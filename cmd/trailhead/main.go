package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// This CLI is intentionally small: it exists to make citation verification workable
// in terminal/CI contexts (where you can't "click" a line_id).

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var ue usageError
		if errors.As(err, &ue) {
			_, _ = fmt.Fprintln(os.Stderr, ue.Error())
			os.Exit(2)
		}
		_, _ = fmt.Fprintf(os.Stderr, "trailhead: %v\n", err)
		os.Exit(1)
	}
}

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return usageError{msg: usage()}
	}
	switch args[0] {
	case "help", "-h", "--help":
		_, _ = fmt.Fprintln(out, usage())
		return nil
	case "show":
		if len(args) < 2 {
			return usageError{msg: usage()}
		}
		lineIDs := args[1:]
		return show(ctx, lineIDs, out, errOut)
	default:
		return usageError{msg: usage()}
	}
}

func usage() string {
	return strings.TrimSpace(`
Usage:
  trailhead show <line_id> [<line_id> ...]

Environment:
  TRAILHEAD_MCP_CMD   Command to run the stdio server (default: ./trailhead-mcp)

Examples:
  trailhead show loki:42
  TRAILHEAD_MCP_CMD=/usr/local/bin/trailhead-mcp trailhead show file:12 docker:7
`)
}

func show(ctx context.Context, lineIDs []string, out, errOut io.Writer) error {
	cmdStr := strings.TrimSpace(os.Getenv("TRAILHEAD_MCP_CMD"))
	if cmdStr == "" {
		cmdStr = "./trailhead-mcp"
	}
	cmdArgs, err := splitCommand(cmdStr)
	if err != nil {
		return fmt.Errorf("TRAILHEAD_MCP_CMD: %w", err)
	}

	// Keep CLI calls bounded: if the server hangs, fail quickly.
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, cmdArgs[0], cmdArgs[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = errOut

	if err := cmd.Start(); err != nil {
		return err
	}

	enc := json.NewEncoder(stdin)
	dec := json.NewDecoder(bufio.NewReader(stdout))

	// We don't need initialize for this minimal server; it accepts tools/call directly.
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  mustJSON(toolsCallParams{Name: "get_lines_by_id", Arguments: mustJSONRaw(map[string]any{"line_ids": lineIDs})}),
	}
	if err := enc.Encode(req); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return err
	}
	_ = stdin.Close() // end the server loop (it exits on EOF)

	var resp rpcResponse
	if err := dec.Decode(&resp); err != nil {
		_ = cmd.Wait()
		return err
	}
	if resp.Error != nil {
		_ = cmd.Wait()
		return fmt.Errorf("server error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	// MCP shim returns {"result":{"structured":<payload>,"content":[...]}}
	var structured GetLinesByIDResult
	if err := decodeStructured(resp.Result, &structured); err != nil {
		_ = cmd.Wait()
		return err
	}

	for _, l := range structured.Lines {
		// Single-line output is easiest to grep/copy.
		ts := l.Timestamp
		if ts == "" {
			ts = "-"
		}
		_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", l.LineID, ts, l.Source, strings.TrimRight(l.Message, "\n"))
	}
	if len(structured.Missing) > 0 {
		_, _ = fmt.Fprintf(errOut, "missing_line_ids: %s\n", strings.Join(structured.Missing, ", "))
	}

	return cmd.Wait()
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type GetLinesByIDResult struct {
	Count   int           `json:"count"`
	Lines   []SearchMatch `json:"lines"`
	Missing []string      `json:"missing_line_ids"`
}

type SearchMatch struct {
	LineID    string `json:"line_id"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

func decodeStructured(result any, dst any) error {
	// result is decoded as map[string]any; structured is nested under result.structured
	m, ok := result.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected result shape")
	}
	s, ok := m["structured"]
	if !ok {
		return fmt.Errorf("missing structured payload")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func mustJSONRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func splitCommand(s string) ([]string, error) {
	// Very small "split": whitespace separated, supports quoting with single or double quotes.
	// Good enough for env var usage without adding deps.
	var out []string
	var cur strings.Builder
	var quote rune

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			cur.WriteRune(r)
		default:
			if r == '\'' || r == '"' {
				quote = r
				continue
			}
			if r == ' ' || r == '\t' || r == '\n' {
				flush()
				continue
			}
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in command")
	}
	flush()
	if len(out) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return out, nil
}
