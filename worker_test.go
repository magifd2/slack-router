package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateResponseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty url", "", true},
		{"valid slack url", "https://hooks.slack.com/commands/T123/456/abc", false},
		{"http scheme rejected", "http://hooks.slack.com/commands/T123/456/abc", true},
		{"wrong host", "https://hooks.example.com/commands/T123/456/abc", true},
		{"wrong host - slack.com only", "https://slack.com/commands/T123/456/abc", true},
		{"ftp scheme", "ftp://hooks.slack.com/commands/T123/456/abc", true},
		{"no scheme", "hooks.slack.com/commands/T123/456/abc", true},
		{"file scheme", "file:///etc/passwd", true},
		{"internal address", "https://169.254.169.254/latest/meta-data/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResponseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateResponseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizedEnv(t *testing.T) {
	t.Setenv("SLACK_APP_TOKEN", "xapp-test-token")
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	t.Setenv("HOME", "/home/testuser")

	result := sanitizedEnv()

	for _, kv := range result {
		key, _, _ := strings.Cut(kv, "=")
		if key == "SLACK_APP_TOKEN" {
			t.Error("sanitizedEnv: SLACK_APP_TOKEN must not appear in output")
		}
		if key == "SLACK_BOT_TOKEN" {
			t.Error("sanitizedEnv: SLACK_BOT_TOKEN must not appear in output")
		}
	}

	// Non-sensitive vars must be preserved.
	found := false
	for _, kv := range result {
		if strings.HasPrefix(kv, "HOME=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("sanitizedEnv: HOME should be preserved in output")
	}
}

func TestSanitizedEnvNoSensitiveKeys(t *testing.T) {
	// Ensure no sensitive keys leak even if the env has only those vars.
	os.Unsetenv("SLACK_APP_TOKEN")
	os.Unsetenv("SLACK_BOT_TOKEN")

	result := sanitizedEnv()
	for _, kv := range result {
		key, _, _ := strings.Cut(kv, "=")
		if _, blocked := sensitiveEnvKeys[key]; blocked {
			t.Errorf("sanitizedEnv: sensitive key %q found in output", key)
		}
	}
}
