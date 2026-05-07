package agent

import (
	"context"
	"encoding/json"
	"testing"
)

func TestACPAgentCodexAppServerTurnStartIncludesModelProvider(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{
		Command:       "codex",
		Args:          []string{"app-server"},
		Model:         "deepseek-v4-pro",
		ModelProvider: "deepseek-thinking-proxy",
	})
	a.started = true

	var turnParams codexTurnStartParams
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/start":
			return json.RawMessage(`{"thread":{"id":"thread-1"}}`), nil
		case "turn/start":
			var ok bool
			turnParams, ok = params.(codexTurnStartParams)
			if !ok {
				t.Fatalf("turn/start params type = %T, want codexTurnStartParams", params)
			}
			go func() {
				a.handleCodexItemCompleted(json.RawMessage(`{"threadId":"thread-1","item":{"id":"msg-1","type":"agentMessage","text":"ok","phase":"final_answer"}}`))
				a.handleCodexTurnEvent("turn/completed", json.RawMessage(`{"threadId":"thread-1"}`))
			}()
			return json.RawMessage(`{"turn":{"id":"turn-1"}}`), nil
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	got, err := a.ChatStream(context.Background(), "conversation-1", "hello", nil)
	if err != nil {
		t.Fatalf("ChatStream error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("ChatStream result = %q, want ok", got)
	}
	if turnParams.Model != "deepseek-v4-pro" {
		t.Fatalf("turn/start model = %q, want deepseek-v4-pro", turnParams.Model)
	}
	if turnParams.ModelProvider != "deepseek-thinking-proxy" {
		t.Fatalf("turn/start modelProvider = %q, want deepseek-thinking-proxy", turnParams.ModelProvider)
	}
}
