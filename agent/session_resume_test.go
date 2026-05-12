package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestACPAgentResumeCodexThreadFallsBackWhenThreadMissing(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true

	if err := a.ResumeSession("user-1", "thread-missing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}

	var turnThreads []string
	threadStartCount := 0
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/start":
			threadStartCount++
			return json.RawMessage(`{"thread":{"id":"thread-recovered"}}`), nil
		case "turn/start":
			turnParams, ok := params.(codexTurnStartParams)
			if !ok {
				t.Fatalf("turn/start params type = %T, want codexTurnStartParams", params)
			}
			turnThreads = append(turnThreads, turnParams.ThreadID)
			if turnParams.ThreadID == "thread-missing" {
				return nil, fmt.Errorf("agent error: thread not found: %s", turnParams.ThreadID)
			}
			if turnParams.ThreadID != "thread-recovered" {
				t.Fatalf("turn/start thread = %q, want thread-recovered", turnParams.ThreadID)
			}
			go func() {
				a.handleCodexItemCompleted(json.RawMessage(`{"threadId":"thread-recovered","item":{"id":"msg-1","type":"agentMessage","text":"ok","phase":"final_answer"}}`))
				a.handleCodexTurnEvent("turn/completed", json.RawMessage(`{"threadId":"thread-recovered"}`))
			}()
			return json.RawMessage(`{"turn":{"id":"turn-1"}}`), nil
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	got, err := a.ChatStream(context.Background(), "user-1", "hello", nil)
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatStream reply = %q, want ok", got)
	}
	if threadStartCount != 1 {
		t.Fatalf("thread/start count = %d, want 1", threadStartCount)
	}
	if len(turnThreads) != 2 || turnThreads[0] != "thread-missing" || turnThreads[1] != "thread-recovered" {
		t.Fatalf("turn/start threads = %#v, want [thread-missing thread-recovered]", turnThreads)
	}
	if gotSession := a.CurrentSessionID("user-1"); gotSession != "thread-recovered" {
		t.Fatalf("CurrentSessionID = %q, want thread-recovered", gotSession)
	}
}
