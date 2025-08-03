package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestInit_DefaultsToInfo(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	l := Init("test-server")
	if l.GetLevel() != zerolog.InfoLevel {
		t.Errorf("want InfoLevel, got %v", l.GetLevel())
	}
}

func TestInit_DebugLevelFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	l := Init("test-server")
	if l.GetLevel() != zerolog.DebugLevel {
		t.Errorf("want DebugLevel, got %v", l.GetLevel())
	}
}

func TestInit_UnknownLevelDefaultsToInfo(t *testing.T) {
	t.Setenv("LOG_LEVEL", "garbage")
	l := Init("test-server")
	if l.GetLevel() != zerolog.InfoLevel {
		t.Errorf("want InfoLevel for unknown, got %v", l.GetLevel())
	}
}
