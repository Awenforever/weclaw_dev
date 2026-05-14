package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestACPAgentResumeCodexThreadCallsThreadResumeAndAvoidsThreadStart(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	if err := a.ResumeSession("user-1", "thread-existing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}
	if got := a.CurrentSessionID("user-1"); got != "thread-existing" {
		t.Fatalf("CurrentSessionID = %q, want thread-existing", got)
	}

	resumeCalls := 0
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/resume":
			resumeCalls++
			resumeParams, ok := params.(map[string]interface{})
			if !ok {
				t.Fatalf("thread/resume params type = %T, want map[string]interface{}", params)
			}
			if resumeParams["threadId"] != "thread-existing" {
				t.Fatalf("thread/resume threadId = %#v, want thread-existing", resumeParams["threadId"])
			}
			if _, ok := resumeParams["excludeTurns"]; ok {
				t.Fatalf("thread/resume should not send experimental excludeTurns without experimentalApi capability")
			}
			return json.RawMessage(`{"thread":{"id":"thread-existing"}}`), nil
		case "thread/start":
			t.Fatalf("thread/start should not be called for resumed thread")
			return nil, nil
		default:
			t.Fatalf("unexpected rpc call %s", method)
			return nil, nil
		}
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
	if resumeCalls != 1 {
		t.Fatalf("thread/resume calls = %d, want 1", resumeCalls)
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

func TestACPAgentResumeCodexThreadFailsWhenThreadMissingWithoutSilentReplacement(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true

	if err := a.ResumeSession("user-1", "thread-missing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}

	threadStartCount := 0
	turnStartCount := 0
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/resume":
			return nil, fmt.Errorf("agent error: thread not found: thread-missing")
		case "thread/start":
			threadStartCount++
			return json.RawMessage(`{"thread":{"id":"thread-recovered"}}`), nil
		case "turn/start":
			turnStartCount++
			return nil, nil
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	_, err := a.ChatStream(context.Background(), "user-1", "hello", nil)
	if err == nil {
		t.Fatal("ChatStream returned nil error for missing resumed thread")
	}
	if !isCodexThreadNotFoundError(err) {
		t.Fatalf("ChatStream error = %v, want thread not found", err)
	}
	if threadStartCount != 0 {
		t.Fatalf("thread/start count = %d, want 0 because replacement is not silent", threadStartCount)
	}
	if turnStartCount != 0 {
		t.Fatalf("turn/start count = %d, want 0 because thread/resume failed first", turnStartCount)
	}
	if gotSession := a.CurrentSessionID("user-1"); gotSession != "thread-missing" {
		t.Fatalf("CurrentSessionID = %q, want original resumed thread", gotSession)
	}
}

func TestACPAgentResumeCodexThreadUsesResumedThreadForTurn(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true

	if err := a.ResumeSession("user-1", "thread-existing"); err != nil {
		t.Fatalf("ResumeSession returned error: %v", err)
	}

	var methods []string
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		methods = append(methods, method)
		switch method {
		case "thread/resume":
			return json.RawMessage(`{"thread":{"id":"thread-existing"}}`), nil
		case "turn/start":
			turnParams, ok := params.(codexTurnStartParams)
			if !ok {
				t.Fatalf("turn/start params type = %T, want codexTurnStartParams", params)
			}
			if turnParams.ThreadID != "thread-existing" {
				t.Fatalf("turn/start thread = %q, want thread-existing", turnParams.ThreadID)
			}
			go func() {
				a.handleCodexItemCompleted(json.RawMessage(`{"threadId":"thread-existing","item":{"id":"msg-1","type":"agentMessage","text":"ok","phase":"final_answer"}}`))
				a.handleCodexTurnEvent("turn/completed", json.RawMessage(`{"threadId":"thread-existing"}`))
			}()
			return json.RawMessage(`{"turn":{"id":"turn-1"}}`), nil
		case "thread/start":
			t.Fatalf("thread/start should not be called for resumed thread")
			return nil, nil
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
	if len(methods) != 2 || methods[0] != "thread/resume" || methods[1] != "turn/start" {
		t.Fatalf("methods = %#v, want [thread/resume turn/start]", methods)
	}
	if gotSession := a.CurrentSessionID("user-1"); gotSession != "thread-existing" {
		t.Fatalf("CurrentSessionID = %q, want thread-existing", gotSession)
	}
}
