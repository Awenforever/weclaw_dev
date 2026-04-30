package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
)

func newTestHandler() *Handler {
	return &Handler{agents: make(map[string]agent.Agent)}
}

func TestParseCommand_NoPrefix(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("hello world")
	if len(names) != 0 {
		t.Errorf("expected nil names, got %v", names)
	}
	if msg != "hello world" {
		t.Errorf("expected full text, got %q", msg)
	}
}

func TestParseCommand_SlashWithAgent(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("/claude explain this code")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude], got %v", names)
	}
	if msg != "explain this code" {
		t.Errorf("expected 'explain this code', got %q", msg)
	}
}

func TestParseCommand_AtPrefix(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("@claude explain this code")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude], got %v", names)
	}
	if msg != "explain this code" {
		t.Errorf("expected 'explain this code', got %q", msg)
	}
}

func TestParseCommand_MultiAgent(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("@cc @cx hello")
	if len(names) != 2 || names[0] != "claude" || names[1] != "codex" {
		t.Errorf("expected [claude codex], got %v", names)
	}
	if msg != "hello" {
		t.Errorf("expected 'hello', got %q", msg)
	}
}

func TestParseCommand_MultiAgentDedup(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("@cc @cc hello")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude] (deduped), got %v", names)
	}
	if msg != "hello" {
		t.Errorf("expected 'hello', got %q", msg)
	}
}

func TestParseCommand_SwitchOnly(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("/claude")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude], got %v", names)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestParseCommand_Alias(t *testing.T) {
	h := newTestHandler()
	names, msg := h.parseCommand("/cc write a function")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude] from /cc alias, got %v", names)
	}
	if msg != "write a function" {
		t.Errorf("expected 'write a function', got %q", msg)
	}
}

func TestParseCommand_CustomAlias(t *testing.T) {
	h := newTestHandler()
	h.customAliases = map[string]string{"ai": "claude", "c": "claude"}
	names, msg := h.parseCommand("/ai hello")
	if len(names) != 1 || names[0] != "claude" {
		t.Errorf("expected [claude] from custom alias, got %v", names)
	}
	if msg != "hello" {
		t.Errorf("expected 'hello', got %q", msg)
	}
}

func TestResolveAlias(t *testing.T) {
	h := newTestHandler()
	tests := map[string]string{
		"cc":  "claude",
		"cx":  "codex",
		"oc":  "openclaw",
		"cs":  "cursor",
		"km":  "kimi",
		"gm":  "gemini",
		"ocd": "opencode",
	}
	for alias, want := range tests {
		got := h.resolveAlias(alias)
		if got != want {
			t.Errorf("resolveAlias(%q) = %q, want %q", alias, got, want)
		}
	}
	if got := h.resolveAlias("unknown"); got != "unknown" {
		t.Errorf("resolveAlias(unknown) = %q, want %q", got, "unknown")
	}
	h.customAliases = map[string]string{"cc": "custom-claude"}
	if got := h.resolveAlias("cc"); got != "custom-claude" {
		t.Errorf("resolveAlias(cc) with custom = %q, want custom-claude", got)
	}
}

func TestBuildHelpText(t *testing.T) {
	text := buildHelpText()
	if text == "" {
		t.Error("help text is empty")
	}
	if !strings.Contains(text, "/info") {
		t.Error("help text should mention /info")
	}
	if !strings.Contains(text, "/help") {
		t.Error("help text should mention /help")
	}
}

func TestResetDefaultSessionUsesConfiguredAgentName(t *testing.T) {
	h := newTestHandler()
	h.SetDefaultAgent("codex", &fakeAgent{
		info:      agent.AgentInfo{Name: "/usr/local/bin/codex", Type: "acp"},
		sessionID: "session-123",
	})

	got := h.resetDefaultSession(context.Background(), "user-1")
	want := "Created a new codex session: session-123"
	if got != want {
		t.Fatalf("resetDefaultSession() = %q, want %q", got, want)
	}
}

func TestSwitchDefaultUsesEnglishReply(t *testing.T) {
	h := NewHandler(func(ctx context.Context, name string) agent.Agent {
		if name != "codex" {
			return nil
		}
		return &fakeAgent{info: agent.AgentInfo{Name: "/usr/local/bin/codex", Type: "acp"}}
	}, nil)

	got := h.switchDefault(context.Background(), "codex")
	want := "Switched default agent to codex."
	if got != want {
		t.Fatalf("switchDefault() = %q, want %q", got, want)
	}
}

func TestSendReplyWithMediaChunksFinalReply(t *testing.T) {
	client, sent := newCaptureClient(t)
	h := NewHandler(nil, nil)
	msg := ilink.WeixinMessage{FromUserID: "user-1", ContextToken: "ctx-token"}

	h.sendReplyWithMedia(context.Background(), client, msg, "codex", "First paragraph.\n\nSecond paragraph.", "client-1")

	if len(*sent) != 2 {
		t.Fatalf("sent %d messages, want 2: %#v", len(*sent), *sent)
	}
	wantTexts := []string{"First paragraph.", "Second paragraph."}
	for i, want := range wantTexts {
		got := (*sent)[i].Msg.ItemList[0].TextItem.Text
		if got != want {
			t.Fatalf("sent[%d] text = %q, want %q", i, got, want)
		}
	}
	if (*sent)[0].Msg.ClientID != "client-1" {
		t.Fatalf("first client ID = %q, want client-1", (*sent)[0].Msg.ClientID)
	}
	if (*sent)[1].Msg.ClientID == "" || (*sent)[1].Msg.ClientID == "client-1" {
		t.Fatalf("second client ID = %q, want a new non-empty ID", (*sent)[1].Msg.ClientID)
	}
}

func TestSendReplyWithMediaOptionsSingleTextModeSendsOneFinalReply(t *testing.T) {
	client, sent := newCaptureClient(t)
	h := NewHandler(nil, nil)
	msg := ilink.WeixinMessage{FromUserID: "user-1", ContextToken: "ctx-token"}

	delivered := h.sendReplyWithMediaOptions(
		context.Background(),
		client,
		msg,
		"codex",
		"First paragraph.\n\nSecond paragraph.",
		"client-1",
		true,
		replyTextSingle,
	)
	if !delivered {
		t.Fatal("sendReplyWithMediaOptions() = false, want true")
	}

	if len(*sent) != 1 {
		t.Fatalf("sent %d messages, want 1: %#v", len(*sent), *sent)
	}
	got := (*sent)[0].Msg.ItemList[0].TextItem.Text
	want := "First paragraph.\n\nSecond paragraph."
	if got != want {
		t.Fatalf("sent[0] text = %q, want %q", got, want)
	}
	if (*sent)[0].Msg.ClientID != "client-1" {
		t.Fatalf("client ID = %q, want client-1", (*sent)[0].Msg.ClientID)
	}
}

func TestProgressSenderIsNotParagraphChunked(t *testing.T) {
	h := NewHandler(nil, nil)
	ag := &fakeStreamingAgent{
		events: []agent.ProgressEvent{
			{Type: agent.ProgressEventToolStart, Text: "memory_router.memory_query"},
		},
		reply: "final answer",
	}
	var sent []string
	h.SetStreamConfig(StreamConfig{
		Enabled:       true,
		Interval:      time.Hour,
		MaxChunkChars: 100,
		ToolEvents:    true,
	})
	h.streamSender = func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
		sent = append(sent, text)
		return nil
	}

	_, _, err := h.chatMaybeStream(context.Background(), &ilink.Client{}, ilink.WeixinMessage{
		FromUserID:   "user-1",
		ContextToken: "ctx-token",
	}, ag, "user-1", "hello")
	if err != nil {
		t.Fatalf("chatMaybeStream returned error: %v", err)
	}
	assertSent(t, sent, []string{"using memory_router.memory_query"})
}

func TestTypingKeepaliveSendsImmediatelyAndRefreshes(t *testing.T) {
	h := NewHandler(nil, nil)
	h.typingEvery = 10 * time.Millisecond

	var mu sync.Mutex
	var sent int
	h.typingSender = func(ctx context.Context, client *ilink.Client, userID, contextToken string) error {
		mu.Lock()
		sent++
		mu.Unlock()
		return nil
	}

	stop := h.startTypingKeepalive(context.Background(), &ilink.Client{}, "user-1", "ctx-token")
	defer stop()

	time.Sleep(35 * time.Millisecond)

	mu.Lock()
	got := sent
	mu.Unlock()
	if got < 2 {
		t.Fatalf("typing sends = %d, want at least 2", got)
	}
}

func TestTypingKeepaliveStopsAfterCancel(t *testing.T) {
	h := NewHandler(nil, nil)
	h.typingEvery = 10 * time.Millisecond

	var mu sync.Mutex
	var sent int
	h.typingSender = func(ctx context.Context, client *ilink.Client, userID, contextToken string) error {
		mu.Lock()
		sent++
		mu.Unlock()
		return nil
	}

	stop := h.startTypingKeepalive(context.Background(), &ilink.Client{}, "user-1", "ctx-token")
	time.Sleep(20 * time.Millisecond)
	stop()

	mu.Lock()
	before := sent
	mu.Unlock()
	time.Sleep(25 * time.Millisecond)
	mu.Lock()
	after := sent
	mu.Unlock()

	if after != before {
		t.Fatalf("typing sends after stop = %d, want unchanged from %d", after, before)
	}
}

type fakeAgent struct {
	info      agent.AgentInfo
	reply     string
	sessionID string
}

func (a *fakeAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	return a.reply, nil
}

func (a *fakeAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	return a.sessionID, nil
}

func (a *fakeAgent) Info() agent.AgentInfo {
	return a.info
}

func (a *fakeAgent) SetCwd(cwd string) {}

func newCaptureClient(t *testing.T) (*ilink.Client, *[]ilink.SendMessageRequest) {
	t.Helper()
	var sent []ilink.SendMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/sendmessage" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var req ilink.SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		sent = append(sent, req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	t.Cleanup(server.Close)

	client := ilink.NewClient(&ilink.Credentials{
		BotToken:   "token",
		ILinkBotID: "bot-1",
		BaseURL:    server.URL,
	})
	return client, &sent
}
