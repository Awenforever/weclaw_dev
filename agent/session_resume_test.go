package agent

import (
	"context"
	"encoding/json"
	"testing"
)

func TestACPAgentResumeCodexThreadAvoidsThreadStart(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	if err := a.ResumeSession("user-1", "thread-existing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}
	if got := a.CurrentSessionID("user-1"); got != "thread-existing" {
		t.Fatalf("CurrentSessionID = %q, want thread-existing", got)
	}
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		t.Fatalf("unexpected rpc call %s", method)
		return nil, nil
	}
	got, isNew, err := a.getOrCreateThread(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("getOrCreateThread returned error: %v", err)
	}
	if isNew {
		t.Fatal("getOrCreateThread reported new thread after resume")
	}
	if got != "thread-existing" {
		t.Fatalf("getOrCreateThread = %q, want thread-existing", got)
	}
}

func TestACPAgentResumeLegacySessionAvoidsSessionNew(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "claude-agent-acp"})
	if err := a.ResumeSession("user-1", "session-existing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}
	if got := a.CurrentSessionID("user-1"); got != "session-existing" {
		t.Fatalf("CurrentSessionID = %q, want session-existing", got)
	}
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		t.Fatalf("unexpected rpc call %s", method)
		return nil, nil
	}
	got, isNew, err := a.getOrCreateSession(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("getOrCreateSession returned error: %v", err)
	}
	if isNew {
		t.Fatal("getOrCreateSession reported new session after resume")
	}
	if got != "session-existing" {
		t.Fatalf("getOrCreateSession = %q, want session-existing", got)
	}
}
