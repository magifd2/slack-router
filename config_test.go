package main

import (
	"os"
	"path/filepath"
	"testing"
)

// makeScript creates a temp file with the given permission bits and returns its path.
func makeScript(t *testing.T, mode os.FileMode) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "script-*.sh")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	f.Close()
	if err := os.Chmod(f.Name(), mode); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	return f.Name()
}

func TestValidateScript(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns path
		wantErr bool
	}{
		{
			name: "valid executable script",
			setup: func(t *testing.T) string {
				return makeScript(t, 0o755)
			},
			wantErr: false,
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.sh")
			},
			wantErr: true,
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		{
			name: "not executable",
			setup: func(t *testing.T) string {
				return makeScript(t, 0o644)
			},
			wantErr: true,
		},
		{
			name: "world-writable",
			setup: func(t *testing.T) string {
				return makeScript(t, 0o777)
			},
			wantErr: true,
		},
		{
			name: "executable but world-writable",
			setup: func(t *testing.T) string {
				return makeScript(t, 0o755|0o002)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			err := validateScript("/test", path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Helper to write a minimal config YAML with a real executable script.
	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return path
	}

	makeExecScript := func(t *testing.T, dir string) string {
		t.Helper()
		p := filepath.Join(dir, "worker.sh")
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return p
	}

	t.Run("missing app token", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "")
		t.Setenv("SLACK_BOT_TOKEN", "")
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		_ = os.WriteFile(cfgPath, []byte(`
slack:
  app_token: ""
  bot_token: ""
`), 0o644)
		_, err := LoadConfig(cfgPath)
		if err == nil {
			t.Error("expected error for missing app token")
		}
	})

	t.Run("tokens from env override yaml", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "xapp-from-env")
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-from-env")

		dir := t.TempDir()
		scriptPath := makeExecScript(t, dir)
		cfgPath := writeConfig(t, `
slack:
  app_token: ""
  bot_token: ""
global:
  max_concurrent_workers: 5
routes:
  - command: /hello
    script: `+scriptPath+`
`)
		cfg, err := LoadConfig(cfgPath)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.Slack.AppToken != "xapp-from-env" {
			t.Errorf("AppToken = %q, want xapp-from-env", cfg.Slack.AppToken)
		}
		if cfg.Slack.BotToken != "xoxb-from-env" {
			t.Errorf("BotToken = %q, want xoxb-from-env", cfg.Slack.BotToken)
		}
	})

	t.Run("defaults applied when omitted", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")

		dir := t.TempDir()
		scriptPath := makeExecScript(t, dir)
		cfgPath := writeConfig(t, `
routes:
  - command: /hello
    script: `+scriptPath+`
`)
		cfg, err := LoadConfig(cfgPath)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.Global.MaxConcurrentWorkers != 10 {
			t.Errorf("MaxConcurrentWorkers = %d, want 10", cfg.Global.MaxConcurrentWorkers)
		}
		if cfg.Global.LogLevel != "info" {
			t.Errorf("LogLevel = %q, want \"info\"", cfg.Global.LogLevel)
		}
		if cfg.Routes[0].Timeout == 0 {
			t.Error("default timeout should be non-zero")
		}
		if cfg.Routes[0].BusyMessage == "" {
			t.Error("BusyMessage default should be non-empty")
		}
		if cfg.Routes[0].DenyMessage == "" {
			t.Error("DenyMessage default should be non-empty")
		}
	})

	t.Run("duplicate command rejected", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")

		dir := t.TempDir()
		scriptPath := makeExecScript(t, dir)
		cfgPath := writeConfig(t, `
routes:
  - command: /hello
    script: `+scriptPath+`
  - command: /hello
    script: `+scriptPath+`
`)
		_, err := LoadConfig(cfgPath)
		if err == nil {
			t.Error("expected error for duplicate command")
		}
	})

	t.Run("relative script path resolved to config dir", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")

		dir := t.TempDir()
		_ = makeExecScript(t, dir) // creates dir/worker.sh

		cfgPath := filepath.Join(dir, "config.yaml")
		_ = os.WriteFile(cfgPath, []byte(`
routes:
  - command: /hello
    script: worker.sh
`), 0o644)

		cfg, err := LoadConfig(cfgPath)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !filepath.IsAbs(cfg.Routes[0].Script) {
			t.Errorf("Script path %q should be absolute", cfg.Routes[0].Script)
		}
	})

	t.Run("invalid timeout rejected", func(t *testing.T) {
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")

		dir := t.TempDir()
		scriptPath := makeExecScript(t, dir)
		cfgPath := writeConfig(t, `
routes:
  - command: /hello
    script: `+scriptPath+`
    timeout: notaduration
`)
		_, err := LoadConfig(cfgPath)
		if err == nil {
			t.Error("expected error for invalid timeout")
		}
	})

	t.Run("nonexistent config file", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("expected error for missing config file")
		}
	})
}
