package messaging

import (
	"context"
	"strings"
	"testing"

	"github.com/fastclaw-ai/weclaw/agent"
)

type runtimeControlTestAgent struct {
	info       agent.AgentInfo
	sessionID  string
	chatCalls  int
	resetCalls int
	stopped    bool
}

func (a *runtimeControlTestAgent) Info() agent.AgentInfo {
	return a.info
}

func (a *runtimeControlTestAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	a.chatCalls++
	return "runtime-control-test-reply", nil
}

func (a *runtimeControlTestAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	a.resetCalls++
	if a.sessionID != "" {
		return a.sessionID, nil
	}
	return "session-runtime-control", nil
}

func (a *runtimeControlTestAgent) SetCwd(cwd string) {}

func (a *runtimeControlTestAgent) Stop() {
	a.stopped = true
}

func TestRuntimeControlRejectsInvalidModelAndEffort(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/model bad-model", "user-1")
	if !ok {
		t.Fatal("/model should be intercepted")
	}
	if !strings.Contains(reply, "Unsupported model") {
		t.Fatalf("reply = %q, want unsupported model message", reply)
	}

	reply, ok = h.handleRuntimeControl(context.Background(), "/effort extreme", "user-1")
	if !ok {
		t.Fatal("/effort should be intercepted")
	}
	if !strings.Contains(reply, "Unsupported effort") {
		t.Fatalf("reply = %q, want unsupported effort message", reply)
	}
}

func TestRuntimeControlRejectsInvalidProfileAlias(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/profile thinking", "user-1")
	if !ok {
		t.Fatal("/profile should be intercepted")
	}
	if !strings.Contains(reply, "Unsupported profile") {
		t.Fatalf("reply = %q, want unsupported profile message", reply)
	}
}

func TestRuntimeControlStatusReturnsDiagnostics(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted as diagnostics")
	}
	if !strings.Contains(reply, "agent:") {
		t.Fatalf("status reply = %q, want agent status", reply)
	}
	if !strings.Contains(reply, "dsproxy status:") {
		t.Fatalf("status reply = %q, want dsproxy status section", reply)
	}
	if !strings.Contains(reply, "dsproxy config:") {
		t.Fatalf("status reply = %q, want dsproxy config section", reply)
	}
}

func TestRuntimeControlRestartWithoutDefault(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/restart", "user-1")
	if !ok {
		t.Fatal("/restart should be intercepted")
	}
	if !strings.Contains(reply, "No default agent configured") {
		t.Fatalf("restart reply = %q, want no default message", reply)
	}
}

func TestRuntimeControlProfileSwitchDoesNotChat(t *testing.T) {
	var created *runtimeControlTestAgent
	factoryCalls := 0

	h := NewHandler(func(ctx context.Context, name string) agent.Agent {
		if name != "deepseek" {
			return nil
		}
		factoryCalls++
		created = &runtimeControlTestAgent{
			info:      agent.AgentInfo{Name: "deepseek", Type: "acp", Model: "deepseek-v4-flash"},
			sessionID: "session-deepseek",
		}
		return created
	}, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek", Type: "acp", Command: "codex", Model: "deepseek-v4-flash"}})

	reply, ok := h.handleRuntimeControl(context.Background(), "/profile deepseek", "user-1")
	if !ok {
		t.Fatal("/profile should be intercepted")
	}
	if !strings.Contains(reply, "Switched default agent to deepseek") {
		t.Fatalf("reply = %q, want profile switch message", reply)
	}
	if !strings.Contains(reply, "Created a new deepseek session") {
		t.Fatalf("reply = %q, want session reset message", reply)
	}
	if factoryCalls != 1 {
		t.Fatalf("factoryCalls = %d, want 1", factoryCalls)
	}
	if created == nil {
		t.Fatal("created agent is nil")
	}
	if created.chatCalls != 0 {
		t.Fatalf("chatCalls = %d, want 0 because control messages must not enter Chat", created.chatCalls)
	}
	if created.resetCalls != 1 {
		t.Fatalf("resetCalls = %d, want 1", created.resetCalls)
	}
}

func TestRuntimeControlRestartCurrentDefaultStopsOldAgent(t *testing.T) {
	old := &runtimeControlTestAgent{
		info:      agent.AgentInfo{Name: "deepseek", Type: "acp"},
		sessionID: "old-session",
	}
	var created *runtimeControlTestAgent

	h := NewHandler(func(ctx context.Context, name string) agent.Agent {
		if name != "deepseek" {
			return nil
		}
		created = &runtimeControlTestAgent{
			info:      agent.AgentInfo{Name: "deepseek", Type: "acp"},
			sessionID: "new-session",
		}
		return created
	}, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek", Type: "acp", Command: "codex"}})
	h.SetDefaultAgent("deepseek", old)

	reply, ok := h.handleRuntimeControl(context.Background(), "/restart", "user-1")
	if !ok {
		t.Fatal("/restart should be intercepted")
	}
	if !old.stopped {
		t.Fatal("old default agent was not stopped")
	}
	if created == nil {
		t.Fatal("new default agent was not created")
	}
	if created.chatCalls != 0 {
		t.Fatalf("chatCalls = %d, want 0 because restart must not enter Chat", created.chatCalls)
	}
	if !strings.Contains(reply, "Switched default agent to deepseek") {
		t.Fatalf("reply = %q, want switch message", reply)
	}
}
