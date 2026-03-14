package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os/exec"
	"syscall"
	"time"
)

// SlashEvent is the JSON payload written to a worker's stdin.
type SlashEvent struct {
	Command     string `json:"command"`
	Text        string `json:"text"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

// runWorker starts script as a child process, writes event as JSON to its
// stdin, then waits for it to exit.
//
// On timeout the worker receives SIGTERM; if it does not exit within
// gracePeriod it receives SIGKILL.
func runWorker(ctx context.Context, script string, timeout time.Duration, event SlashEvent) {
	workerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(script) //nolint:gosec // script path comes from config, not user input
	// Place the child in its own process group so SIGTERM/SIGKILL hits
	// the whole subtree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("worker: stdin pipe failed", "command", event.Command, "script", script, "err", err)
		return
	}

	if err := cmd.Start(); err != nil {
		slog.Error("worker: start failed", "command", event.Command, "script", script, "err", err)
		return
	}

	pid := cmd.Process.Pid
	slog.Info("worker started", "pid", pid, "command", event.Command, "script", script, "user", event.UserID)

	// Write JSON payload to stdin then close, in a separate goroutine so it
	// does not block if the child is not reading.
	go func() {
		enc := json.NewEncoder(stdin)
		if err := enc.Encode(event); err != nil {
			slog.Warn("worker: stdin write error", "pid", pid, "err", err)
		}
		stdin.Close()
	}()

	// Collect the process exit asynchronously.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			slog.Error("worker abnormal exit", "pid", pid, "command", event.Command, "err", err)
		} else {
			slog.Info("worker exited normally", "pid", pid, "command", event.Command)
		}

	case <-workerCtx.Done():
		// Timeout hit (or parent context cancelled).
		reason := "timeout"
		if ctx.Err() != nil {
			reason = "shutdown"
		}
		slog.Warn("worker: sending SIGTERM", "pid", pid, "command", event.Command, "reason", reason)
		_ = syscall.Kill(-pid, syscall.SIGTERM)

		select {
		case <-done:
			slog.Info("worker: exited after SIGTERM", "pid", pid)
		case <-time.After(5 * time.Second):
			slog.Warn("worker: sending SIGKILL", "pid", pid)
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			<-done
			slog.Warn("worker: killed", "pid", pid)
		}
	}
}

// slackResponse is the payload sent to a Slack response_url.
type slackResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

// notifyEphemeral posts a message visible only to the requesting user.
// response_type "ephemeral" ensures the message is not shown to the channel.
func notifyEphemeral(responseURL, message string) {
	if responseURL == "" {
		return
	}
	body, _ := json.Marshal(slackResponse{
		ResponseType: "ephemeral",
		Text:         message,
	})
	resp, err := http.Post(responseURL, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		slog.Warn("notifyEphemeral: http post failed", "err", err)
		return
	}
	resp.Body.Close()
}

