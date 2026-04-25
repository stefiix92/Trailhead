package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/michalstefanec/trailhead/internal/config"
)

func TestRun_initializeAndToolsList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := bytes.NewBufferString("" +
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	var out bytes.Buffer

	if err := Run(ctx, &config.Config{}, in, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))

	var r1 rpcResponse
	if err := dec.Decode(&r1); err != nil {
		t.Fatalf("decode r1: %v", err)
	}
	if r1.Error != nil {
		t.Fatalf("unexpected error: %+v", r1.Error)
	}
	if r1.Result == nil {
		t.Fatalf("expected result")
	}

	var r2 rpcResponse
	if err := dec.Decode(&r2); err != nil {
		t.Fatalf("decode r2: %v", err)
	}
	if r2.Error != nil {
		t.Fatalf("unexpected error: %+v", r2.Error)
	}
	m, ok := r2.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", r2.Result)
	}
	tools, ok := m["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected non-empty tools list")
	}
}

func TestRun_methodNotFound(t *testing.T) {
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":"x","method":"nope"}` + "\n")
	var out bytes.Buffer

	if err := Run(context.Background(), &config.Config{}, in, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected method not found, got %+v", resp.Error)
	}
}

func TestHandleToolsCall_invalidParams(t *testing.T) {
	s := newServer(&config.Config{})
	resp := s.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`not-valid-json`),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected invalid params error, got %+v", resp.Error)
	}
}

func TestHandleToolsCall_unknownTool(t *testing.T) {
	s := newServer(&config.Config{})
	resp := s.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"does_not_exist","arguments":{}}`),
	})
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected tool not found error, got %+v", resp.Error)
	}
}

func TestHandleToolsCall_testCoverage_disabledByDefault(t *testing.T) {
	s := newServer(&config.Config{DevTools: false})
	resp := s.handle(context.Background(), rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_coverage","arguments":{}}`),
	})
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected disabled tool error, got %+v", resp.Error)
	}
}

