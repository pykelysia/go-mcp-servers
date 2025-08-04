// Package testutil provides MCP client helpers for in-process server tests.
package testutil

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewInProcessClient builds an MCP client wired to s in the same process,
// starts it, and runs the initialize handshake. Returns a ready-to-use
// client; the caller need not call Start or Initialize.
func NewInProcessClient(t *testing.T, s *server.MCPServer) *client.Client {
	t.Helper()
	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	req := mcp.InitializeRequest{}
	req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	req.Params.ClientInfo = mcp.Implementation{Name: "testutil", Version: "0.0.0"}
	if _, err := c.Initialize(ctx, req); err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}
	return c
}

// CallTool invokes the named tool with the given arguments and returns the
// first text content. Fails the test on transport error or a non-text result.
// For error-result assertions use CallToolRaw.
func CallTool(t *testing.T, c *client.Client, name string, args map[string]any) string {
	t.Helper()
	result := CallToolRaw(t, c, name, args)
	if result.IsError {
		t.Fatalf("CallTool(%s) returned IsError=true: %v", name, result.Content)
	}
	return firstText(t, result)
}

// CallToolRaw is the non-failing variant; use it to assert IsError=true.
func CallToolRaw(t *testing.T, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool(%s) transport error: %v", name, err)
	}
	return result
}

func firstText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		return ""
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("first content is not TextContent: %T", result.Content[0])
	}
	return text.Text
}
