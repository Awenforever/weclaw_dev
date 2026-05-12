package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestCleanAgentErrorTextStripsANSIAndFormatsBubblewrap(t *testing.T) {
	raw := "\x1b[2m2026-05-12T16:25:20Z\x1b[0m \x1b[31mERROR\x1b[0m \x1b[2mcodex_app_server\x1b[0m: Codex could not find bubblewrap on PATH. Codex will use the bundled bubblewrap in the meantime."
	got := cleanAgentErrorText(raw)
	if strings.Contains(got, "\x1b") || strings.Contains(got, "[31m") {
		t.Fatalf("cleanAgentErrorText kept ANSI escapes: %q", got)
	}
	for _, want := range []string{"Codex sandbox dependency warning", "sudo apt install -y bubblewrap", "restart WeClaw"} {
		if !strings.Contains(got, want) {
			t.Fatalf("cleanAgentErrorText = %q, missing %q", got, want)
		}
	}
}

func TestACPStderrWriterKeepsBubblewrapAdviceAcrossWrappedLines(t *testing.T) {
	w := &acpStderrWriter{prefix: "[test]"}
	_, _ = w.Write([]byte("\x1b[31mERROR\x1b[0m codex_app_server: Codex could not find bubblewrap on PATH.\npackage manager. See sandbox prerequisites.\n"))
	got := w.LastError()
	if strings.Contains(got, "\x1b") || strings.Contains(got, "[31m") {
		t.Fatalf("LastError kept ANSI escapes: %q", got)
	}
	if !strings.Contains(got, "Codex sandbox dependency warning") || !strings.Contains(got, "sudo apt install -y bubblewrap") {
		t.Fatalf("LastError = %q, want bubblewrap advice", got)
	}
}

func TestACPAgentRPCBubblewrapErrorIsUserReadable(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true
	a.stderr = &acpStderrWriter{prefix: "[test]"}

	_, _ = a.stderr.Write([]byte("\x1b[31mERROR\x1b[0m codex_app_server: Codex could not find bubblewrap on PATH. Codex will use the bundled bubblewrap in the meantime.\n"))

	got := cleanAgentErrorText(a.stderr.LastError())
	if strings.Contains(got, "\x1b") || strings.Contains(got, "[31m") {
		t.Fatalf("stderr-derived error kept ANSI escapes: %q", got)
	}
	if !strings.Contains(got, "Codex sandbox dependency warning") || !strings.Contains(got, "sudo apt install -y bubblewrap") {
		t.Fatalf("stderr-derived error = %q, want bubblewrap advice", got)
	}
}

func TestACPAgentCodexTurnStartBubblewrapErrorIsUserReadable(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true

	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/start":
			return json.RawMessage(`{"thread":{"id":"thread-1"}}`), nil
		case "turn/start":
			return nil, fmt.Errorf("\x1b[31mERROR\x1b[0m codex_app_server: Codex could not find bubblewrap on PATH. Codex will use the bundled bubblewrap in the meantime")
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	_, err := a.ChatStream(context.Background(), "user-1", "hello", nil)
	if err == nil {
		t.Fatal("ChatStream returned nil error")
	}
	got := err.Error()
	if strings.Contains(got, "\x1b") || strings.Contains(got, "[31m") {
		t.Fatalf("ChatStream error kept ANSI escapes: %q", got)
	}
	for _, want := range []string{"turn error:", "Codex sandbox dependency warning", "sudo apt install -y bubblewrap"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ChatStream error = %q, missing %q", got, want)
		}
	}
}
