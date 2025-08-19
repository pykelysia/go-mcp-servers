package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

func (fs *FilesystemServer) registerDeleteFile() {
	fs.mcp.AddTool(
		mcp.NewTool("delete_file",
			mcp.WithDescription("Delete a file under FS_ROOT. Requires FS_ALLOW_DELETE=true. "+
				"Refuses to delete directories."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Path relative to FS_ROOT")),
		),
		fs.deleteFile,
	)
}

func (fs *FilesystemServer) deleteFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !fs.allowDelete {
		return mcp.NewToolResultError("delete disabled: set FS_ALLOW_DELETE=true to enable"), nil
	}
	path, _ := req.GetArguments()["path"].(string)
	full, err := fs.safePath(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	info, err := os.Stat(full)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("stat: %v", err)), nil
	}
	if info.IsDir() {
		return mcp.NewToolResultError(fmt.Sprintf("%s is a directory; refusing to delete", path)), nil
	}
	if err := os.Remove(full); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("delete: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("deleted %s", path)), nil
}
