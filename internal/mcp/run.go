package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/stefiix92/Trailhead/internal/config"
)

// NOTE: This is a minimal stdio JSON-RPC-ish shim to get a working skeleton in place
// without committing to a particular MCP Go SDK yet. We’ll swap this for a proper MCP
// implementation once we pick the SDK.
//
// Supported:
// - initialize
// - tools/list
// - tools/call (all core log tools; optional test_coverage with TRAILHEAD_DEV_TOOLS=1)
//
// Everything else returns method-not-found.

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

func Run(ctx context.Context, cfg *config.Config, in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)

	s := newServer(cfg)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		resp := s.handle(ctx, req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}
