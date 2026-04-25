package mcp

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/michalstefanec/trailhead/internal/coverage"
	"github.com/michalstefanec/trailhead/internal/query"
)

type server struct {
	startedAt time.Time
	session   *query.Session
}

func newServer() *server {
	return &server{
		startedAt: time.Now().UTC(),
		session:   query.NewSession(),
	}
}

func (s *server) handle(ctx context.Context, req rpcRequest) rpcResponse {
	// Keep a strict, structured surface; unknown methods should fail loudly.
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"serverInfo": map[string]any{
					"name":    "trailhead",
					"version": "0.0.0-dev",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}
	case "tools/list":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": []any{
					map[string]any{
						"name":        "search",
						"description": "Search logs with bounded, citeable results. Returns structured matches with stable session-scoped line IDs.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"query": map[string]any{"type": "string"},
								"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "default": 20},
								"source": map[string]any{
									"type":        "string",
									"description": "Backend source. v0 skeleton supports 'file'.",
									"default":     "file",
								},
								"filePath": map[string]any{
									"type":        "string",
									"description": "For source='file', path to a log file.",
								},
							},
							"required": []string{"query"},
						},
					},
					map[string]any{
						"name":        "get_lines_by_id",
						"description": "Fetch full log lines for previously returned LineIDs (session-scoped). Use this to let humans verify cited evidence.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"line_ids": map[string]any{
									"type":     "array",
									"minItems": 1,
									"maxItems": 200,
									"items":    map[string]any{"type": "string"},
								},
							},
							"required": []string{"line_ids"},
						},
					},
					map[string]any{
						"name":        "test_coverage",
						"description": "Run Go unit tests with coverage and return a structured summary (total percent + bounded per-file breakdown).",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"packages": map[string]any{
									"type":        "array",
									"description": "Go package patterns to test (passed to `go test`). Default: ['./...']",
									"items":       map[string]any{"type": "string"},
								},
								"top_files": map[string]any{
									"type":        "integer",
									"minimum":     0,
									"maximum":     100,
									"default":     10,
									"description": "Number of files to include in the per-file breakdown (bounded).",
								},
							},
						},
					},
				},
			},
		}
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32601,
				Message: "method not found",
			},
		}
	}
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func (s *server) handleToolsCall(ctx context.Context, req rpcRequest) rpcResponse {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid params"}}
	}

	switch p.Name {
	case "search":
		var args query.SearchArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		if args.Limit == 0 {
			args.Limit = 20
		}
		res, err := s.session.Search(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		// Minimal MCP tool result shape (text is optional; keep structured JSON primary).
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "ok",
					},
				},
				"structured": res,
			},
		}
	case "get_lines_by_id":
		var args query.GetLinesByIDArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		if len(args.LineIDs) == 0 {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "line_ids is required"}}
		}
		lines, missing, err := s.session.GetLinesByID(ctx, args.LineIDs)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		res := query.GetLinesByIDResult{
			Query: map[string]any{
				"line_ids": args.LineIDs,
			},
			Count:   len(lines),
			Lines:   lines,
			Missing: missing,
		}
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "ok",
					},
				},
				"structured": res,
			},
		}
	case "test_coverage":
		var args coverage.ToolArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		res, err := coverage.Run(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "ok",
					},
				},
				"structured": res,
			},
		}
	default:
		_, _ = os.Stderr.WriteString("unknown tool: " + p.Name + "\n")
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "tool not found"}}
	}
}

