package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestACPAgentChatStreamCodexRawInputAndDeltasAreFallbackOnly(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
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
				time.Sleep(10 * time.Millisecond)
				a.handleCodexItemDelta(json.RawMessage(`{"threadId":"thread-1","delta":"Hel"}`))
				a.handleCodexItemDelta(json.RawMessage(`{"threadId":"thread-1","delta":"lo"}`))
				a.handleCodexTurnEvent("turn/completed", json.RawMessage(`{"threadId":"thread-1"}`))
			}()
			return json.RawMessage(`{}`), nil
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	var events []ProgressEvent
	reply, err := a.ChatStream(context.Background(), "user-1", "hi", func(evt ProgressEvent) error {
		events = append(events, evt)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if reply != "Hello" {
		t.Fatalf("reply = %q, want %q", reply, "Hello")
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v, want no user-visible delta events", events)
	}
	if len(turnParams.Input) != 1 {
		t.Fatalf("turn/start input count = %d, want 1", len(turnParams.Input))
	}
	input := turnParams.Input[0].Text
	if input != "hi" {
		t.Fatalf("turn/start input = %q, want raw user message", input)
	}
	if strings.Contains(input, "[EOF]") || strings.Contains(input, "WeClaw bridge instruction") {
		t.Fatalf("turn/start input contains bridge marker/instruction: %q", input)
	}
}

func TestACPAgentChatStreamCodexCompletedAgentMessageEmitsCompleteText(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	a.started = true
	a.rpcCall = func(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
		switch method {
		case "thread/start":
			return json.RawMessage(`{"thread":{"id":"thread-1"}}`), nil
		case "turn/start":
			go func() {
				time.Sleep(10 * time.Millisecond)
				a.handleCodexItemDelta(json.RawMessage(`{"threadId":"thread-1","itemId":"msg-1","delta":"ignored fallback"}`))
				a.handleCodexItemCompleted(json.RawMessage(`{"threadId":"thread-1","item":{"id":"msg-1","type":"agentMessage","text":"Complete answer.","phase":"final_answer"}}`))
				a.handleCodexTurnEvent("turn/completed", json.RawMessage(`{"threadId":"thread-1"}`))
			}()
			return json.RawMessage(`{}`), nil
		default:
			t.Fatalf("unexpected rpc method %s", method)
			return nil, nil
		}
	}

	var events []ProgressEvent
	reply, err := a.ChatStream(context.Background(), "user-1", "hi", func(evt ProgressEvent) error {
		events = append(events, evt)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if reply != "Complete answer." {
		t.Fatalf("reply = %q, want completed item text", reply)
	}
	if len(events) != 1 {
		t.Fatalf("events = %#v, want one complete assistant message", events)
	}
	if events[0].Type != ProgressEventAssistantMessageComplete || events[0].Text != "Complete answer." || !events[0].Final {
		t.Fatalf("event = %#v, want final assistant message complete", events[0])
	}
}

func TestACPAgentCodexCompletedAgentMessageExtractsContentTextAndParamPhase(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	ch := make(chan *codexTurnEvent, 10)
	a.turnCh["thread-1"] = ch

	a.handleCodexItemCompleted(json.RawMessage(`{"threadId":"thread-1","phase":"final_answer","item":{"id":"msg-2","type":"agentMessage","content":[{"type":"text","text":"First."},{"type":"text","text":"Second."}]}}`))

	got := drainTurnEvents(ch)
	if len(got) != 1 {
		t.Fatalf("got %d events %#v, want one completed item event", len(got), got)
	}
	if got[0].Text != "First.\n\nSecond." || got[0].ItemID != "msg-2" || !got[0].Final {
		t.Fatalf("event = %#v, want joined content text with final phase", got[0])
	}
}

func TestACPAgentReadLoopFiltersInternalEventsAndForwardsCompactToolMetadata(t *testing.T) {
	a := NewACPAgent(ACPAgentConfig{Command: "codex", Args: []string{"app-server"}})
	ch := make(chan *codexTurnEvent, 10)
	a.turnCh["thread-1"] = ch

	lines := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"turn/started","params":{"threadId":"thread-1"}}`,
		`{"jsonrpc":"2.0","method":"thread/status/changed","params":{"threadId":"thread-1","status":"running"}}`,
		`{"jsonrpc":"2.0","method":"mcpServer/startupStatus/updated","params":{"threadId":"thread-1","server":"memory_router","status":"starting"}}`,
		`{"jsonrpc":"2.0","method":"item/reasoning/textDelta","params":{"threadId":"thread-1","delta":"hidden"}}`,
		`{"jsonrpc":"2.0","method":"item/started","params":{"threadId":"thread-1","item":{"type":"userMessage"}}}`,
		`{"jsonrpc":"2.0","method":"item/started","params":{"threadId":"thread-1","item":{"type":"mcpToolCall","name":"memory_router.memory_query"}}}`,
		`{"jsonrpc":"2.0","method":"item/mcpToolCall/progress","params":{"threadId":"thread-1","name":"memory_router.memory_query","message":"running"}}`,
		`{"jsonrpc":"2.0","method":"item/completed","params":{"threadId":"thread-1","item":{"type":"mcpToolCall","name":"memory_router.memory_query","durationMs":2100}}}`,
		`{"jsonrpc":"2.0","method":"item/agentMessage/delta","params":{"threadId":"thread-1","delta":"visible"}}`,
	}, "\n")
	a.scanner = bufio.NewScanner(strings.NewReader(lines))
	a.readLoop()

	got := drainTurnEvents(ch)
	if len(got) != 3 {
		t.Fatalf("got %d events %#v, want 3 visible events", len(got), got)
	}
	if got[0].Progress == nil || got[0].Progress.Text != "using memory_router.memory_query" {
		t.Fatalf("first event = %#v, want compact tool metadata", got[0])
	}
	if got[1].Progress == nil || got[1].Progress.Text != "using memory_router.memory_query" {
		t.Fatalf("second event = %#v, want compact tool progress metadata", got[1])
	}
	if got[2].Delta != "visible" {
		t.Fatalf("third event = %#v, want visible assistant delta only", got[2])
	}
}

func drainTurnEvents(ch <-chan *codexTurnEvent) []*codexTurnEvent {
	var events []*codexTurnEvent
	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
		default:
			return events
		}
	}
}
