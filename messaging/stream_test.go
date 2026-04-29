package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
)

func TestAssistantTextStreamBufferIntervalRequiresBoundary(t *testing.T) {
	var b assistantTextStreamBuffer
	b.text = "short status"

	if got := b.FlushReadyForInterval(); len(got) != 0 {
		t.Fatalf("FlushReadyForInterval() = %#v, want no chunks", got)
	}
	if b.text != "short status" {
		t.Fatalf("buffer text = %q, want unchanged", b.text)
	}
}

func TestAssistantTextStreamBufferIntervalFlushesSentencePrefix(t *testing.T) {
	var b assistantTextStreamBuffer
	b.text = "First sentence. Second partial"

	assertSent(t, b.FlushReadyForInterval(), []string{"First sentence."})
	if b.text != " Second partial" {
		t.Fatalf("buffer text = %q, want trailing partial retained", b.text)
	}
}

func TestAssistantTextStreamBufferIntervalPrefersParagraphBoundary(t *testing.T) {
	var b assistantTextStreamBuffer
	b.text = "First paragraph.\n\nSecond sentence. Third partial"

	assertSent(t, b.FlushReadyForInterval(), []string{"First paragraph."})
	if b.text != "Second sentence. Third partial" {
		t.Fatalf("buffer text = %q, want second paragraph retained", b.text)
	}
}

func TestAssistantTextStreamBufferSizeFlushPrefersSentenceBoundary(t *testing.T) {
	var b assistantTextStreamBuffer
	b.text = "First sentence. Second partial"

	assertSent(t, b.flushBySize(20), []string{"First sentence."})
	if b.text != "Second partial" {
		t.Fatalf("buffer text = %q, want trailing partial retained without leading space", b.text)
	}
}

func TestAssistantTextStreamBufferSizeFlushFallsBackToWhitespace(t *testing.T) {
	var b assistantTextStreamBuffer
	b.text = "one two three four five"

	assertSent(t, b.flushBySize(7), []string{"one"})
	if b.text != "two three four five" {
		t.Fatalf("buffer text = %q, want trailing words retained", b.text)
	}
}

func TestHandlerStreamingSendsCompletedAssistantMessages(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "First paragraph."},
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-2", Text: "Second paragraph."},
		},
		reply: "First paragraph.\n\nSecond paragraph.",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	reply, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID:   "user-1",
		ContextToken: "ctx-token",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if reply != "First paragraph.\n\nSecond paragraph." {
		t.Fatalf("reply = %q, want final response", reply)
	}
	if !streamedText {
		t.Fatalf("streamedText = false, want true")
	}
	assertSent(t, sent, []string{"First paragraph.", "Second paragraph.", "Done."})
}

func TestHandlerStreamingIgnoresAssistantDeltas(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantDelta, Text: "token fragment"},
		},
		reply: "final answer",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	reply, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID: "user-1",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if reply != "final answer" {
		t.Fatalf("reply = %q, want final answer returned to caller", reply)
	}
	if streamedText {
		t.Fatalf("streamedText = true, want false")
	}
	if len(sent) != 0 {
		t.Fatalf("sent = %#v, want no token-delta WeChat sends", sent)
	}
}

func TestHandlerStreamingSendsDoneAfterCompletedContent(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "final answer"},
		},
		reply: "final answer",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	_, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID: "user-1",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if !streamedText {
		t.Fatalf("streamedText = false, want true")
	}
	assertSent(t, sent, []string{"final answer", "Done."})
}

func TestHandlerStreamingAssistantFailureFallsBackToFinalReply(t *testing.T) {
	client, sent := newCaptureClient(t)
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "First paragraph."},
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-2", Text: "Second paragraph."},
		},
		reply: "First paragraph.\n\nSecond paragraph.",
	}
	h := newStreamTestHandler()

	var streamSent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		streamSent = append(streamSent, text)
		if text == "Second paragraph." {
			return errors.New("send failed")
		}
		return nil
	}
	msg := ilink.WeixinMessage{FromUserID: "user-1", ContextToken: "ctx-token"}

	reply, streamedText, err := h.chatWithAgent(context.Background(), client, msg, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatWithAgent returned error: %v", err)
	}
	if streamedText {
		t.Fatalf("streamedText = true, want false after assistant send failure")
	}
	h.sendReplyWithMediaOptions(context.Background(), client, msg, "codex", reply, "final-client", !streamedText)

	assertSent(t, streamSent, []string{"First paragraph.", "Second paragraph."})
	assertCapturedTexts(t, sent, []string{"First paragraph.", "Second paragraph.", "Done."})
}

func TestHandlerStreamingSuccessfulCompletedItemsSkipFinalTextReplay(t *testing.T) {
	client, sent := newCaptureClient(t)
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "final answer"},
		},
		reply: "final answer",
	}
	h := newStreamTestHandler()
	h.streamSender = SendTextReply
	msg := ilink.WeixinMessage{FromUserID: "user-1", ContextToken: "ctx-token"}

	reply, streamedText, err := h.chatWithAgent(context.Background(), client, msg, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatWithAgent returned error: %v", err)
	}
	if !streamedText {
		t.Fatalf("streamedText = false, want true")
	}
	h.sendReplyWithMediaOptions(context.Background(), client, msg, "codex", reply, "final-client", !streamedText)

	assertCapturedTexts(t, sent, []string{"final answer", "Done."})
}

func TestHandlerStreamingDeduplicatesCompletedAssistantMessages(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "final answer"},
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "final answer"},
		},
		reply: "final answer",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	_, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID: "user-1",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if !streamedText {
		t.Fatalf("streamedText = false, want true")
	}
	assertSent(t, sent, []string{"final answer", "Done."})
}

func TestHandlerStreamingSendsSpecificToolProgress(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventToolStart, Text: "tool started: memory_router.memory_query"},
		},
		reply: "done",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	reply, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID: "user-1",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if reply != "done" {
		t.Fatalf("reply = %q, want final response", reply)
	}
	if streamedText {
		t.Fatalf("streamedText = true, want false")
	}
	assertSent(t, sent, []string{"using memory_router.memory_query"})
}

func TestHandlerStreamingAssistantSendFailureDoesNotDisableToolProgress(t *testing.T) {
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventAssistantMessageComplete, ID: "msg-1", Text: "final answer"},
			{Type: agent.ProgressEventToolStart, Text: "using memory_router.memory_query"},
		},
		reply: "final answer",
	}
	h := newStreamTestHandler()

	var sent []string
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		if text == "final answer" {
			return errors.New("send failed")
		}
		return nil
	}

	_, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID: "user-1",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if streamedText {
		t.Fatalf("streamedText = true, want false")
	}
	assertSent(t, sent, []string{"final answer", "using memory_router.memory_query"})
}

func TestHandlerStreamingUsesConservativeDefaultPace(t *testing.T) {
	h := NewHandler(nil, nil)
	if h.streamPace != 1200*time.Millisecond {
		t.Fatalf("streamPace = %s, want 1200ms", h.streamPace)
	}
}

func TestHandlerStreamingDisabledFallsBackToChat(t *testing.T) {
	ag := &fakeStreamingAgent{reply: "fallback"}
	h := newStreamTestHandler()
	h.SetStreamConfig(StreamConfig{
		Enabled:       false,
		Interval:      time.Hour,
		MaxChunkChars: 5,
		ToolEvents:    true,
	})

	reply, streamedText, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	if reply != "fallback" {
		t.Fatalf("reply = %q, want fallback Chat response", reply)
	}
	if streamedText {
		t.Fatalf("streamedText = true, want false")
	}
	if ag.streamCalls != 0 {
		t.Fatalf("streamCalls = %d, want 0", ag.streamCalls)
	}
	if ag.chatCalls != 1 {
		t.Fatalf("chatCalls = %d, want 1", ag.chatCalls)
	}
}

func TestHandlerUserTurnLockSerializesSameUser(t *testing.T) {
	h := NewHandler(nil, nil)
	unlockFirst := h.lockUserTurn("user-1")

	acquired := make(chan struct{})
	releaseSecond := make(chan struct{})
	go func() {
		unlockSecond := h.lockUserTurn("user-1")
		close(acquired)
		<-releaseSecond
		unlockSecond()
	}()

	select {
	case <-acquired:
		t.Fatal("second same-user turn acquired lock before first turn finished")
	case <-time.After(20 * time.Millisecond):
	}

	unlockFirst()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second same-user turn did not acquire lock after first turn finished")
	}
	close(releaseSecond)
}

func newStreamTestHandler() *Handler {
	h := NewHandler(nil, nil)
	h.streamPace = 0
	h.SetStreamConfig(StreamConfig{
		Enabled:       true,
		Interval:      time.Hour,
		MaxChunkChars: 100,
		ToolEvents:    true,
	})
	return h
}

type fakeStreamingAgent struct {
	events      []agent.ProgressEvent
	delays      []time.Duration
	reply       string
	chatCalls   int
	streamCalls int
}

func (a *fakeStreamingAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	a.chatCalls++
	return a.reply, nil
}

func (a *fakeStreamingAgent) ChatStream(ctx context.Context, conversationID string, message string, onEvent func(agent.ProgressEvent) error) (string, error) {
	a.streamCalls++
	for i, evt := range a.events {
		if i < len(a.delays) && a.delays[i] > 0 {
			time.Sleep(a.delays[i])
		}
		if onEvent != nil {
			_ = onEvent(evt)
		}
	}
	return a.reply, nil
}

func (a *fakeStreamingAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	return "", nil
}

func (a *fakeStreamingAgent) Info() agent.AgentInfo {
	return agent.AgentInfo{Name: "fake", Type: "test"}
}

func (a *fakeStreamingAgent) SetCwd(cwd string) {}

func assertSent(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("sent = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sent[%d] = %q, want %q (all sent = %#v)", i, got[i], want[i], got)
		}
	}
}

func assertCapturedTexts(t *testing.T, sent *[]ilink.SendMessageRequest, want []string) {
	t.Helper()
	if len(*sent) != len(want) {
		t.Fatalf("sent %d messages, want %d: %#v", len(*sent), len(want), *sent)
	}
	for i, wantText := range want {
		got := (*sent)[i].Msg.ItemList[0].TextItem.Text
		if got != wantText {
			t.Fatalf("sent[%d] = %q, want %q", i, got, wantText)
		}
	}
}
