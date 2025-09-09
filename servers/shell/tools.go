package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools wires the three shell tools.
func (s *ShellServer) registerTools() {
	s.registerRunCommand()
	s.registerRunScript()
	s.registerGetEnv()
}

type runResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

func (s *ShellServer) registerRunCommand() {
	s.mcp.AddTool(
		mcp.NewTool("run_command",
			mcp.WithDescription("Execute a command via 'bash -c' inside SHELL_WORKDIR. "+
				"Returns JSON with stdout, stderr, exit_code. Times out after SHELL_TIMEOUT seconds."),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command string passed to bash -c")),
		),
		s.handleRunCommand,
	)
}

func (s *ShellServer) handleRunCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd, _ := req.GetArguments()["command"].(string)
	if msg, ok := s.preflight(cmd); !ok {
		return mcp.NewToolResultError(msg), nil
	}
	res := s.runBash(ctx, cmd)
	body, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(body)), nil
}

func (s *ShellServer) registerRunScript() {
	s.mcp.AddTool(
		mcp.NewTool("run_script",
			mcp.WithDescription("Write a script body to a temporary file inside SHELL_WORKDIR and execute it "+
				"with the configured interpreter (default 'bash'). Same timeout and output cap as run_command."),
			mcp.WithString("script", mcp.Required(), mcp.Description("Script body to write")),
			mcp.WithString("interpreter", mcp.Description("Interpreter to invoke. Default 'bash'.")),
		),
		s.handleRunScript,
	)
}

func (s *ShellServer) handleRunScript(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	body, _ := args["script"].(string)
	interp, _ := args["interpreter"].(string)
	if interp == "" {
		interp = "bash"
	}
	if msg, ok := s.preflight(interp); !ok {
		return mcp.NewToolResultError("interpreter rejected: " + msg), nil
	}
	if body == "" {
		return mcp.NewToolResultError("script body is required"), nil
	}

	tmp, err := os.CreateTemp(s.workdir, "mcp-script-*.sh")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("tempfile: %v", err)), nil
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		return mcp.NewToolResultError(fmt.Sprintf("write script: %v", err)), nil
	}
	if err := tmp.Close(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("close script: %v", err)), nil
	}
	if err := os.Chmod(tmp.Name(), 0o700); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("chmod: %v", err)), nil
	}

	res := s.runExec(ctx, interp, []string{tmp.Name()})
	out, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func (s *ShellServer) registerGetEnv() {
	s.mcp.AddTool(
		mcp.NewTool("get_env",
			mcp.WithDescription("Return the value of an environment variable, but only if its name appears in SHELL_ENV_PASSTHROUGH."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Variable name")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, _ := req.GetArguments()["name"].(string)
			allowed := false
			for _, n := range s.envPassthrough {
				if n == name {
					allowed = true
					break
				}
			}
			if !allowed {
				return mcp.NewToolResultError(fmt.Sprintf("%s not in SHELL_ENV_PASSTHROUGH", name)), nil
			}
			return mcp.NewToolResultText(os.Getenv(name)), nil
		},
	)
}

func (s *ShellServer) runBash(ctx context.Context, command string) runResult {
	return s.runExec(ctx, "bash", []string{"-c", command})
}

func (s *ShellServer) runExec(ctx context.Context, name string, args []string) runResult {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeoutSeconds)*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, name, args...)
	c.Dir = s.workdir
	c.Env = s.buildEnv()

	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	res := runResult{
		Stdout: truncate(stripAnsi(stdout.String()), s.maxOutputBytes),
		Stderr: truncate(stripAnsi(stderr.String()), s.maxOutputBytes),
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = 124
		return res
	}
	if err != nil {
		var exitErr *exec.ExitError
		if asExit(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.Stderr += "\n[exec error: " + err.Error() + "]"
			res.ExitCode = 1
		}
	}
	return res
}

// asExit is a tiny errors.As wrapper that lets handleRunCommand stay readable.
func asExit(err error, target **exec.ExitError) bool {
	for cur := err; cur != nil; {
		if e, ok := cur.(*exec.ExitError); ok {
			*target = e
			return true
		}
		type unwrap interface{ Unwrap() error }
		u, ok := cur.(unwrap)
		if !ok {
			return false
		}
		cur = u.Unwrap()
	}
	return false
}

// Silence unused import when running tests with limited features.
var _ = filepath.Join
