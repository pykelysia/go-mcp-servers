package testutil

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func newEchoServer(t *testing.T) *server.MCPServer {
	t.Helper()
	s := server.NewMCPServer("echo", "1.0.0", server.WithToolCapabilities(true))
	s.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("Echo back the provided text"),
			mcp.WithString("text", mcp.Required()),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			txt, _ := args["text"].(string)
			if txt == "" {
				return mcp.NewToolResultError("text required"), nil
			}
			return mcp.NewToolResultText(txt), nil
		},
	)
	return s
}

func TestCallTool_HappyPath(t *testing.T) {
	s := newEchoServer(t)
	c := NewInProcessClient(t, s)
	got := CallTool(t, c, "echo", map[string]any{"text": "hello"})
	if got != "hello" {
		t.Errorf("want %q, got %q", "hello", got)
	}
}

func TestCallToolRaw_CapturesErrorResult(t *testing.T) {
	s := newEchoServer(t)
	c := NewInProcessClient(t, s)
	result := CallToolRaw(t, c, "echo", map[string]any{"text": ""})
	if !result.IsError {
		t.Fatal("expected IsError=true for empty text")
	}
}
