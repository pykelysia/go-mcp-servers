package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (fsrv *FilesystemServer) registerFindInFiles() {
	fsrv.mcp.AddTool(
		mcp.NewTool("find_in_files",
			mcp.WithDescription("Grep-like text search across files under a directory. "+
				"Returns 'path:line: matching content'. Skips files larger than FS_MAX_FILE_SIZE."),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Text to search for (case-sensitive)")),
			mcp.WithString("path", mcp.Description("Directory under FS_ROOT to search. Defaults to root.")),
		),
		fsrv.findInFiles,
	)
}

func (fsrv *FilesystemServer) findInFiles(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return mcp.NewToolResultError("pattern is required"), nil
	}
	searchPath, _ := args["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}
	full, err := fsrv.safePath(searchPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	const maxMatches = 500
	var results []string
	err = filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(results) >= maxMatches {
			return filepath.SkipAll
		}
		info, err := d.Info()
		if err != nil || info.Size() > fsrv.maxSize {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(fsrv.root, p)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, pattern) {
				results = append(results, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				if len(results) >= maxMatches {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("walk: %v", err)), nil
	}
	if len(results) == 0 {
		return mcp.NewToolResultText("(no matches found)"), nil
	}
	out := strings.Join(results, "\n")
	if len(results) >= maxMatches {
		out += fmt.Sprintf("\n[truncated at %d matches]", maxMatches)
	}
	return mcp.NewToolResultText(out), nil
}
