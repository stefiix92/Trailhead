package mcp

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/stefiix92/Trailhead/internal/config"
	"github.com/stefiix92/Trailhead/internal/coverage"
	"github.com/stefiix92/Trailhead/internal/query"
	"github.com/stefiix92/Trailhead/internal/version"
)

type server struct {
	startedAt time.Time
	session   *query.Session
	cfg       *config.Config
}

func newServer(cfg *config.Config) *server {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &server{
		startedAt: time.Now().UTC(),
		session:   query.NewSessionWithConfig(cfg),
		cfg:       cfg,
	}
}

func (s *server) handle(ctx context.Context, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"serverInfo": map[string]any{
					"name":    "trailhead",
					"version": version.Version,
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
				"tools": s.toolList(),
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

func (s *server) toolList() []any {
	tools := []any{
		searchToolDef(),
		summarizeErrorsToolDef(),
		sampleClusterToolDef(),
		correlatedEventsToolDef(),
		diffErrorRateToolDef(),
		getLinesByIDToolDef(),
	}
	if s.cfg != nil && s.cfg.DevTools {
		tools = append(tools, testCoverageToolDef())
	}
	return tools
}

func searchToolDef() map[string]any {
	return map[string]any{
		"name": "search",
		"description": "Search logs with bounded, citeable results. Returns structured matches with stable session-scoped line IDs " +
			"(source-prefixed: file:N, loki:N, docker:N, journald:N). When reporting findings, cite line_id values from matches. " +
			"If a claim is not supported by a returned line_id, state that explicitly. " +
			"For Loki, set source=loki and lokiService or lokiLogql, with since/start/end to bound time. " +
			"For Docker, set source=docker and dockerContainer or dockerComposeService (optional time bounds apply if timestamps exist). " +
			"For journald, set source=journald and optionally journaldUnit or journaldIdentifier.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":  map[string]any{"type": "string", "description": "Substring filter (file) or additional |= filter (loki) when lokiLogql is not set."},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "default": 20},
				"source": map[string]any{"type": "string", "enum": []string{"file", "loki", "docker", "journald"}, "default": "file"},
				"filePath": map[string]any{
					"type":        "string",
					"description": "For source=file, path to a log file.",
				},
				"dockerContainer":       map[string]any{"type": "string", "description": "For source=docker, container name or id for `docker logs`."},
				"dockerComposeService":  map[string]any{"type": "string", "description": "For source=docker, docker compose service name (resolved via `docker compose ps -q`)."},
				"dockerComposeProject":  map[string]any{"type": "string", "description": "Optional: docker compose project name (passed as --project-name)."},
				"dockerComposeFilePath": map[string]any{"type": "string", "description": "Optional: compose file path (passed as -f)."},
				"journaldUnit":          map[string]any{"type": "string", "description": "For source=journald, systemd unit (e.g. nginx.service)."},
				"journaldIdentifier":    map[string]any{"type": "string", "description": "For source=journald, SYSLOG_IDENTIFIER match (journalctl field filter)."},
				"lokiLogql": map[string]any{
					"type":        "string",
					"description": "For source=loki, full LogQL. If empty, built from lokiService/lokiStreamSelector and query.",
				},
				"lokiService":        map[string]any{"type": "string", "description": "Loki 'service' label value for {service=...} when lokiLogql is empty."},
				"lokiStreamSelector": map[string]any{"type": "string", "description": "Raw stream selector, e.g. {app=\"api\"}."},
				"since":              map[string]any{"type": "string", "description": "Duration before end, e.g. 30m, 1h (Loki and time-bounded use)."},
				"start":              map[string]any{"type": "string", "description": "RFC3339 start (optional)."},
				"end":                map[string]any{"type": "string", "description": "RFC3339 end (optional, default now)."},
				"until":              map[string]any{"type": "string", "description": "RFC3339 synonym for end."},
				"filterErrors":       map[string]any{"type": "boolean", "description": "For file/docker/journald sources, only return lines that look like errors."},
			},
		},
	}
}

func summarizeErrorsToolDef() map[string]any {
	return map[string]any{
		"name": "summarize_errors",
		"description": "Fetch error-like log lines, cluster with TF–IDF similarity, and return counts, coverage, cluster ids, and representative line_ids. " +
			"When reporting, cite line IDs from representative_line_ids or line_ids. Label uncited inferences as hypotheses.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source":                map[string]any{"type": "string", "enum": []string{"file", "loki", "docker", "journald"}, "default": "file"},
				"filePath":              map[string]any{"type": "string"},
				"dockerContainer":       map[string]any{"type": "string"},
				"dockerComposeService":  map[string]any{"type": "string"},
				"dockerComposeProject":  map[string]any{"type": "string"},
				"dockerComposeFilePath": map[string]any{"type": "string"},
				"journaldUnit":          map[string]any{"type": "string"},
				"journaldIdentifier":    map[string]any{"type": "string"},
				"lokiService":           map[string]any{"type": "string"},
				"lokiStreamSelector":    map[string]any{"type": "string"},
				"lokiLogql":             map[string]any{"type": "string", "description": "Full error-scoped LogQL; overrides heuristics if set."},
				"since":                 map[string]any{"type": "string"},
				"start":                 map[string]any{"type": "string"},
				"end":                   map[string]any{"type": "string"},
				"until":                 map[string]any{"type": "string"},
				"maxLines":              map[string]any{"type": "integer", "default": 2000, "description": "Max lines to pull before clustering."},
			},
		},
	}
}

func sampleClusterToolDef() map[string]any {
	return map[string]any{
		"name":        "sample_cluster",
		"description": "After summarize_errors, fetch up to n full records for a cluster_id. Cite line_ids from the response.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "string"},
				"n":          map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "default": 5},
			},
			"required": []string{"cluster_id"},
		},
	}
}

func correlatedEventsToolDef() map[string]any {
	return map[string]any{
		"name": "correlated_events",
		"description": "List citeable log lines in a time window around an RFC3339 instant. Optional deploy markers from TRAILHEAD_MARKERS_FILE are included. " +
			"Cite line_id values for any log claims.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"around":                map[string]any{"type": "string", "description": "Center time RFC3339."},
				"window":                map[string]any{"type": "string", "description": "Total span (go duration e.g. 2m, 1h), default 2m."},
				"source":                map[string]any{"type": "string", "enum": []string{"file", "loki", "docker", "journald"}},
				"filePath":              map[string]any{"type": "string"},
				"dockerContainer":       map[string]any{"type": "string"},
				"dockerComposeService":  map[string]any{"type": "string"},
				"dockerComposeProject":  map[string]any{"type": "string"},
				"dockerComposeFilePath": map[string]any{"type": "string"},
				"journaldUnit":          map[string]any{"type": "string"},
				"journaldIdentifier":    map[string]any{"type": "string"},
				"lokiService":           map[string]any{"type": "string"},
				"lokiStreamSelector":    map[string]any{"type": "string"},
				"limit":                 map[string]any{"type": "integer", "default": 100},
			},
			"required": []string{"around"},
		},
	}
}

func diffErrorRateToolDef() map[string]any {
	return map[string]any{
		"name": "diff_error_rate",
		"description": "Compare counts of error-like lines between time window A (current) and B (defaults to same duration immediately before A). " +
			"Requires source=loki. Cite any line-level claims with line_ids from other tools, not this aggregate alone.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source":             map[string]any{"type": "string", "enum": []string{"loki"}},
				"filePath":           map[string]any{"type": "string"},
				"lokiService":        map[string]any{"type": "string"},
				"lokiStreamSelector": map[string]any{"type": "string"},
				"a_since":            map[string]any{"type": "string", "description": "Duration of window A before its end (if a_start not set)."},
				"b_since":            map[string]any{"type": "string", "description": "Optional: duration of baseline window if b_start not set; else ignored."},
				"a_start":            map[string]any{"type": "string"},
				"a_end":              map[string]any{"type": "string"},
				"b_start":            map[string]any{"type": "string"},
				"b_end":              map[string]any{"type": "string"},
			},
		},
	}
}

func getLinesByIDToolDef() map[string]any {
	return map[string]any{
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
	}
}

func testCoverageToolDef() map[string]any {
	return map[string]any{
		"name":        "test_coverage",
		"description": "Developer-only: run Go unit tests with coverage (enable TRAILHEAD_DEV_TOOLS=1 on the server).",
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
		return structuredResponse(req, res)
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
		res, err := s.session.GetLinesByID(ctx, args.LineIDs)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return structuredResponse(req, res)
	case "summarize_errors":
		var args query.SummarizeErrorsArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		res, err := s.session.SummarizeErrors(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return structuredResponse(req, res)
	case "sample_cluster":
		var args query.SampleClusterArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		res, err := s.session.SampleCluster(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return structuredResponse(req, res)
	case "correlated_events":
		var args query.CorrelatedEventsArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		res, err := s.session.CorrelatedEvents(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return structuredResponse(req, res)
	case "diff_error_rate":
		var args query.DiffErrorRateArgs
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}
			}
		}
		if args.Source == "" {
			args.Source = "loki"
		}
		res, err := s.session.DiffErrorRate(ctx, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
		}
		return structuredResponse(req, res)
	case "test_coverage":
		if s.cfg == nil || !s.cfg.DevTools {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "test_coverage is disabled; set TRAILHEAD_DEV_TOOLS=1"}}
		}
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
		return structuredResponse(req, res)
	default:
		_, _ = os.Stderr.WriteString("unknown tool: " + p.Name + "\n")
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "tool not found"}}
	}
}

func structuredResponse(req rpcRequest, v any) rpcResponse {
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
			"structured": v,
		},
	}
}
