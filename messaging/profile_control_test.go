package messaging

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fastclaw-ai/weclaw/agent"
)

type runtimeControlTestAgent struct {
	info             agent.AgentInfo
	sessionID        string
	currentSessionID string
	ensureSessionID  string
	ensureCalls      int
	tokenUsage       agent.TokenUsageSnapshot
	tokenUsageOK     bool
	chatCalls        int
	resetCalls       int
	stopped          bool
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

func (a *runtimeControlTestAgent) CurrentSessionID(conversationID string) string {
	return a.currentSessionID
}

func (a *runtimeControlTestAgent) ResumeSession(conversationID, sessionID string) error {
	a.currentSessionID = sessionID
	return nil
}

func (a *runtimeControlTestAgent) EnsureSession(ctx context.Context, conversationID string) (string, error) {
	a.ensureCalls++
	if a.currentSessionID == "" {
		a.currentSessionID = a.ensureSessionID
	}
	return a.currentSessionID, nil
}

func (a *runtimeControlTestAgent) CurrentTokenUsage(conversationID string) (agent.TokenUsageSnapshot, bool) {
	return a.tokenUsage, a.tokenUsageOK
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
	if !strings.Contains(reply, "🧩 Agent") {
		t.Fatalf("status reply = %q, want agent card", reply)
	}
	if !strings.Contains(reply, "🔌 Proxy") {
		t.Fatalf("status reply = %q, want proxy card", reply)
	}
	if !strings.Contains(reply, "📁 Paths") {
		t.Fatalf("status reply = %q, want paths card", reply)
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
	if !strings.Contains(reply, "Existing session was preserved") {
		t.Fatalf("reply = %q, want preserved session message", reply)
	}
	if !strings.Contains(reply, "thinking: disabled") {
		t.Fatalf("reply = %q, want thinking disabled marker", reply)
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
	if created.resetCalls != 0 {
		t.Fatalf("resetCalls = %d, want 0 because /profile must preserve the current session", created.resetCalls)
	}
}

func TestRuntimeControlProfileSwitchThinkingEnabled(t *testing.T) {
	var created *runtimeControlTestAgent
	factoryCalls := 0

	h := NewHandler(func(ctx context.Context, name string) agent.Agent {
		if name != "deepseek-thinking" {
			return nil
		}
		factoryCalls++
		created = &runtimeControlTestAgent{
			info:      agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
			sessionID: "session-deepseek-thinking",
		}
		return created
	}, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek-thinking", Type: "acp", Command: "codex", Model: "deepseek-v4-pro"}})

	reply, ok := h.handleRuntimeControl(context.Background(), "/profile deepseek-thinking", "user-1")
	if !ok {
		t.Fatal("/profile should be intercepted")
	}
	if !strings.Contains(reply, "Switched default agent to deepseek-thinking") {
		t.Fatalf("reply = %q, want profile switch message", reply)
	}
	if !strings.Contains(reply, "Existing session was preserved") {
		t.Fatalf("reply = %q, want preserved session message", reply)
	}
	if !strings.Contains(reply, "thinking: enabled") {
		t.Fatalf("reply = %q, want thinking enabled marker", reply)
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
	if created.resetCalls != 0 {
		t.Fatalf("resetCalls = %d, want 0 because /profile must preserve the current session", created.resetCalls)
	}
}

func TestRuntimeControlRestartCurrentDefaultResetsSessionOnly(t *testing.T) {
	old := &runtimeControlTestAgent{
		info:      agent.AgentInfo{Name: "deepseek", Type: "acp"},
		sessionID: "new-session",
	}
	factoryCalls := 0

	h := NewHandler(func(ctx context.Context, name string) agent.Agent {
		factoryCalls++
		return nil
	}, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek", Type: "acp", Command: "codex"}})
	h.SetDefaultAgent("deepseek", old)

	reply, ok := h.handleRuntimeControl(context.Background(), "/restart", "user-1")
	if !ok {
		t.Fatal("/restart should be intercepted")
	}
	if old.stopped {
		t.Fatal("/restart should not stop the current default agent")
	}
	if factoryCalls != 0 {
		t.Fatalf("factoryCalls = %d, want 0 because /restart must not recreate the profile agent", factoryCalls)
	}
	if old.resetCalls != 1 {
		t.Fatalf("resetCalls = %d, want 1", old.resetCalls)
	}
	if old.chatCalls != 0 {
		t.Fatalf("chatCalls = %d, want 0 because restart must not enter Chat", old.chatCalls)
	}
	if strings.Contains(reply, "Profile switched") || strings.Contains(reply, "Switched default agent") {
		t.Fatalf("reply = %q, want no profile switch message", reply)
	}
	if !strings.Contains(reply, "config: preserved") {
		t.Fatalf("reply = %q, want config preserved marker", reply)
	}
}

func TestRuntimeControlNowReportsRunningTurn(t *testing.T) {
	h := NewHandler(nil, nil)
	_, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "check balance")
	defer cleanup()

	turn.observeProgress(agent.ProgressEvent{
		Type: agent.ProgressEventToolStart,
		Text: "using memory_router.memory_query",
	})

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "Current task") {
		t.Fatalf("reply = %q, want current task card", reply)
	}
	if !strings.Contains(reply, "deepseek-thinking") {
		t.Fatalf("reply = %q, want profile", reply)
	}
	if !strings.Contains(reply, "memory_router.memory_query") {
		t.Fatalf("reply = %q, want latest progress", reply)
	}
}

func TestRuntimeControlNowReportsAssistantProgressPreview(t *testing.T) {
	h := NewHandler(nil, nil)
	_, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "audit now command")
	defer cleanup()

	turn.observeProgress(agent.ProgressEvent{
		Type: agent.ProgressEventAssistantMessageComplete,
		Text: "I found the /now handler and am checking the progress state.",
	})

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "drafting: I found the /now handler") {
		t.Fatalf("reply = %q, want assistant progress preview", reply)
	}
	if strings.Contains(reply, "assistant message received") {
		t.Fatalf("reply = %q, want no generic assistant placeholder when text is available", reply)
	}
}

func TestRuntimeControlNowKeepsGenericAssistantProgressForEmptyText(t *testing.T) {
	h := NewHandler(nil, nil)
	_, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "audit now command")
	defer cleanup()

	turn.observeProgress(agent.ProgressEvent{
		Type: agent.ProgressEventAssistantMessageComplete,
		Text: "   \n\n",
	})

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "assistant message received") {
		t.Fatalf("reply = %q, want generic assistant placeholder for empty text", reply)
	}
}

func TestRuntimeControlNowReportsAssistantDeltaPreviewWithoutStreamingIt(t *testing.T) {
	h := NewHandler(nil, nil)
	_, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "draft answer")
	defer cleanup()

	turn.observeProgress(agent.ProgressEvent{
		Type: agent.ProgressEventAssistantDelta,
		Text: "Reading handler.go and checking progress updates.",
	})

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "drafting: Reading handler.go") {
		t.Fatalf("reply = %q, want assistant delta progress preview", reply)
	}
}

func TestRuntimeControlNowIdle(t *testing.T) {
	h := NewHandler(nil, nil)
	h.defaultName = "deepseek"

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "Idle") {
		t.Fatalf("reply = %q, want idle card", reply)
	}
	if !strings.Contains(reply, "deepseek") {
		t.Fatalf("reply = %q, want current profile", reply)
	}
}

func TestRuntimeControlCancelRunningTurn(t *testing.T) {
	h := NewHandler(nil, nil)
	turnCtx, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "long task")
	defer cleanup()

	reply, ok := h.handleRuntimeControl(context.Background(), "/cancel", "user-1")
	if !ok {
		t.Fatal("/cancel should be intercepted")
	}
	if !strings.Contains(reply, "Cancel requested") {
		t.Fatalf("reply = %q, want cancel requested card", reply)
	}
	if !turn.wasCancelRequested() {
		t.Fatal("turn was not marked as cancel requested")
	}
	if turnCtx.Err() == nil {
		t.Fatal("turn context was not cancelled")
	}
}

func TestRuntimeControlCancelIdle(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/cancel", "user-1")
	if !ok {
		t.Fatal("/cancel should be intercepted")
	}
	if !strings.Contains(reply, "No running task") {
		t.Fatalf("reply = %q, want no running task card", reply)
	}
}

func TestRuntimeControlUnknownSlashCommandIsBlocked(t *testing.T) {
	h := NewHandler(nil, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek", Type: "acp", Command: "codex"}})

	reply, ok := h.handleRuntimeControl(context.Background(), "/cancle", "user-1")
	if !ok {
		t.Fatal("unknown slash command should be intercepted")
	}
	if !strings.Contains(reply, "Unknown slash command") {
		t.Fatalf("reply = %q, want unknown slash command card", reply)
	}
	if !strings.Contains(reply, "Not sent to agent") {
		t.Fatalf("reply = %q, want local block marker", reply)
	}
	if !strings.Contains(reply, "/cancel") {
		t.Fatalf("reply = %q, want typo suggestion", reply)
	}
}

func TestRuntimeControlKnownSlashAgentCommandStillRoutes(t *testing.T) {
	h := NewHandler(nil, nil)
	h.SetAgentMetas([]AgentMeta{{Name: "deepseek", Type: "acp", Command: "codex"}})

	_, ok := h.handleRuntimeControl(context.Background(), "/deepseek hello", "user-1")
	if ok {
		t.Fatal("known slash agent command should not be intercepted by runtime control")
	}
}

func TestRuntimeControlEffortOnlySupportsDeepSeekProfiles(t *testing.T) {
	h := NewHandler(nil, nil)
	h.defaultName = "codex"

	reply, ok := h.handleRuntimeControl(context.Background(), "/effort max", "user-1")
	if !ok {
		t.Fatal("/effort should be intercepted")
	}
	if !strings.Contains(reply, "only supported for deepseek and deepseek-thinking") {
		t.Fatalf("reply = %q, want DeepSeek-only message", reply)
	}
}

func TestRuntimeControlNowReportsSessionID(t *testing.T) {
	h := NewHandler(nil, nil)
	_, turn, cleanup := h.beginRunningTurn(context.Background(), "user-1", "deepseek-thinking", "long task")
	defer cleanup()

	turn.observeProgress(agent.ProgressEvent{
		Type:      agent.ProgressEventStatus,
		Text:      "session ready",
		SessionID: "thread-now-123",
	})

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "session: thread-now-123") {
		t.Fatalf("reply = %q, want session ID", reply)
	}
}

func TestPendingResumeAppliesToFirstMatchingUserTurn(t *testing.T) {
	ag := &runtimeControlTestAgent{
		info: agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)
	h.SetPendingResume("deepseek-thinking", "thread-resume-123")

	h.applyPendingResume(context.Background(), "deepseek-thinking", ag, "user-1")

	if ag.currentSessionID != "thread-resume-123" {
		t.Fatalf("currentSessionID = %q, want thread-resume-123", ag.currentSessionID)
	}
}

func TestRuntimeControlNowIdleReportsOpenSessionID(t *testing.T) {
	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		currentSessionID: "thread-idle-123",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if !strings.Contains(reply, "running: no") {
		t.Fatalf("reply = %q, want idle status", reply)
	}
	if !strings.Contains(reply, "session: thread-idle-123") {
		t.Fatalf("reply = %q, want idle session ID", reply)
	}
}

func TestRuntimeControlNowIdleEnsuresSessionID(t *testing.T) {
	ag := &runtimeControlTestAgent{
		info:            agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		ensureSessionID: "thread-ensure-123",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if ag.ensureCalls != 1 {
		t.Fatalf("ensureCalls = %d, want 1", ag.ensureCalls)
	}
	if !strings.Contains(reply, "running: no") {
		t.Fatalf("reply = %q, want idle status", reply)
	}
	if !strings.Contains(reply, "session: thread-ensure-123") {
		t.Fatalf("reply = %q, want ensured session ID", reply)
	}
}

func TestRuntimeControlStatusReportsTokenUsage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(home+"/.codex", 0o755); err != nil {
		t.Fatalf("create codex dir: %v", err)
	}
	if err := os.WriteFile(home+"/.codex/config.toml", []byte("[profiles.deepseek-thinking]\nmodel_context_window = 1000000\n"), 0o644); err != nil {
		t.Fatalf("write codex config: %v", err)
	}
	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		currentSessionID: "thread-usage-1",
		tokenUsageOK:     true,
		tokenUsage: agent.TokenUsageSnapshot{
			ThreadID:           "thread-usage-1",
			TurnID:             "turn-usage-1",
			ModelContextWindow: 258400,
			Total: agent.TokenUsageBreakdown{
				TotalTokens:           43564,
				InputTokens:           43214,
				CachedInputTokens:     1200,
				OutputTokens:          350,
				ReasoningOutputTokens: 17,
			},
			Last: agent.TokenUsageBreakdown{
				TotalTokens:  22325,
				InputTokens:  22055,
				OutputTokens: 270,
			},
		},
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted")
	}
	for _, want := range []string{
		"📊 Context",
		"session: thread-usage-1",
		"context: 43.6k / 1M (4.4%)",
		"input: 43.2k",
		"cached input: 1.2k",
		"output: 350",
		"reasoning output: 17",
		"tools: --",
		"other: --",
		"last turn: 22.3k total, 22.1k input, 270 output",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
}

func TestRuntimeControlStatusShowsFallbackContextWindowWhileUsageIsWaiting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(home+"/.codex", 0o755); err != nil {
		t.Fatalf("create codex dir: %v", err)
	}
	if err := os.WriteFile(home+"/.codex/config.toml", []byte("[profiles.deepseek-thinking]\nmodel_context_window = 1000000\n"), 0o644); err != nil {
		t.Fatalf("write codex config: %v", err)
	}
	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-flash"},
		currentSessionID: "thread-waiting-1",
		tokenUsageOK:     false,
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted")
	}
	for _, want := range []string{
		"📊 Context",
		"session: thread-waiting-1",
		"context: -- / 1M (--%)",
		"input: --",
		"output: --",
		"tools: --",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
}

func TestDsproxyStatusArgsForThinkingProfile(t *testing.T) {
	got := dsproxyStatusArgsForProfile("deepseek-thinking")
	if len(got) != 2 || got[0] != "status" || got[1] != "thinking" {
		t.Fatalf("dsproxyStatusArgsForProfile(deepseek-thinking) = %#v, want status thinking", got)
	}
	got = dsproxyStatusArgsForProfile("deepseek")
	if len(got) != 1 || got[0] != "status" {
		t.Fatalf("dsproxyStatusArgsForProfile(deepseek) = %#v, want status", got)
	}
}

func TestFormatContextUsageLine(t *testing.T) {
	if got := formatContextUsageLine(0, 258400, false); got != "• context: -- / 258.4k (--%)" {
		t.Fatalf("formatContextUsageLine waiting = %q", got)
	}
	if got := formatContextUsageLine(43564, 258400, true); got != "• context: 43.6k / 258.4k (16.9%)" {
		t.Fatalf("formatContextUsageLine used = %q", got)
	}
}

func TestParseCodexProfileInt(t *testing.T) {
	configText := `[profiles.deepseek]
model_context_window = 1000000

[profiles.deepseek-thinking]
model_context_window = 750000
`
	if got := parseCodexProfileInt(configText, "deepseek", "model_context_window"); got != 1000000 {
		t.Fatalf("deepseek model_context_window = %d, want 1000000", got)
	}
	if got := parseCodexProfileInt(configText, "deepseek-thinking", "model_context_window"); got != 750000 {
		t.Fatalf("deepseek-thinking model_context_window = %d, want 750000", got)
	}
}
