package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

func (fs *FilesystemServer) registerCreateDirectory() {
	fs.mcp.AddTool(
		mcp.NewTool("create_directory",
			mcp.WithDescription("Create a directory under FS_ROOT, including parent directories. "+
				"Succeeds if the directory already exists."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Directory relative to FS_ROOT")),
		),
		fs.createDirectory,
	)
}

func (fs *FilesystemServer) createDirectory(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, _ := req.GetArguments()["path"].(string)
	full, err := fs.safePath(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("mkdir: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("created %s", path)), nil
}
