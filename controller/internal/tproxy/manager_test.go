package tproxy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerRunsOptionalLifecycleScript(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "actions.log")
	scriptPath := filepath.Join(dir, "tproxy.sh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"" + logPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	m := New(Config{Enabled: true, Script: scriptPath})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(content)); got != "start\nstop" {
		t.Fatalf("actions = %q, want %q", got, "start\\nstop")
	}
}

func TestManagerDisabledDoesNotRunScript(t *testing.T) {
	m := New(Config{Script: filepath.Join(t.TempDir(), "missing.sh")})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("disabled Start() error = %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("disabled Stop() error = %v", err)
	}
}

func TestManagerStopCleansExistingStateWithoutStart(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "actions.log")
	scriptPath := filepath.Join(dir, "tproxy.sh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"" + logPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	m := New(Config{Enabled: true, Script: scriptPath})
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(content)); got != "stop" {
		t.Fatalf("actions = %q, want %q", got, "stop")
	}
}
