package main

import (
	"testing"
)

func TestNewPostgresServer_RequiresDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "")
	if _, err := NewPostgresServer(); err == nil {
		t.Fatal("expected error for empty POSTGRES_DSN")
	}
}

func TestNewPostgresServer_RejectsUnreachable(t *testing.T) {
	// Use a port nothing listens on; ping should fail in <5s.
	t.Setenv("POSTGRES_DSN", "postgres://nouser:nopass@127.0.0.1:1/none?sslmode=disable&connect_timeout=2")
	if _, err := NewPostgresServer(); err == nil {
		t.Fatal("expected ping error against unreachable host")
	}
}

func TestValidIdent(t *testing.T) {
	cases := map[string]bool{
		"users":          true,
		"_users":         true,
		"users_1":        true,
		"public.users":   true,
		"my_schema.tbl1": true,
		"":               false,
		".bad":           false,
		"bad.":           false,
		"1users":         false,
		"a.b.c":          false,
		"users; DROP TABLE x":         false,
		"users' OR '1'='1":            false,
		"users WHERE 1=1":             false,
	}
	for in, want := range cases {
		if got := validIdent(in); got != want {
			t.Errorf("validIdent(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalize_BytesToString(t *testing.T) {
	if got := normalize([]byte("hi")); got != "hi" {
		t.Errorf("want string conversion, got %#v", got)
	}
	if got := normalize(int64(42)); got != int64(42) {
		t.Errorf("want passthrough, got %#v", got)
	}
}
