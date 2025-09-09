package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dimasd-angga/go-mcp-servers/shared/testutil"
	"github.com/mark3labs/mcp-go/client"
)

func newClient(t *testing.T) *client.Client {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SHELL_WORKDIR", dir)
	t.Setenv("SHELL_TIMEOUT", "5")
	t.Setenv("SHELL_ALLOWED_CMDS", "echo,ls,pwd,sleep,bash,sh,cat,true,false")
	s, err := NewShellServer()
	if err != nil {
		t.Fatal(err)
	}
	return testutil.NewInProcessClient(t, s.MCP())
}

func decode(t *testing.T, s string) runResult {
	t.Helper()
	var r runResult
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return r
}

func TestRunCommand_HappyPath(t *testing.T) {
	c := newClient(t)
	out := testutil.CallTool(t, c, "run_command", map[string]any{"command": "echo from-mcp"})
	res := decode(t, out)
	if !strings.Contains(res.Stdout, "from-mcp") {
		t.Errorf("missing stdout: %+v", res)
	}
	if res.ExitCode != 0 {
		t.Errorf("want exit 0, got %d", res.ExitCode)
	}
}

func TestRunCommand_StderrAndExit(t *testing.T) {
	c := newClient(t)
	out := testutil.CallTool(t, c, "run_command", map[string]any{"command": "bash -c 'echo oops >&2; exit 3'"})
	res := decode(t, out)
	if !strings.Contains(res.Stderr, "oops") {
		t.Errorf("missing stderr: %+v", res)
	}
	if res.ExitCode != 3 {
		t.Errorf("want exit 3, got %d", res.ExitCode)
	}
}

func TestRunCommand_RejectsDangerous(t *testing.T) {
	c := newClient(t)
	r := testutil.CallToolRaw(t, c, "run_command", map[string]any{"command": "rm -rf /"})
	if !r.IsError {
		t.Error("must reject rm -rf /")
	}
}

func TestRunCommand_RejectsNotAllowed(t *testing.T) {
	c := newClient(t)
	r := testutil.CallToolRaw(t, c, "run_command", map[string]any{"command": "curl https://example.com"})
	if !r.IsError {
		t.Error("must reject command outside allowlist")
	}
}

func TestRunCommand_Timeout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELL_WORKDIR", dir)
	t.Setenv("SHELL_TIMEOUT", "1")
	t.Setenv("SHELL_ALLOWED_CMDS", "sleep,bash")
	s, err := NewShellServer()
	if err != nil {
		t.Fatal(err)
	}
	c := testutil.NewInProcessClient(t, s.MCP())
	out := testutil.CallTool(t, c, "run_command", map[string]any{"command": "sleep 5"})
	res := decode(t, out)
	if !res.TimedOut {
		t.Errorf("expected TimedOut=true: %+v", res)
	}
	if res.ExitCode != 124 {
		t.Errorf("want exit 124 for timeout, got %d", res.ExitCode)
	}
}

func TestRunScript_HappyPath(t *testing.T) {
	c := newClient(t)
	out := testutil.CallTool(t, c, "run_script", map[string]any{
		"script": "#!/usr/bin/env bash\necho scripted",
	})
	res := decode(t, out)
	if !strings.Contains(res.Stdout, "scripted") {
		t.Errorf("missing stdout: %+v", res)
	}
}

func TestRunScript_EmptyBody(t *testing.T) {
	c := newClient(t)
	r := testutil.CallToolRaw(t, c, "run_script", map[string]any{"script": ""})
	if !r.IsError {
		t.Error("empty script should fail")
	}
}

func TestRunScript_RejectsBadInterpreter(t *testing.T) {
	c := newClient(t)
	r := testutil.CallToolRaw(t, c, "run_script", map[string]any{
		"script":      "echo hi",
		"interpreter": "perl",
	})
	if !r.IsError {
		t.Error("perl not in allowlist; must reject")
	}
}

func TestGetEnv_AllowedAndDenied(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELL_WORKDIR", dir)
	t.Setenv("SHELL_ENV_PASSTHROUGH", "PATH,FOO")
	t.Setenv("FOO", "bar")
	t.Setenv("SECRET", "x")
	s, err := NewShellServer()
	if err != nil {
		t.Fatal(err)
	}
	c := testutil.NewInProcessClient(t, s.MCP())
	got := testutil.CallTool(t, c, "get_env", map[string]any{"name": "FOO"})
	if got != "bar" {
		t.Errorf("want bar, got %q", got)
	}
	r := testutil.CallToolRaw(t, c, "get_env", map[string]any{"name": "SECRET"})
	if !r.IsError {
		t.Error("SECRET should be rejected")
	}
}
