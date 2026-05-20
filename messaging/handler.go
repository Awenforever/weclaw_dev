package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/config"
	"github.com/fastclaw-ai/weclaw/ilink"
	"github.com/fastclaw-ai/weclaw/runtime_state"
	"github.com/google/uuid"
)

// AgentFactory creates an agent by config name. Returns nil if the name is unknown.
type AgentFactory func(ctx context.Context, name string) agent.Agent

// SaveDefaultFunc persists the default agent name to config file.
type SaveDefaultFunc func(name string) error

// AgentMeta holds static config info about an agent (for /status display).
type AgentMeta struct {
	Name    string
	Type    string // "acp", "cli", "http"
	Command string // binary path or endpoint
	Model   string
}

// StreamConfig controls optional progress forwarding while an agent is working.
type StreamConfig struct {
	Enabled       bool
	Interval      time.Duration
	MaxChunkChars int
	ToolEvents    bool
}

type streamSenderFunc func(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error
type typingSenderFunc func(ctx context.Context, client *ilink.Client, userID, contextToken string) error

type replyTextMode int

const (
	replyTextChunked replyTextMode = iota
	replyTextSingle
)

// Handler processes incoming WeChat messages and dispatches replies.
type Handler struct {
	mu            sync.RWMutex
	defaultName   string
	agents        map[string]agent.Agent // name -> running agent
	agentMetas    []AgentMeta            // all configured agents (for /status)
	agentWorkDirs map[string]string      // agent name -> configured/runtime cwd
	customAliases map[string]string      // custom alias -> agent name (from config)
	factory       AgentFactory
	saveDefault   SaveDefaultFunc
	contextTokens sync.Map // map[userID]contextToken
	saveDir       string   // directory to save images/files to
	seenMsgs      sync.Map // map[int64]time.Time — dedup by message_id
	streamConfig  StreamConfig
	streamSender  streamSenderFunc
	streamPace    time.Duration
	typingSender  typingSenderFunc
	typingEvery   time.Duration
	userTurns     sync.Map // map[userID]*sync.Mutex — serializes agent turns per user
	runningTurns  sync.Map // map[userID]*runningTurnState — current cancellable agent turn

	pendingResumeProfile string
	pendingResumeID      string
	pendingResumeApplied sync.Map // map[userID|profile|sessionID]bool
}

// NewHandler creates a new message handler.
func NewHandler(factory AgentFactory, saveDefault SaveDefaultFunc) *Handler {
	return &Handler{
		agents:        make(map[string]agent.Agent),
		agentWorkDirs: make(map[string]string),
		factory:       factory,
		saveDefault:   saveDefault,
		streamConfig: StreamConfig{
			Interval:      1500 * time.Millisecond,
			MaxChunkChars: 1200,
			ToolEvents:    true,
		},
		streamSender: SendTextReply,
		streamPace:   streamSendPaceDelay,
		typingSender: SendTypingState,
		typingEvery:  6 * time.Second,
	}
}

type runningTurnSnapshot struct {
	userID          string
	agentName       string
	status          string
	lastProgress    string
	startedAt       time.Time
	lastUpdateAt    time.Time
	cancelRequested bool
	sessionID       string
}

type runningTurnState struct {
	mu              sync.RWMutex
	userID          string
	agentName       string
	message         string
	startedAt       time.Time
	lastUpdateAt    time.Time
	status          string
	lastProgress    string
	cancel          context.CancelFunc
	cancelRequested bool
	sessionID       string
}

type defaultSessionStatus struct {
	profile   string
	ag        agent.Agent
	sessionID string
	source    string
}

func (s *runningTurnState) update(status, progress string) {
	if s == nil {
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(status) != "" {
		s.status = strings.TrimSpace(status)
	}
	if strings.TrimSpace(progress) != "" {
		s.lastProgress = truncate(normalizeLineEndings(strings.TrimSpace(progress)), 180)
	}
	s.lastUpdateAt = now
}

func (s *runningTurnState) updateSessionID(sessionID string) {
	if s == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	s.sessionID = sessionID
	s.lastUpdateAt = time.Now()
	s.mu.Unlock()
}

func (s *runningTurnState) observeProgress(evt agent.ProgressEvent) {
	if s == nil {
		return
	}
	if evt.SessionID != "" {
		s.updateSessionID(evt.SessionID)
	}
	if text, ok := compactStreamEvent(evt); ok {
		s.update("working", text)
		return
	}
	switch evt.Type {
	case agent.ProgressEventStatus:
		s.update("status", evt.Text)
	case agent.ProgressEventToolEnd:
		s.update("tool done", evt.Text)
	case agent.ProgressEventError:
		s.update("error", evt.Text)
	case agent.ProgressEventAssistantDelta:
		if text := compactAssistantProgress(evt.Text); text != "" {
			s.update("responding", "drafting: "+text)
		}
	case agent.ProgressEventAssistantMessageComplete:
		if evt.Final {
			s.update("finalizing", "final answer ready")
		} else if text := compactAssistantProgress(evt.Text); text != "" {
			s.update("responding", "drafting: "+text)
		} else {
			s.update("responding", "assistant message received")
		}
	}
}

func (s *runningTurnState) requestCancel() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cancelRequested = true
	s.status = "cancelling"
	s.lastProgress = "cancel requested"
	s.lastUpdateAt = time.Now()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *runningTurnState) wasCancelRequested() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancelRequested
}

func (s *runningTurnState) snapshot() runningTurnSnapshot {
	if s == nil {
		return runningTurnSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return runningTurnSnapshot{
		userID:          s.userID,
		agentName:       s.agentName,
		status:          s.status,
		lastProgress:    s.lastProgress,
		startedAt:       s.startedAt,
		lastUpdateAt:    s.lastUpdateAt,
		cancelRequested: s.cancelRequested,
		sessionID:       s.sessionID,
	}
}

func (h *Handler) beginRunningTurn(ctx context.Context, userID, agentName, message string) (context.Context, *runningTurnState, func()) {
	if userID == "" {
		userID = "default"
	}
	turnCtx, cancel := context.WithCancel(ctx)
	now := time.Now()
	state := &runningTurnState{
		userID:       userID,
		agentName:    agentName,
		message:      message,
		startedAt:    now,
		lastUpdateAt: now,
		status:       "running",
		lastProgress: "dispatching to agent",
		cancel:       cancel,
	}
	h.runningTurns.Store(userID, state)
	cleanup := func() {
		if value, ok := h.runningTurns.Load(userID); ok && value == state {
			h.runningTurns.Delete(userID)
		}
	}
	return turnCtx, state, cleanup
}

func (h *Handler) runningTurnSnapshot(userID string) (runningTurnSnapshot, bool) {
	if userID == "" {
		userID = "default"
	}
	value, ok := h.runningTurns.Load(userID)
	if !ok {
		return runningTurnSnapshot{}, false
	}
	state, ok := value.(*runningTurnState)
	if !ok {
		h.runningTurns.Delete(userID)
		return runningTurnSnapshot{}, false
	}
	return state.snapshot(), true
}

func (h *Handler) cancelRunningTurn(userID string) string {
	if userID == "" {
		userID = "default"
	}
	value, ok := h.runningTurns.Load(userID)
	if !ok {
		return commandCard(
			"ℹ️ No running task",
			"> There is no active turn to cancel.",
			"",
			"```text",
			"Action   none",
			"```",
		)
	}
	state, ok := value.(*runningTurnState)
	if !ok {
		h.runningTurns.Delete(userID)
		return commandCard(
			"ℹ️ No running task",
			"> There is no active turn to cancel.",
			"",
			"```text",
			"Action   cleared stale task state",
			"```",
		)
	}
	snap := state.snapshot()
	state.requestCancel()
	return commandCard(
		"🛑 Cancel requested",
		slashBoldField("Profile", slashInlineCode(snap.agentName)),
		slashBoldField("Session", "preserved"),
		"",
		"```text",
		"State    cancelling",
		"Running  "+formatTurnDuration(time.Since(snap.startedAt)),
		"```",
	)
}

func (h *Handler) buildNowStatus(ctx context.Context, userID string) string {
	if v, ok := h.runningTurns.Load(userID); ok {
		turn, _ := v.(*runningTurnState)
		snap := turn.snapshot()
		status := snap.status
		if snap.cancelRequested {
			status = "cancelling"
		}
		progress := snap.lastProgress
		if progress == "" {
			progress = "working"
		}
		sessionID := valueOrUnknown(snap.sessionID)
		h.recordRuntimeSession(snap.agentName, userID, snap.sessionID)
		updated := "unknown"
		if !snap.lastUpdateAt.IsZero() {
			updated = formatTurnDuration(time.Since(snap.lastUpdateAt)) + " ago"
		}
		return commandCard(
			"⏳ Current task",
			slashBoldField("Profile", slashInlineCode(snap.agentName)),
			slashBoldField("Session", slashInlineCode(sessionID)),
			"",
			"```text",
			"State    "+status,
			"Running  "+formatTurnDuration(time.Since(snap.startedAt)),
			"Updated  "+updated,
			"Now      "+compactCommandOutput(progress, 120),
			"```",
		)
	}

	resolved := h.resolveDefaultSessionForRuntimeControl(ctx, userID)
	return commandCard(
		"✅ Idle",
		slashBoldField("Profile", slashInlineCode(valueOrUnknown(resolved.profile))),
		slashBoldField("Session", slashInlineCode(valueOrUnknown(resolved.sessionID))),
		"",
		"```text",
		"State    idle",
		"Running  no",
		"Source   "+valueOrUnknown(resolved.source),
		"```",
	)
}

func formatTurnDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return "<1s"
	}
	seconds := int(d.Round(time.Second) / time.Second)
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%02dm", hours, minutes)
}

func compactAssistantProgress(text string) string {
	text = strings.TrimSpace(normalizeLineEndings(text))
	if text == "" {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "weclaw bridge instruction") {
			continue
		}
		return truncate(line, 140)
	}
	return ""
}

// SetStreamConfig enables or disables progress forwarding.
func (h *Handler) SetStreamConfig(cfg StreamConfig) {
	if cfg.Interval <= 0 {
		cfg.Interval = 1500 * time.Millisecond
	}
	if cfg.MaxChunkChars <= 0 {
		cfg.MaxChunkChars = 1200
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streamConfig = cfg
}

// SetSaveDir sets the directory for saving images and files.
func (h *Handler) SetSaveDir(dir string) {
	h.saveDir = dir
}

// cleanSeenMsgs removes entries older than 5 minutes from the dedup cache.
func (h *Handler) cleanSeenMsgs() {
	cutoff := time.Now().Add(-5 * time.Minute)
	h.seenMsgs.Range(func(key, value any) bool {
		if t, ok := value.(time.Time); ok && t.Before(cutoff) {
			h.seenMsgs.Delete(key)
		}
		return true
	})
}

// SetCustomAliases sets custom alias mappings from config.
func (h *Handler) SetCustomAliases(aliases map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.customAliases = aliases
}

// SetAgentMetas sets the list of all configured agents (for /status).
func (h *Handler) SetAgentMetas(metas []AgentMeta) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentMetas = metas
}

// SetPendingResume configures a startup resume ID to apply to the first matching user turn.
func (h *Handler) SetPendingResume(profile, sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingResumeProfile = strings.TrimSpace(profile)
	h.pendingResumeID = strings.TrimSpace(sessionID)
}

// SetAgentWorkDirs sets the configured working directory for each agent.
func (h *Handler) SetAgentWorkDirs(workDirs map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.agentWorkDirs = make(map[string]string, len(workDirs))
	for name, dir := range workDirs {
		h.agentWorkDirs[name] = dir
	}
}

// SetDefaultAgent sets the default agent (already started).
func (h *Handler) SetDefaultAgent(name string, ag agent.Agent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.defaultName = name
	h.agents[name] = ag
	log.Printf("[handler] default agent ready: %s (%s)", name, ag.Info())
}

// getAgent returns a running agent by name, or starts it on demand via factory.
func (h *Handler) getAgent(ctx context.Context, name string) (agent.Agent, error) {
	// Fast path: already running
	h.mu.RLock()
	ag, ok := h.agents[name]
	h.mu.RUnlock()
	if ok {
		return ag, nil
	}

	// Slow path: create on demand
	if h.factory == nil {
		return nil, fmt.Errorf("agent %q not found and no factory configured", name)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if ag, ok := h.agents[name]; ok {
		return ag, nil
	}

	log.Printf("[handler] starting agent %q on demand...", name)
	ag = h.factory(ctx, name)
	if ag == nil {
		return nil, fmt.Errorf("agent %q not available", name)
	}

	h.agents[name] = ag
	log.Printf("[handler] agent started on demand: %s (%s)", name, ag.Info())
	return ag, nil
}

// getDefaultAgent returns the default agent (may be nil if not ready yet).
func (h *Handler) getDefaultAgent() agent.Agent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.defaultName == "" {
		return nil
	}
	return h.agents[h.defaultName]
}

func (h *Handler) getDefaultAgentWithName() (string, agent.Agent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.defaultName == "" {
		return "", nil
	}
	return h.defaultName, h.agents[h.defaultName]
}

func (h *Handler) clearPendingResumeForProfile(profile string) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pendingResumeProfile != profile {
		return
	}
	h.pendingResumeProfile = ""
	h.pendingResumeID = ""
	h.pendingResumeApplied = sync.Map{}
	log.Printf("[handler] cleared pending resume for profile %s after explicit new session", profile)
}

func (h *Handler) applyPendingResume(ctx context.Context, name string, ag agent.Agent, userID string) {
	if ag == nil || userID == "" {
		return
	}
	h.mu.RLock()
	profile := h.pendingResumeProfile
	sessionID := h.pendingResumeID
	h.mu.RUnlock()
	if sessionID == "" || profile == "" || profile != name {
		return
	}

	key := userID + "|" + name + "|" + sessionID
	if _, loaded := h.pendingResumeApplied.LoadOrStore(key, true); loaded {
		return
	}

	resumer, ok := ag.(agent.SessionResumer)
	if !ok {
		log.Printf("[handler] pending resume ignored because agent %q does not support resume", name)
		return
	}
	if err := resumer.ResumeSession(userID, sessionID); err != nil {
		log.Printf("[handler] pending resume failed for agent %q: %v", name, err)
		return
	}
	log.Printf("[handler] pending resume applied (profile=%s, session=%s, user=%s)", name, sessionID, userID)
	h.recordRuntimeSession(name, userID, sessionID)
	_ = ctx
}

func (h *Handler) recordRuntimeSession(profile, userID, sessionID string) {
	if strings.HasSuffix(os.Args[0], ".test") {
		return
	}
	if err := runtime_state.UpsertSession(profile, userID, sessionID); err != nil {
		log.Printf("[handler] failed to persist runtime session hint: %v", err)
	}
}

func currentAgentSessionID(ag agent.Agent, userID string) string {
	inspector, ok := ag.(agent.SessionInspector)
	if !ok {
		return ""
	}
	return inspector.CurrentSessionID(userID)
}

func (h *Handler) resolveDefaultSessionForRuntimeControl(ctx context.Context, userID string) defaultSessionStatus {
	h.mu.RLock()
	name := h.defaultName
	ag := h.agents[name]
	h.mu.RUnlock()

	if name != "" && ag == nil && h.factory != nil {
		if started, err := h.getAgent(ctx, name); err == nil {
			ag = started
		} else {
			log.Printf("[handler] default agent %q not ready for runtime control: %v", name, err)
		}
	}

	if ag != nil {
		h.applyPendingResume(ctx, name, ag, userID)
		sessionID := ensureAgentSession(ctx, ag, userID)
		if sessionID != "" {
			h.recordRuntimeSession(name, userID, sessionID)
			return defaultSessionStatus{profile: name, ag: ag, sessionID: sessionID, source: "agent"}
		}
		return defaultSessionStatus{profile: name, ag: ag, source: "agent_without_session"}
	}

	if !strings.HasSuffix(os.Args[0], ".test") {
		if name != "" {
			if hint, ok := runtime_state.MostRecentSessionForProfile(name); ok {
				return defaultSessionStatus{profile: hint.Profile, sessionID: hint.SessionID, source: "runtime_state"}
			}
		}
		if profile, sessionID, ok := runtime_state.MostRecentSessionForDefaultProfile(); ok {
			return defaultSessionStatus{profile: profile, sessionID: sessionID, source: "runtime_state"}
		}
	}

	return defaultSessionStatus{profile: name, ag: ag, source: "unavailable"}
}

func ensureAgentSession(ctx context.Context, ag agent.Agent, userID string) string {
	if ag == nil || strings.TrimSpace(userID) == "" {
		return ""
	}
	if sessionID := currentAgentSessionID(ag, userID); sessionID != "" {
		return sessionID
	}
	ensurer, ok := ag.(agent.SessionEnsurer)
	if !ok {
		return ""
	}

	ensureCtx := ctx
	if ensureCtx == nil {
		ensureCtx = context.Background()
	}
	ensureCtx, cancel := context.WithTimeout(ensureCtx, 10*time.Second)
	defer cancel()

	sessionID, err := ensurer.EnsureSession(ensureCtx, userID)
	if err != nil {
		log.Printf("[handler] ensure session failed: %v", err)
		return currentAgentSessionID(ag, userID)
	}
	return sessionID
}

func (h *Handler) trackAgentSessionIDDuringTurn(ctx context.Context, ag agent.Agent, userID string, state *runningTurnState) {
	if state == nil {
		return
	}
	inspector, ok := ag.(agent.SessionInspector)
	if !ok {
		return
	}

	update := func() bool {
		sessionID := inspector.CurrentSessionID(userID)
		if sessionID == "" {
			return false
		}
		state.updateSessionID(sessionID)
		snap := state.snapshot()
		h.recordRuntimeSession(snap.agentName, userID, sessionID)
		return true
	}

	if update() {
		return
	}

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if update() {
					return
				}
			}
		}
	}()
}

// isKnownAgent checks if a name corresponds to a configured agent.
func (h *Handler) isKnownAgent(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Check running agents
	if _, ok := h.agents[name]; ok {
		return true
	}
	// Check configured agents (metas)
	for _, meta := range h.agentMetas {
		if meta.Name == name {
			return true
		}
	}
	return false
}

// agentAliases maps short aliases to agent config names.
var agentAliases = map[string]string{
	"cc":  "claude",
	"cx":  "codex",
	"oc":  "openclaw",
	"cs":  "cursor",
	"km":  "kimi",
	"gm":  "gemini",
	"ocd": "opencode",
	"pi":  "pi",
	"cp":  "copilot",
	"dr":  "droid",
	"if":  "iflow",
	"kr":  "kiro",
	"qw":  "qwen",
}

// resolveAlias returns the full agent name for an alias, or the original name if no alias matches.
// Checks custom aliases (from config) first, then built-in aliases.
func (h *Handler) resolveAlias(name string) string {
	h.mu.RLock()
	custom := h.customAliases
	h.mu.RUnlock()
	if custom != nil {
		if full, ok := custom[name]; ok {
			return full
		}
	}
	if full, ok := agentAliases[name]; ok {
		return full
	}
	return name
}

// parseCommand checks if text starts with "/" or "@" followed by agent name(s).
// Supports multiple agents: "@cc @cx hello" returns (["claude","codex"], "hello").
// Returns (agentNames, actualMessage). Aliases are resolved automatically.
// If no command prefix, returns (nil, originalText).
func (h *Handler) parseCommand(text string) ([]string, string) {
	if !strings.HasPrefix(text, "/") && !strings.HasPrefix(text, "@") {
		return nil, text
	}

	// Parse consecutive @name or /name tokens from the start
	var names []string
	rest := text
	for {
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(rest, "/") && !strings.HasPrefix(rest, "@") {
			break
		}

		// Strip prefix
		after := rest[1:]
		idx := strings.IndexAny(after, " /@")
		var token string
		if idx < 0 {
			// Rest is just the name, no message
			token = after
			rest = ""
		} else if after[idx] == '/' || after[idx] == '@' {
			// Next token is another @name or /name
			token = after[:idx]
			rest = after[idx:]
		} else {
			// Space — name ends here
			token = after[:idx]
			rest = strings.TrimSpace(after[idx+1:])
		}

		if token != "" {
			names = append(names, h.resolveAlias(token))
		}

		if rest == "" {
			break
		}
	}

	// Deduplicate names preserving order
	seen := make(map[string]bool)
	unique := names[:0]
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			unique = append(unique, n)
		}
	}

	return unique, rest
}

// HandleMessage processes a single incoming message.
func (h *Handler) HandleMessage(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage) {
	// Only process user messages that are finished
	if msg.MessageType != ilink.MessageTypeUser {
		return
	}
	if msg.MessageState != ilink.MessageStateFinish {
		return
	}

	// Deduplicate by message_id to avoid processing the same message multiple times
	// (voice messages may trigger multiple finish-state updates)
	if msg.MessageID != 0 {
		if _, loaded := h.seenMsgs.LoadOrStore(msg.MessageID, time.Now()); loaded {
			return
		}
		// Clean up old entries periodically (fire-and-forget)
		go h.cleanSeenMsgs()
	}

	// Extract text from item list (text message or voice transcription)
	text := extractText(msg)
	if text == "" {
		if voiceText := extractVoiceText(msg); voiceText != "" {
			text = voiceText
			log.Printf("[handler] voice transcription from %s: %q", msg.FromUserID, truncate(text, 80))
		}
	}
	if text == "" {
		// Check for image message
		if img := extractImage(msg); img != nil && h.saveDir != "" {
			h.handleImageSave(ctx, client, msg, img)
			return
		}
		log.Printf("[handler] received non-text message from %s, skipping", msg.FromUserID)
		return
	}

	log.Printf("[handler] received from %s: %q", msg.FromUserID, truncate(text, 80))

	// Store context token for this user
	h.contextTokens.Store(msg.FromUserID, msg.ContextToken)

	// Generate a clientID for this reply (used to correlate typing → finish)
	clientID := NewClientID()

	// Intercept URLs: save to Linkhoard directly without AI agent
	trimmed := strings.TrimSpace(text)
	if reply, ok := h.handleRuntimeControl(ctx, trimmed, msg.FromUserID); ok {
		if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
			log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
		}
		return
	}
	if h.saveDir != "" && IsURL(trimmed) {
		rawURL := ExtractURL(trimmed)
		if rawURL != "" {
			log.Printf("[handler] saving URL to linkhoard: %s", rawURL)
			title, err := SaveLinkToLinkhoard(ctx, h.saveDir, rawURL)
			var reply string
			if err != nil {
				log.Printf("[handler] link save failed: %v", err)
				reply = fmt.Sprintf("Failed to save link: %v", err)
			} else {
				reply = fmt.Sprintf("Saved link: %s", title)
			}
			if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
				log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
			}
			return
		}
	}

	// Built-in commands (no typing needed)
	if trimmed == "/info" {
		reply := h.buildStatus()
		if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
			log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
		}
		return
	} else if trimmed == "/help" {
		reply := buildHelpText()
		if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
			log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
		}
		return
	} else if trimmed == "/new" || trimmed == "/clear" {
		reply := h.resetDefaultSession(ctx, msg.FromUserID)
		if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
			log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
		}
		return
	} else if strings.HasPrefix(trimmed, "/cwd") {
		reply := h.handleCwd(trimmed)
		if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
			log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
		}
		return
	}

	// Route: "/agentname message" or "@agent1 @agent2 message" -> specific agent(s)
	agentNames, message := h.parseCommand(text)

	// No command prefix -> send to default agent
	if len(agentNames) == 0 {
		h.sendToDefaultAgent(ctx, client, msg, text, clientID)
		return
	}

	// No message -> switch default agent (only first name)
	if message == "" {
		if len(agentNames) == 1 && h.isKnownAgent(agentNames[0]) {
			reply := h.switchDefault(ctx, agentNames[0])
			if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
				log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
			}
		} else if len(agentNames) == 1 && !h.isKnownAgent(agentNames[0]) {
			// Unknown agent -> forward to default
			h.sendToDefaultAgent(ctx, client, msg, text, clientID)
		} else {
			reply := "Usage: specify one agent to switch, or add a message to broadcast"
			if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
				log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
			}
		}
		return
	}

	// Filter to known agents; if single unknown agent -> forward to default
	var knownNames []string
	for _, name := range agentNames {
		if h.isKnownAgent(name) {
			knownNames = append(knownNames, name)
		}
	}
	if len(knownNames) == 0 {
		// No known agents -> forward entire text to default agent
		h.sendToDefaultAgent(ctx, client, msg, text, clientID)
		return
	}

	if len(knownNames) == 1 {
		// Single agent
		h.sendToNamedAgent(ctx, client, msg, knownNames[0], message, clientID)
	} else {
		// Multi-agent broadcast: parallel dispatch, send replies as they arrive
		h.broadcastToAgents(ctx, client, msg, knownNames, message)
	}
}

func (h *Handler) handleRuntimeControl(ctx context.Context, trimmed, userID string) (string, bool) {
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}

	switch fields[0] {
	case "/status":
		if len(fields) != 1 {
			return commandCard(
				"ℹ️ Usage",
				"> Show the compact runtime dashboard.",
				"",
				"```text",
				"/status",
				"```",
			), true
		}
		return h.buildStatusDiagnostics(ctx, userID), true

	case "/balance":
		if len(fields) != 1 {
			return commandCard(
				"ℹ️ Usage",
				"> Show backend account balance when dsproxy supports it.",
				"",
				"```text",
				"/balance",
				"```",
			), true
		}

		balanceReply := runDsproxyCommand(ctx, "balance")
		return formatBalanceReply(balanceReply), true

	case "/now":
		if len(fields) != 1 {
			return commandCard(
				"ℹ️ Usage",
				"> Show the current running turn, or idle session context.",
				"",
				"```text",
				"/now",
				"```",
			), true
		}
		return h.buildNowStatus(ctx, userID), true

	case "/cancel":
		if len(fields) != 1 {
			return commandCard(
				"ℹ️ Usage",
				"> Request cancellation for the current running turn.",
				"",
				"```text",
				"/cancel",
				"```",
			), true
		}
		return h.cancelRunningTurn(userID), true

	case "/model":
		if len(fields) != 2 {
			return commandCard(
				"ℹ️ Usage",
				"> Switch the DeepSeek runtime model and keep the current session.",
				"",
				"```text",
				"/model deepseek-v4-pro",
				"/model deepseek-v4-flash",
				"```",
			), true
		}

		requestedModel := fields[1]
		switch requestedModel {
		case "deepseek-v4-pro", "deepseek-v4-flash":
		default:
			return commandCard(
				"⚠️ Unsupported model",
				"> Not applied.",
				"",
				"```text",
				"Allowed  deepseek-v4-pro | deepseek-v4-flash",
				"```",
			), true
		}

		h.mu.RLock()
		defaultName := h.defaultName
		h.mu.RUnlock()
		if !isDeepSeekRuntimeAgent(defaultName) {
			return commandCard(
				"⛔ DeepSeek only",
				"> `/model` is only supported for `deepseek` and `deepseek-thinking`.",
			), true
		}

		dsproxyReply := runDsproxyCommand(ctx, "config", "set-model", requestedModel)
		weclawStatus := "updated"
		if err := persistDeepSeekRuntimeModel(defaultName, requestedModel); err != nil {
			weclawStatus = "warning: " + err.Error()
		}

		runtimeStatus := "updated for subsequent turns"
		if !h.setRunningAgentModel(defaultName, requestedModel) {
			runtimeStatus = "not updated; use /restart only if the next turn still uses the old model"
		}

		return commandCard(
			"✅ Model updated",
			slashBoldField("Profile", slashInlineCode(defaultName)),
			slashBoldField("Model", slashInlineCode(requestedModel)),
			slashBoldField("Session", "preserved"),
			"",
			"```text",
			"dsproxy  "+commandStatusFromOutput(dsproxyReply),
			"weclaw   "+weclawStatus,
			"runtime  "+runtimeStatus,
			"```",
		), true

	case "/effort":
		if len(fields) != 2 {
			return commandCard(
				"ℹ️ Usage",
				"> Switch reasoning effort and keep the current session.",
				"",
				"```text",
				"/effort high",
				"/effort max",
				"```",
			), true
		}

		requestedEffort := strings.ToLower(fields[1])
		dsproxyEffort := ""
		displayEffort := ""
		switch requestedEffort {
		case "high":
			dsproxyEffort = "high"
			displayEffort = "high"
		case "max":
			dsproxyEffort = "max"
			displayEffort = "max"
		default:
			return commandCard(
				"⚠️ Unsupported effort",
				"> Not applied.",
				"",
				"```text",
				"Allowed  high | max",
				"```",
			), true
		}

		h.mu.RLock()
		defaultName := h.defaultName
		h.mu.RUnlock()
		if !isDeepSeekRuntimeAgent(defaultName) {
			return commandCard(
				"⛔ DeepSeek only",
				"> `/effort` is only supported for `deepseek` and `deepseek-thinking`.",
			), true
		}

		dsproxyReply := runDsproxyCommandForHandler(ctx, "profile", "set-effort", defaultName, dsproxyEffort, "--json")
		displayEffort = effortDisplayFromProfileStatusJSON(dsproxyReply, displayEffort)
		lines := []string{
			slashBoldField("Profile", slashInlineCode(defaultName)),
			slashBoldField("Effort", slashInlineCode(displayEffort)),
			slashBoldField("Session", "preserved"),
			"> Applied through dsproxy profile contract.",
		}
		if status := commandStatusFromOutput(dsproxyReply); commandStatusNeedsAttention(status) {
			lines = append(lines, slashBoldField("Proxy", status))
		}
		return commandCard("✅ Effort updated", lines...), true

	case "/profile":
		if len(fields) != 2 {
			return commandCard(
				"ℹ️ Usage",
				"> Switch the active DeepSeek profile.",
				"",
				"```text",
				"/profile deepseek",
				"/profile deepseek-thinking",
				"```",
			), true
		}
		switch fields[1] {
		case "deepseek", "deepseek-thinking":
			return h.restartProfileAgent(ctx, fields[1], userID), true
		default:
			return commandCard(
				"⚠️ Unsupported profile",
				"> Not applied.",
				"",
				"```text",
				"Allowed  deepseek | deepseek-thinking",
				"```",
			), true
		}

	case "/restart":
		if len(fields) != 1 {
			return commandCard(
				"ℹ️ Usage",
				"> Create a new session for the current profile.",
				"",
				"```text",
				"/restart",
				"```",
			), true
		}
		return h.restartCurrentDefaultAgent(ctx, userID), true
	}

	if strings.HasPrefix(fields[0], "/") && !isDeferredBuiltinSlashCommand(fields[0]) && !h.isKnownSlashAgentCommand(trimmed) {
		return unknownSlashCommandCard(fields[0]), true
	}

	return "", false
}

func isDeferredBuiltinSlashCommand(command string) bool {
	switch command {
	case "/info", "/help", "/new", "/clear":
		return true
	}
	return strings.HasPrefix(command, "/cwd")
}

func (h *Handler) isKnownSlashAgentCommand(trimmed string) bool {
	agentNames, _ := h.parseCommand(trimmed)
	if len(agentNames) == 0 {
		return false
	}
	return h.isKnownAgent(agentNames[0])
}

func unknownSlashCommandCard(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "/"
	}
	lines := []string{
		"> Not sent to agent.",
		slashBoldField("Command", slashInlineCode(command)),
		slashBoldField("Next", "use `/help` to list supported commands"),
	}
	switch strings.ToLower(command) {
	case "/cancle", "/canel", "/cnacel":
		lines = append(lines, slashBoldField("Did you mean", "`/cancel`"))
	case "/stats":
		lines = append(lines, slashBoldField("Did you mean", "`/status`"))
	case "/restat", "/restrat":
		lines = append(lines, slashBoldField("Did you mean", "`/restart`"))
	}
	return commandCard("⚠️ Unknown slash command", lines...)
}

func (h *Handler) buildStatusDiagnostics(ctx context.Context, userID string) string {
	resolved := h.resolveDefaultSessionForRuntimeControl(ctx, userID)
	defaultName := resolved.profile
	ag := resolved.ag
	sessionID := resolved.sessionID

	agentType := "none"
	agentModel := "none"
	if ag != nil {
		info := ag.Info()
		agentType = info.Type
		if info.Model != "" {
			agentModel = info.Model
		}
	}

	proxyRoute := "default"
	proxyEndpoint := "127.0.0.1:8000"
	if defaultName == "deepseek-thinking" {
		proxyRoute = "thinking"
		proxyEndpoint = "127.0.0.1:8001"
	}

	if payload, ok := runDsproxyWeClawStatusPayload(ctx, defaultName, sessionID); ok {
		return h.buildStatusDiagnosticsFromDsproxyTelemetry(defaultName, agentType, agentModel, sessionID, proxyRoute, proxyEndpoint, payload)
	}

	panelLines := buildCompactStatusPanel(ag, userID, 0, proxyRoute, proxyEndpoint, "telemetry unavailable", "balance n/a")
	panelLines = append(panelLines, "Contract unavailable")
	lines := []string{
		slashBoldField("Profile", slashInlineCode(valueOrUnknown(defaultName))+" "+slashInlineCode(slashAgentTypeBadge(agentType))),
		slashBoldField("Model", slashInlineCode(valueOrUnknown(agentModel))+" "+slashInlineCode("unknown")),
		slashBoldField("Session", slashInlineCode(valueOrUnknown(sessionID))),
		"",
	}
	lines = append(lines, visualCommandFence("", panelLines...)...)
	return commandCard("🧩 Status", lines...)
}

func (h *Handler) buildStatusDiagnosticsFromDsproxyTelemetry(defaultName, agentType, agentModel, sessionID, proxyRoute, proxyEndpoint string, payload map[string]any) string {
	model := nestedStringDefault(payload, "unknown", "model", "effective_model")
	if model == "unknown" {
		model = nestedStringDefault(payload, "unknown", "model", "weclaw_display_model")
	}
	if model == "unknown" {
		model = nestedStringDefault(payload, agentModel, "model", "display_model")
	}
	effort := nestedStringDefault(payload, "unknown", "effort", "user_facing")
	if effort == "unknown" {
		effort = nestedStringDefault(payload, "unknown", "effort", "deepseek_reasoning_effort")
	}

	lines := []string{
		slashBoldField("Profile", slashInlineCode(valueOrUnknown(defaultName))+" "+slashInlineCode(slashAgentTypeBadge(agentType))),
		slashBoldField("Model", slashInlineCode(valueOrUnknown(model))+" "+slashInlineCode(slashEffortDisplay(effort))),
		slashBoldField("Session", slashInlineCode(valueOrUnknown(sessionID))),
		"",
	}

	panelLines := buildDsproxyTelemetryPanel(payload, proxyRoute, proxyEndpoint)
	lines = append(lines, visualCommandFence("", panelLines...)...)
	return commandCard("🧩 Status", lines...)
}

var dsproxyCommandRunner = runDsproxyCommand
var dsproxyCommandRunnerAllowExternalInTests bool

func runDsproxyCommandForHandler(ctx context.Context, args ...string) string {
	if strings.HasSuffix(os.Args[0], ".test") && !dsproxyCommandRunnerAllowExternalInTests {
		return fmt.Sprintf("dsproxy %s unavailable during tests", strings.Join(args, " "))
	}
	return dsproxyCommandRunner(ctx, args...)
}

func runDsproxyWeClawStatusPayload(ctx context.Context, profile string, sessionID string) (map[string]any, bool) {
	payload, _, ok := runDsproxyJSONCommand(ctx, dsproxyWeClawStatusArgsForProfile(profile, sessionID)...)
	if !ok {
		return payload, false
	}
	if strings.TrimSpace(sessionID) == "" || !dsproxyWeClawPayloadNeedsRouteFallback(payload) {
		return payload, true
	}

	routePayload, _, routeOK := runDsproxyJSONCommand(ctx, dsproxyWeClawStatusArgsForProfile(profile)...)
	if !routeOK {
		return payload, true
	}
	return mergeDsproxyRouteFallbackPayload(payload, routePayload), true
}

func dsproxyWeClawPayloadNeedsRouteFallback(payload map[string]any) bool {
	runtimeStatus, ok := nestedMap(payload, "runtime_status")
	if ok && !nestedBoolDefault(runtimeStatus, false, "available") {
		return false
	}

	contextLine := formatDsproxyContextLine(payload)
	if strings.Contains(contextLine, " n/a ") || strings.Contains(contextLine, "—/") {
		return true
	}

	guardLines := formatDsproxyCompactionLines(payload)
	if len(guardLines) == 0 {
		return true
	}
	for _, line := range guardLines {
		if strings.Contains(line, " n/a ") || strings.Contains(line, "--/") || strings.Contains(line, "no report") {
			return true
		}
	}
	return false
}

func dsproxyRoutePolicyFallback(routePayload map[string]any) (map[string]any, bool) {
	compaction, ok := nestedMap(routePayload, "compaction")
	if !ok {
		return nil, false
	}

	allowed := map[string]bool{
		"available":               true,
		"policy":                  true,
		"compaction_policy":       true,
		"context_policy":          true,
		"trigger_chars":           true,
		"effective_trigger_chars": true,
		"target_chars":            true,
		"target_context_chars":    true,
		"min_target_chars":        true,
		"max_target_chars":        true,
		"keep_last_messages":      true,
		"keep_last_n_messages":    true,
		"keep_messages":           true,
		"min_new_chars":           true,
		"min_turns":               true,
		"unit":                    true,
		"reason":                  true,
		"action":                  true,
		"source":                  true,
		"source_kind":             true,
		"used_chars_available":    true,
		"trigger_chars_available": true,
		"target_chars_available":  true,
		"keep_messages_available": true,
	}

	filtered := make(map[string]any)
	for key, value := range compaction {
		if allowed[key] {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return nil, false
	}
	return filtered, true
}

func mergeDsproxyRouteFallbackPayload(sessionPayload map[string]any, routePayload map[string]any) map[string]any {
	if sessionPayload == nil {
		sessionPayload = map[string]any{}
	}
	merged := make(map[string]any, len(sessionPayload)+8)
	for key, value := range sessionPayload {
		merged[key] = value
	}

	// Route fallback is intentionally limited to route-level metadata. Do not
	// fallback Context, Details, Tokens, Cost, Compact, or Trim because those
	// fields would make a newly created session look as if it inherited the
	// previous session's observed prompt and payload state.
	for _, key := range []string{
		"pricing",
		"balance",
		"proxy",
		"paths",
		"health",
		"diagnostics",
	} {
		if value, ok := routePayload[key]; ok {
			merged[key] = value
		}
	}
	if policy, ok := dsproxyRoutePolicyFallback(routePayload); ok {
		merged["compaction"] = policy
	}

	return merged
}

func runDsproxyJSONCommand(ctx context.Context, args ...string) (map[string]any, string, bool) {
	text := runDsproxyCommandForHandler(ctx, args...)
	payload, ok := parseJSONMap(text)
	return payload, text, ok
}

func parseJSONMap(text string) (map[string]any, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func dsproxyWeClawStatusArgsForProfile(profile string, sessionIDs ...string) []string {
	var args []string
	if profile == "deepseek-thinking" {
		args = []string{"status", "thinking", "--weclaw-json"}
	} else {
		args = []string{"status", "--weclaw-json"}
	}

	sessionID := ""
	for _, candidate := range sessionIDs {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			sessionID = candidate
			break
		}
	}
	if sessionID != "" {
		args = append(args, "--session-id", sessionID)
	}
	return args
}

func effortDisplayFromProfileStatusJSON(text, fallback string) string {
	payload, ok := parseJSONMap(text)
	if !ok {
		return fallback
	}
	for _, keys := range [][]string{
		{"effort", "user_facing"},
		{"effort", "deepseek_reasoning_effort"},
	} {
		if value := nestedStringDefault(payload, "", keys...); value != "" {
			return slashEffortDisplay(value)
		}
	}
	return fallback
}

func nestedValue(root map[string]any, keys ...string) (any, bool) {
	var cur any = root
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func nestedMap(root map[string]any, keys ...string) (map[string]any, bool) {
	value, ok := nestedValue(root, keys...)
	if !ok {
		return nil, false
	}
	obj, ok := value.(map[string]any)
	return obj, ok
}

func nestedStringDefault(root map[string]any, fallback string, keys ...string) string {
	value, ok := nestedValue(root, keys...)
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return trimFixedDecimal(typed)
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		if text == "" || text == "<nil>" {
			return fallback
		}
		return text
	}
}

func nestedBoolDefault(root map[string]any, fallback bool, keys ...string) bool {
	value, ok := nestedValue(root, keys...)
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "1":
			return true
		case "false", "no", "0":
			return false
		}
	}
	return fallback
}

func nestedInt64Default(root map[string]any, fallback int64, keys ...string) int64 {
	value, ok := nestedValue(root, keys...)
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed
		}
	case string:
		normalized := strings.NewReplacer("_", "", ",", "").Replace(strings.TrimSpace(typed))
		if parsed, err := strconv.ParseInt(normalized, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func nestedFloat64Default(root map[string]any, fallback float64, keys ...string) float64 {
	value, ok := nestedValue(root, keys...)
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return parsed
		}
	case string:
		normalized := strings.NewReplacer("_", "", ",", "").Replace(strings.TrimSpace(typed))
		if parsed, err := strconv.ParseFloat(normalized, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func nestedStringList(root map[string]any, keys ...string) []string {
	value, ok := nestedValue(root, keys...)
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return typed
	case string:
		if strings.TrimSpace(typed) != "" {
			return []string{strings.TrimSpace(typed)}
		}
	}
	return nil
}

func buildDsproxyTelemetryPanel(payload map[string]any, proxyRoute, proxyEndpoint string) []string {
	lines := []string{
		formatDsproxyContextLine(payload),
		formatDsproxyTokensLine(payload),
		formatDsproxyDetailsLine(payload),
		formatDsproxyCostLine(payload),
		formatDsproxyBalanceLine(payload),
	}
	lines = append(lines, buildDsproxyRound3SummaryLines(payload)...)
	lines = append(lines, formatDsproxyCompactionLines(payload)...)

	proxyState := "reachable"
	if status := nestedStringDefault(payload, "", "status"); status != "" && status != "ok" {
		proxyState = status
	}
	lines = append(lines,
		fmt.Sprintf("Proxy    %s · %s · %s", valueOrUnknown(proxyRoute), valueOrUnknown(proxyEndpoint), proxyState),
		"Paths    cfg ~/.weclaw/config.json · log ~/.weclaw/weclaw.log",
	)
	return lines
}

func buildDsproxyRound3SummaryLines(payload map[string]any) []string {
	lines := make([]string, 0, 2)
	if line := formatDsproxyPricingSummaryLine(payload); line != "" {
		lines = append(lines, line)
	}
	if line := formatDsproxyCompactionPolicySummaryLine(payload); line != "" {
		lines = append(lines, line)
	}
	return lines
}

func formatDsproxyPricingSummaryLine(payload map[string]any) string {
	pricing, ok := nestedMap(payload, "pricing")
	if !ok || !nestedBoolDefault(pricing, false, "available") {
		return ""
	}

	parts := make([]string, 0, 3)
	if sourceLabel := pricingSourceLabel(nestedStringDefault(pricing, "unknown", "source_kind")); sourceLabel != "" {
		parts = append(parts, sourceLabel)
	}
	if priceSummary := pricingPricesSummary(pricing); priceSummary != "" {
		parts = append(parts, priceSummary)
	}
	parts = append(parts, "updated "+pricingUpdatedLabel(pricing))

	return "Pricing  " + strings.Join(parts, " · ")
}

func pricingPricesSummary(pricing map[string]any) string {
	prices, ok := nestedMap(pricing, "prices_display")
	if !ok {
		prices, ok = nestedMap(pricing, "effective_prices")
	}
	if !ok {
		prices, ok = nestedMap(pricing, "prices")
	}
	if !ok {
		return ""
	}

	currency := nestedStringDefault(pricing, "", "display_currency")
	if currency == "" {
		currency = nestedStringDefault(prices, "", "display_currency")
	}
	if currency == "" {
		currency = nestedStringDefault(prices, "", "currency")
	}
	if currency == "" || strings.EqualFold(currency, "USD") {
		currency = "CNY"
	}

	parts := make([]string, 0, 4)
	if value, ok := pricingFloatValue(prices, "input_cache_hit"); ok {
		parts = append(parts, "hit "+formatPerMillionPrice(value, currency))
	}
	if value, ok := pricingFloatValue(prices, "input_cache_miss"); ok {
		parts = append(parts, "miss "+formatPerMillionPrice(value, currency))
	} else if value, ok := pricingFloatValue(prices, "input"); ok {
		parts = append(parts, "in "+formatPerMillionPrice(value, currency))
	}
	if value, ok := pricingFloatValue(prices, "output"); ok {
		parts = append(parts, "out "+formatPerMillionPrice(value, currency))
	}
	if value, ok := pricingFloatValue(prices, "reasoning"); ok {
		parts = append(parts, "reason "+formatPerMillionPrice(value, currency))
	}

	return strings.Join(parts, " ")
}

func pricingFloatValue(root map[string]any, keys ...string) (float64, bool) {
	value, ok := nestedValue(root, keys...)
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		normalized := strings.NewReplacer("_", "", ",", "").Replace(strings.TrimSpace(typed))
		if normalized == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(normalized, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func formatPerMillionPrice(value float64, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	amount := trimMoneyDecimal(value)
	switch currency {
	case "CNY", "RMB", "CNH":
		return "￥" + amount + "/M"
	case "USD":
		return "$" + amount + "/M"
	case "":
		return amount + "/M"
	default:
		return currency + " " + amount + "/M"
	}
}

func formatDsproxyCompactionPolicySummaryLine(payload map[string]any) string {
	compaction, ok := nestedMap(payload, "compaction")
	if !ok || !nestedBoolDefault(compaction, false, "available") {
		return ""
	}

	policy := nestedStringDefault(compaction, "", "runtime_context", "compaction", "last_report", "policy")
	if policy == "" {
		policy = nestedStringDefault(compaction, "", "runtime_context", "compaction", "last_report", "policy_decision", "policy")
	}
	if policy == "" {
		policy = nestedStringDefault(compaction, "", "runtime_context", "compaction", "config", "policy")
	}
	if policy == "" {
		return ""
	}

	trigger := nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "effective_trigger_chars")
	if trigger <= 0 {
		trigger = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "trigger_chars")
	}
	if trigger <= 0 {
		trigger = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "config", "trigger_chars")
	}

	target := nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "effective_target_chars")
	if target <= 0 {
		target = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "target_chars")
	}
	if target <= 0 {
		target = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "config", "target_chars")
	}

	keep := nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "keep_recent_messages")
	if keep <= 0 {
		keep = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "config", "keep_recent_messages")
	}

	parts := []string{policy}
	if trigger > 0 {
		parts = append(parts, "trigger "+formatCompactStatusNumber(trigger)+" chars")
	}
	if target > 0 {
		parts = append(parts, "target "+formatCompactStatusNumber(target)+" chars")
	}
	if keep > 0 {
		parts = append(parts, fmt.Sprintf("keep ⤒%d msgs", keep))
	}
	return "Policy   " + strings.Join(parts, " · ")
}

func pricingSourceLabel(sourceKind string) string {
	switch sourceKind {
	case "project_default_config", "project_default_pricing_config":
		return "default config"
	case "bundled_official_docs_snapshot", "bundled_official_snapshot":
		return ""
	case "official_docs_html", "official_live_cache", "official_cache":
		return "official cache"
	case "user_cache", "cache", "pricing_cache":
		return "pricing cache"
	case "":
		return "unknown"
	default:
		return compactCommandOutput(sourceKind, 32)
	}
}

func pricingUpdatedLabel(pricing map[string]any) string {
	for _, key := range []string{"fetched_at", "updated_at", "snapshot_created_at", "pricing_updated_at"} {
		if value, ok := nestedValue(pricing, key); ok && value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				if strings.Contains(text, "T") {
					text = strings.Split(text, "T")[0]
				}
				return compactCommandOutput(text, 32)
			}
		}
	}
	return "n/a"
}

func formatCompactStatusNumber(value int64) string {
	if value >= 1000000 {
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", float64(value)/1000000), "0"), ".") + "M"
	}
	if value >= 1000 {
		return fmt.Sprintf("%dk", value/1000)
	}
	return fmt.Sprintf("%d", value)
}

func availabilityText(root map[string]any) string {
	if root == nil {
		return "n/a"
	}
	if _, ok := nestedValue(root, "available"); !ok {
		return "n/a"
	}
	return availabilityBoolText(nestedBoolDefault(root, false, "available"))
}

func availabilityBoolText(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func formatDsproxyContextLine(payload map[string]any) string {
	limit := dsproxyContextDisplayLimit(payload)
	used, ok := dsproxyContextUsedTokens(payload)
	if !ok {
		return fmt.Sprintf(
			"Context  [%s]  n/a  —/%s",
			formatCommandProgressBar(0, limit, 20),
			formatContextLimit(limit),
		)
	}

	return fmt.Sprintf(
		"Context  [%s]  %s  %s/%s",
		formatCommandProgressBar(used, limit, 20),
		formatTokenPercent(used, limit),
		formatTokenCount(maxInt64(used, 0)),
		formatContextLimit(limit),
	)
}

func dsproxyContextDisplayLimit(payload map[string]any) int64 {
	for _, keys := range [][]string{
		{"context_window", "display_limit_tokens"},
		{"context_window", "limit_explanation", "display_limit_tokens"},
		{"context_window", "effective_safe_window_tokens"},
	} {
		if value := nestedInt64Default(payload, 0, keys...); value > 0 {
			return value
		}
	}
	return 0
}

func dsproxyContextUsedTokens(payload map[string]any) (int64, bool) {
	contextWindow, ok := nestedMap(payload, "context_window")
	if !ok || !nestedBoolDefault(contextWindow, false, "used_tokens_available") {
		return 0, false
	}
	used := nestedInt64Default(contextWindow, -1, "used_tokens")
	if used < 0 {
		return 0, false
	}
	return used, true
}

func dsproxyContextUsedTokensEstimated(payload map[string]any) bool {
	contextWindow, ok := nestedMap(payload, "context_window")
	if !ok {
		return false
	}
	if nestedBoolDefault(contextWindow, false, "used_tokens_is_estimated") {
		return true
	}
	if nestedBoolDefault(contextWindow, false, "is_estimated") {
		return true
	}
	if latest, ok := nestedMap(contextWindow, "latest_upstream_prompt_tokens"); ok {
		return nestedBoolDefault(latest, false, "is_estimated_for_context_window")
	}
	return false
}

func tokenBucketTotal(bucket map[string]any) (int64, bool) {
	if bucket == nil {
		return 0, false
	}
	if total, ok := tokenTotalFromMap(bucket); ok {
		return total, true
	}
	if summary, ok := nestedMap(bucket, "summary"); ok {
		if total, ok := tokenTotalFromMap(summary); ok {
			return total, true
		}
	}
	return 0, false
}

func tokenTotalFromMap(bucket map[string]any) (int64, bool) {
	if bucket == nil {
		return 0, false
	}
	for _, key := range []string{"total_tokens", "total", "tokens", "provider_total_tokens"} {
		if value := nestedInt64Default(bucket, -1, key); value >= 0 {
			return value, true
		}
	}
	input := nestedInt64Default(bucket, 0, "input_tokens")
	if input == 0 {
		input = nestedInt64Default(bucket, 0, "prompt_tokens")
	}
	output := nestedInt64Default(bucket, 0, "output_tokens")
	if output == 0 {
		output = nestedInt64Default(bucket, 0, "completion_tokens")
	}
	reasoning := nestedInt64Default(bucket, 0, "reasoning_tokens")
	total := input + output + reasoning
	if total > 0 {
		return total, true
	}
	return 0, false
}

func dsproxyTokenSectionTotal(section map[string]any) (int64, bool) {
	if section == nil || !nestedBoolDefault(section, true, "available") {
		return 0, false
	}
	for _, keys := range [][]string{
		{"summary", "total_tokens"},
		{"usage", "total_tokens"},
		{"provider_usage", "total_tokens"},
		{"total_tokens"},
		{"tokens"},
	} {
		if value, ok := pricingFloatValue(section, keys...); ok {
			if value < 0 {
				return 0, false
			}
			return int64(value + 0.5), true
		}
	}
	return 0, false
}

func dsproxyTokenSectionText(section map[string]any) string {
	for _, key := range []string{"total_tokens", "total", "tokens", "provider_total_tokens"} {
		if value := nestedInt64Default(section, -1, key); value >= 0 {
			return formatTokenCount(value)
		}
	}
	if total, ok := dsproxyTokenSectionTotal(section); ok {
		return formatTokenCount(total)
	}
	return "n/a"
}

func dsproxyTokenMap(payload map[string]any, keys ...string) (map[string]any, bool) {
	return nestedMap(payload, keys...)
}

func dsproxyCacheSectionHasProviderFields(section map[string]any) bool {
	if section == nil {
		return false
	}
	if _, ok := pricingFloatValue(section, "cache_hit_ratio"); ok {
		return true
	}
	for _, key := range []string{"prompt_cache_hit_tokens", "prompt_cache_miss_tokens", "cached_tokens", "prompt_tokens"} {
		if nestedInt64Default(section, -1, key) >= 0 {
			return true
		}
	}
	return false
}

func dsproxyCacheSectionFor(tokens map[string]any, key string) (map[string]any, bool) {
	if section, ok := nestedMap(tokens, "cache", key); ok &&
		nestedBoolDefault(section, false, "available") &&
		dsproxyMapScopeIsCurrentSession(section) &&
		dsproxyCacheSectionHasProviderFields(section) {
		return section, true
	}
	if section, ok := dsproxyTokenMap(tokens, key); ok &&
		nestedBoolDefault(section, false, "available") &&
		dsproxyMapScopeIsCurrentSession(section) {
		if cache, ok := nestedMap(section, "cache"); ok &&
			nestedBoolDefault(cache, false, "available") &&
			dsproxyCacheSectionHasProviderFields(cache) {
			return cache, true
		}
		if dsproxyCacheSectionHasProviderFields(section) {
			return section, true
		}
	}
	return nil, false
}

func dsproxyCacheTotalPromptTokens(section map[string]any) int64 {
	for _, key := range []string{"prompt_tokens", "provider_prompt_tokens", "input_tokens"} {
		if value := nestedInt64Default(section, -1, key); value >= 0 {
			return value
		}
	}
	if hit := nestedInt64Default(section, -1, "prompt_cache_hit_tokens"); hit >= 0 {
		miss := nestedInt64Default(section, 0, "prompt_cache_miss_tokens")
		return hit + miss
	}
	if cached := nestedInt64Default(section, -1, "cached_tokens"); cached >= 0 {
		miss := nestedInt64Default(section, 0, "prompt_cache_miss_tokens")
		return cached + miss
	}
	for _, key := range []string{"total_tokens", "total", "tokens", "provider_total_tokens"} {
		if value := nestedInt64Default(section, -1, key); value >= 0 {
			return value
		}
	}
	return -1
}

func dsproxyCacheHitPercentText(section map[string]any) string {
	if ratio, ok := pricingFloatValue(section, "cache_hit_ratio"); ok && ratio >= 0 {
		if ratio <= 1 {
			ratio *= 100
		}
		return fmt.Sprintf("%.1f%%", ratio)
	}
	total := dsproxyCacheTotalPromptTokens(section)
	if total == 0 {
		return "0.0%"
	}
	hit := nestedInt64Default(section, -1, "prompt_cache_hit_tokens")
	if hit < 0 {
		hit = nestedInt64Default(section, -1, "cached_tokens")
	}
	if total > 0 && hit >= 0 {
		return fmt.Sprintf("%.1f%%", float64(hit)*100/float64(total))
	}
	return "n/a"
}

func dsproxyCacheAwareTokenText(section map[string]any) string {
	if section == nil || !nestedBoolDefault(section, true, "available") {
		return "hit~n/a/total~n/a"
	}
	hitText := dsproxyCacheHitPercentText(section)
	total := dsproxyCacheTotalPromptTokens(section)
	totalText := "n/a"
	if total >= 0 {
		totalText = formatTokenCount(total)
	}
	return fmt.Sprintf("hit~%s/total~%s", hitText, totalText)
}

func formatDsproxyTokensLine(payload map[string]any) string {
	tokens, ok := nestedMap(payload, "tokens")
	if !ok {
		return "Tokens   last n/a  session n/a  aux n/a"
	}

	lastText := "n/a"
	if section, ok := dsproxyCacheSectionFor(tokens, "last_turn"); ok {
		lastText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyCacheSectionFor(tokens, "latest_primary_turn"); ok {
		lastText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyTokenMap(tokens, "last_turn"); ok {
		lastText = dsproxyTokenSectionText(section)
	} else if section, ok := dsproxyTokenMap(tokens, "latest_primary_turn"); ok {
		lastText = dsproxyTokenSectionText(section)
	}

	sessionText := "n/a"
	if section, ok := dsproxyCacheSectionFor(tokens, "session"); ok {
		sessionText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyCacheSectionFor(tokens, "session_total"); ok {
		sessionText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyTokenMap(tokens, "session"); ok && nestedBoolDefault(section, false, "available") {
		sessionText = dsproxyTokenSectionText(section)
	}

	auxText := "n/a"
	if section, ok := dsproxyCacheSectionFor(tokens, "auxiliary_model_calls"); ok {
		auxText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyCacheSectionFor(tokens, "latest_auxiliary_call"); ok {
		auxText = dsproxyCacheAwareTokenText(section)
	} else if section, ok := dsproxyTokenMap(tokens, "auxiliary_model_calls"); ok &&
		nestedBoolDefault(section, false, "available") &&
		dsproxyMapScopeIsCurrentSession(section) {
		auxText = dsproxyTokenSectionText(section)
	} else if section, ok := dsproxyTokenMap(tokens, "latest_auxiliary_call"); ok &&
		nestedBoolDefault(section, false, "available") &&
		dsproxyMapScopeIsCurrentSession(section) {
		auxText = dsproxyTokenSectionText(section)
	}

	return fmt.Sprintf("Tokens   last %s  session %s  aux %s", lastText, sessionText, auxText)
}

func dsproxyMapScope(mapValue map[string]any) string {
	for _, key := range []string{"scope", "ledger_scope"} {
		value := strings.ToLower(strings.TrimSpace(nestedStringDefault(mapValue, "", key)))
		if value != "" {
			return value
		}
	}
	return ""
}

func dsproxyMapScopeIsCurrentSession(mapValue map[string]any) bool {
	scope := dsproxyMapScope(mapValue)
	return scope == "current_session" || scope == "session"
}

func dsproxySessionIDsMatch(expected string, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	return expected == "" || actual == "" || expected == actual
}

func dsproxyPromptSplitMatchesCurrentSession(tokens map[string]any, split map[string]any) bool {
	if !dsproxyMapScopeIsCurrentSession(split) {
		return false
	}
	splitSessionID := nestedStringDefault(split, "", "session_id")
	sessionID := nestedStringDefault(tokens, "", "session", "session_id")
	if sessionID == "" {
		sessionID = nestedStringDefault(tokens, "", "session_total", "session_id")
	}
	return dsproxySessionIDsMatch(sessionID, splitSessionID)
}

func dsproxyCostCurrentSessionScope(cost map[string]any, session map[string]any) bool {
	return dsproxyMapScopeIsCurrentSession(cost) || dsproxyMapScopeIsCurrentSession(session)
}

func dsproxyOriginComponentTokens(components map[string]any, key string) int64 {
	if value := nestedInt64Default(components, -1, key); value >= 0 {
		return value
	}
	for _, field := range []string{"tokens", "token_count", "total_tokens"} {
		if value := nestedInt64Default(components, -1, key, field); value >= 0 {
			return value
		}
	}
	return 0
}

func dsproxyOriginComponentAbsTokens(components map[string]any, key string) int64 {
	if value := nestedInt64Default(components, -1, key, "abs_tokens"); value >= 0 {
		return value
	}
	value := dsproxyOriginComponentTokens(components, key)
	if value < 0 {
		return -value
	}
	return value
}

func dsproxyOriginComponentWithinTolerance(components map[string]any, key string) bool {
	if nestedBoolDefault(components, false, key, "within_tolerance") {
		return true
	}
	if nestedBoolDefault(components, false, key, "is_within_tolerance") {
		return true
	}
	if nestedBoolDefault(components, false, key, "hide_when_within_tolerance") {
		return dsproxyOriginComponentAbsTokens(components, key) == 0
	}
	return false
}

func dsproxyDetailsOriginBreakdownMatchesCurrentSession(tokens map[string]any, breakdown map[string]any) bool {
	if !nestedBoolDefault(breakdown, false, "available") {
		return false
	}
	if nestedStringDefault(breakdown, "", "display_semantics") != "token_origin_breakdown_not_classified_total" {
		return false
	}
	if !dsproxyMapScopeIsCurrentSession(breakdown) {
		return false
	}
	breakdownSessionID := nestedStringDefault(breakdown, "", "session_id")
	sessionID := nestedStringDefault(tokens, "", "session", "session_id")
	if sessionID == "" {
		sessionID = nestedStringDefault(tokens, "", "session_total", "session_id")
	}
	return dsproxySessionIDsMatch(sessionID, breakdownSessionID)
}

func formatDsproxyDetailsOriginBreakdownLine(tokens map[string]any) (string, bool) {
	breakdown, ok := nestedMap(tokens, "prompt_reconciliation", "details_origin_breakdown")
	if !ok || !dsproxyDetailsOriginBreakdownMatchesCurrentSession(tokens, breakdown) {
		return "", false
	}
	components, ok := nestedMap(breakdown, "components")
	if !ok {
		return "", false
	}

	user := dsproxyOriginComponentTokens(components, "user")
	history := dsproxyOriginComponentTokens(components, "history")
	system := dsproxyOriginComponentTokens(components, "system")
	developer := dsproxyOriginComponentTokens(components, "developer")
	compaction := dsproxyOriginComponentTokens(components, "compaction_summary")
	environment := dsproxyOriginComponentTokens(components, "environment") +
		dsproxyOriginComponentTokens(components, "runtime_injected") +
		dsproxyOriginComponentTokens(components, "other_prompt")
	tools := dsproxyOriginComponentTokens(components, "tool_output") +
		dsproxyOriginComponentTokens(components, "tools_schema")
	overhead := dsproxyOriginComponentTokens(components, "message_protocol_overhead")

	parts := []string{
		"Details",
		fmt.Sprintf("user~%s", formatTokenCount(user)),
		fmt.Sprintf("hist~%s", formatTokenCount(history)),
		fmt.Sprintf("sys~%s", formatTokenCount(system)),
		fmt.Sprintf("env~%s", formatTokenCount(environment)),
		fmt.Sprintf("tools~%s", formatTokenCount(tools)),
		fmt.Sprintf("overhead~%s", formatTokenCount(overhead)),
	}
	if developer > 0 {
		parts = append(parts, fmt.Sprintf("dev~%s", formatTokenCount(developer)))
	}
	if compaction > 0 {
		parts = append(parts, fmt.Sprintf("comp~%s", formatTokenCount(compaction)))
	}
	if residual := dsproxyOriginComponentTokens(components, "provider_residual"); residual > 0 &&
		!dsproxyOriginComponentWithinTolerance(components, "provider_residual") {
		parts = append(parts, fmt.Sprintf("resid~%s", formatTokenCount(residual)))
	}

	return strings.Join(parts, "  "), true
}

func dsproxyDetailsCoverageSuffix(split map[string]any) string {
	categoriesSum := nestedInt64Default(split, -1, "categories_sum_tokens")
	providerReference := nestedInt64Default(split, 0, "provider_reference_tokens")
	if categoriesSum < 0 || providerReference <= 0 {
		return ""
	}
	label := "partial"
	if nestedBoolDefault(split, false, "coverage_complete") {
		label = "covered"
	}
	return fmt.Sprintf("  %s~%s/%s", label, formatTokenCount(categoriesSum), formatTokenCount(providerReference))
}

func formatDsproxyDetailsLine(payload map[string]any) string {
	tokens, ok := nestedMap(payload, "tokens")
	if !ok {
		return "Details  n/a"
	}

	if line, ok := formatDsproxyDetailsOriginBreakdownLine(tokens); ok {
		return line
	}

	profileTokenizer, profileOK := nestedMap(tokens, "profile_tokenizer")
	if !profileOK || !nestedBoolDefault(profileTokenizer, false, "available") {
		return "Details  n/a · tokenizer unavailable"
	}

	split, splitOK := nestedMap(tokens, "prompt_subcategory_split")
	if !splitOK || !nestedBoolDefault(split, false, "available") {
		reason := nestedStringDefault(split, "", "reason")
		if reason == "" {
			reason = nestedStringDefault(profileTokenizer, "", "summary", "reason")
		}
		if reason == "profile_tokenizer_available_but_no_observed_prompt" {
			return "Details  n/a · waiting first prompt"
		}
		return "Details  n/a"
	}

	if !dsproxyPromptSplitMatchesCurrentSession(tokens, split) {
		return "Details  n/a"
	}

	categories, ok := nestedMap(split, "categories")
	if !ok {
		return "Details  n/a"
	}

	user := promptCategoryTokens(categories, "user")
	history := promptCategoryTokens(categories, "assistant_history")
	tool := promptCategoryTokens(categories, "tool_output")
	system := promptCategoryTokens(categories, "system")
	developer := promptCategoryTokens(categories, "developer")
	compaction := promptCategoryTokens(categories, "compaction_summary")
	other := promptCategoryTokens(categories, "environment") +
		promptCategoryTokens(categories, "runtime_injected") +
		promptCategoryTokens(categories, "other_prompt")
	coverageSuffix := dsproxyDetailsCoverageSuffix(split)

	return fmt.Sprintf(
		"Details  user~%s  hist~%s  tool~%s  sys~%s  dev~%s  comp~%s  other~%s%s",
		formatTokenCount(user),
		formatTokenCount(history),
		formatTokenCount(tool),
		formatTokenCount(system),
		formatTokenCount(developer),
		formatTokenCount(compaction),
		formatTokenCount(other),
		coverageSuffix,
	)
}

func promptCategoryTokens(categories map[string]any, key string) int64 {
	if value := nestedInt64Default(categories, -1, key); value >= 0 {
		return value
	}
	for _, field := range []string{"tokens", "token_count", "total_tokens"} {
		if value := nestedInt64Default(categories, -1, key, field); value >= 0 {
			return value
		}
	}
	return 0
}

func formatTokenBucket(label string, bucket map[string]any) string {
	if bucket == nil {
		return label + " n/a"
	}
	if !nestedBoolDefault(bucket, false, "available") {
		return label + " n/a"
	}
	if total, ok := tokenBucketTotal(bucket); ok {
		return label + " " + formatTokenCount(total)
	}
	return label + " n/a"
}

func formatDsproxyCostLine(payload map[string]any) string {
	cost, ok := nestedMap(payload, "cost")
	if !ok || !nestedBoolDefault(cost, false, "available") {
		return "Cost     session~n/a  last~n/a  aux~n/a  total~n/a"
	}

	currency := nestedStringDefault(cost, nestedStringDefault(cost, "USD", "currency"), "display_currency")
	if currency == "" {
		currency = "CNY"
	}

	sessionText := "n/a"
	totalText := "n/a"
	currentSessionScope := false
	if session, ok := nestedMap(cost, "session"); ok && nestedBoolDefault(session, false, "available") && dsproxyCostCurrentSessionScope(cost, session) {
		currentSessionScope = true
		amountCurrency := nestedStringDefault(session, currency, "display_currency")
		if value, ok := pricingFloatValue(session, "estimated_cost"); ok {
			sessionText = formatMoney(value, amountCurrency)
		} else if amount, ok := nestedMap(session, "amount"); ok {
			amountCurrency = nestedStringDefault(amount, amountCurrency, "display_currency")
			if value, ok := pricingFloatValue(amount, "amount"); ok {
				sessionText = formatMoney(value, amountCurrency)
			}
		} else if value, ok := pricingFloatValue(session, "amount"); ok {
			sessionText = formatMoney(value, amountCurrency)
		}
	} else if dsproxyMapScopeIsCurrentSession(cost) {
		currentSessionScope = true
		sessionText = formatMoney(nestedFloat64Default(cost, 0, "session_estimated_cost"), currency)
	}

	last := nestedFloat64Default(cost, 0, "last_turn_estimated_cost")
	aux := nestedFloat64Default(cost, 0, "auxiliary_estimated_cost")

	if currentSessionScope {
		if total, ok := pricingFloatValue(cost, "total_estimated_cost"); ok {
			totalText = formatMoney(total, currency)
		} else if total, ok := pricingFloatValue(cost, "cash_estimated_cost"); ok {
			totalText = formatMoney(total, currency)
		} else if amount, ok := nestedMap(cost, "amounts", "cash"); ok {
			amountCurrency := nestedStringDefault(amount, currency, "display_currency")
			if total, ok := pricingFloatValue(amount, "amount"); ok {
				totalText = formatMoney(total, amountCurrency)
			}
		}
	}

	return fmt.Sprintf(
		"Cost     session~%s  last~%s  aux~%s  total~%s",
		sessionText,
		formatMoney(last, currency),
		formatMoney(aux, currency),
		totalText,
	)
}

func formatMoney(value float64, currency string) string {
	return formatCurrencyAmount(formatMoneyDisplayAmount(value), currency)
}

func formatCurrencyAmount(amount string, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	switch currency {
	case "USD":
		return "$" + amount
	case "CNY", "RMB", "CNH":
		return "￥" + amount
	case "":
		return amount
	default:
		return currency + " " + amount
	}
}

func formatMoneyDisplayAmount(value float64) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs == 0:
		return "0"
	case abs < 0.001:
		return trimTrailingMoneyZeros(fmt.Sprintf("%.6f", value))
	case abs < 1:
		return trimTrailingMoneyZeros(fmt.Sprintf("%.4f", value))
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

func trimTrailingMoneyZeros(value string) string {
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "-0" {
		return "0"
	}
	return value
}

func trimMoneyDecimal(value float64) string {
	if value == 0 {
		return "0"
	}
	abs := value
	if abs < 0 {
		abs = -abs
	}
	decimals := 2
	switch {
	case abs < 0.01:
		decimals = 6
	case abs < 1:
		decimals = 4
	}
	text := fmt.Sprintf("%.*f", decimals, value)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "-0" {
		return "0"
	}
	return text
}

func formatDsproxyBalanceLine(payload map[string]any) string {
	balance, ok := nestedMap(payload, "balance")
	if !ok {
		if costBalance, ok := nestedMap(payload, "cost", "balance"); ok {
			balance = costBalance
		} else {
			return "Balance  n/a"
		}
	}
	if !nestedBoolDefault(balance, false, "available") {
		return "Balance  n/a"
	}

	currency := nestedStringDefault(balance, "CNY", "currency")
	if amount, ok := pricingFloatValue(balance, "amount"); ok {
		return "Balance  " + formatMoney(amount, currency)
	}
	if amount, ok := pricingFloatValue(balance, "total_balance"); ok {
		return "Balance  " + formatMoney(amount, currency)
	}
	if text := strings.TrimSpace(nestedStringDefault(balance, "", "display")); text != "" {
		fields := strings.Fields(text)
		if len(fields) == 2 {
			if amount, err := strconv.ParseFloat(fields[0], 64); err == nil {
				return "Balance  " + formatMoney(amount, fields[1])
			}
		}
		return "Balance  " + compactCommandOutput(text, 64)
	}
	if value := nestedStringDefault(balance, "", "balance"); value != "" {
		return "Balance  " + compactCommandOutput(value, 64)
	}
	return "Balance  available"
}

func formatDsproxyCompactionLines(payload map[string]any) []string {
	if lines, ok := formatDsproxyRuntimePayloadGuardLines(payload); ok {
		return lines
	}

	compaction, ok := nestedMap(payload, "compaction")
	if !ok || !nestedBoolDefault(compaction, false, "available") {
		return []string{"Compact n/a"}
	}
	unit := nestedStringDefault(compaction, "chars", "unit")

	before := nestedInt64Default(compaction, -1, "runtime_context", "compaction", "last_report", "before_chars")
	trigger := nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "effective_trigger_chars")
	if trigger <= 0 {
		trigger = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "last_report", "trigger_chars")
	}
	if trigger <= 0 {
		trigger = nestedInt64Default(compaction, 0, "runtime_context", "compaction", "config", "trigger_chars")
	}
	reason := nestedStringDefault(compaction, "", "runtime_context", "compaction", "last_report", "reason")
	if reason == "" {
		if !nestedBoolDefault(compaction, false, "runtime_context", "compaction", "last_report", "exists") {
			reason = "no report"
		} else {
			reason = "unknown"
		}
	}

	trimBefore := nestedInt64Default(compaction, -1, "runtime_context", "trimming", "last_report", "before_chars")
	trimMax := nestedInt64Default(compaction, 0, "runtime_context", "trimming", "last_report", "max_context_chars")
	if trimMax <= 0 {
		trimMax = nestedInt64Default(compaction, 0, "runtime_context", "trimming", "config", "max_context_chars")
	}
	removed := nestedInt64Default(compaction, 0, "runtime_context", "trimming", "last_report", "chars_removed")
	trimReason := "removed " + formatTokenCount(removed)
	if trimBefore < 0 && !nestedBoolDefault(compaction, false, "runtime_context", "trimming", "last_report", "exists") {
		trimReason = "no report"
	}

	return []string{
		fmt.Sprintf(
			"Compact [%s]  %s  %s/%s %s · %s",
			formatCommandProgressBar(maxInt64(before, 0), trigger, 20),
			formatStatusPercent(before, trigger),
			formatStatusCount(before),
			formatContextLimit(trigger),
			unit,
			compactCommandOutput(displayRuntimePayloadGuardStatus(reason), 36),
		),
		fmt.Sprintf(
			"Trim    [%s]  %s  %s/%s %s · %s",
			formatCommandProgressBar(maxInt64(trimBefore, 0), trimMax, 20),
			formatStatusPercent(trimBefore, trimMax),
			formatStatusCount(trimBefore),
			formatContextLimit(trimMax),
			unit,
			displayRuntimePayloadGuardStatus(trimReason),
		),
	}
}

func runtimePayloadGuardProgressValues(section map[string]any, legacyNumeratorKeys [][]string, legacyDenominatorKeys [][]string) (int64, int64, string) {
	numerator := nestedInt64Default(section, -1, "display_numerator_chars")
	if numerator < 0 {
		numerator = nestedInt64Default(section, -1, "progress_numerator_chars")
	}
	if numerator < 0 {
		numerator = nestedInt64Default(section, -1, "retention_numerator_chars")
	}
	for _, keys := range legacyNumeratorKeys {
		if numerator >= 0 {
			break
		}
		numerator = nestedInt64Default(section, -1, keys...)
	}

	denominator := nestedInt64Default(section, 0, "display_denominator_chars")
	if denominator <= 0 {
		denominator = nestedInt64Default(section, 0, "progress_denominator_chars")
	}
	if denominator <= 0 {
		denominator = nestedInt64Default(section, 0, "retention_denominator_chars")
	}
	for _, keys := range legacyDenominatorKeys {
		if denominator > 0 {
			break
		}
		denominator = nestedInt64Default(section, 0, keys...)
	}

	percent := ""
	for _, key := range []string{"display_ratio", "progress_ratio", "retention_ratio"} {
		if ratio, ok := pricingFloatValue(section, key); ok {
			percent = formatRuntimeProgressRatioPercent(ratio)
			break
		}
	}
	if percent == "" {
		percent = formatStatusPercent(numerator, denominator)
	}
	return numerator, denominator, percent
}

func formatRuntimeProgressRatioPercent(ratio float64) string {
	if ratio < 0 {
		return ""
	}
	if ratio > 1 {
		return fmt.Sprintf("%.1f%%", ratio)
	}
	return fmt.Sprintf("%.1f%%", ratio*100)
}

func formatDsproxyRuntimePayloadGuardLines(payload map[string]any) ([]string, bool) {
	guard, ok := nestedMap(payload, "runtime_payload_guard")
	if !ok || !nestedBoolDefault(guard, false, "available") {
		return nil, false
	}
	unit := nestedStringDefault(guard, "chars", "unit")
	lines := make([]string, 0, 2)

	compaction, compactionOK := nestedMap(guard, "compaction")
	if !compactionOK {
		compaction, compactionOK = nestedMap(guard, "compact")
	}
	if compactionOK && nestedBoolDefault(compaction, false, "available") {
		current, trigger, percent := runtimePayloadGuardProgressValues(
			compaction,
			[][]string{{"current_chars"}},
			[][]string{{"trigger_chars"}, {"effective_trigger_chars"}},
		)
		if current >= 0 && trigger > 0 {
			status := displayRuntimePayloadGuardStatus(nestedStringDefault(compaction, "unknown", "status"))
			lines = append(lines, fmt.Sprintf(
				"Compact [%s]  %s  %s/%s %s · %s",
				formatCommandProgressBar(current, trigger, 20),
				percent,
				formatStatusCount(current),
				formatContextLimit(trigger),
				unit,
				compactCommandOutput(status, 36),
			))
		}
	}

	trimming, trimmingOK := nestedMap(guard, "trimming")
	if !trimmingOK {
		trimming, trimmingOK = nestedMap(guard, "trim")
	}
	if trimmingOK && nestedBoolDefault(trimming, false, "available") {
		current, limit, percent := runtimePayloadGuardProgressValues(
			trimming,
			[][]string{{"current_chars"}},
			[][]string{{"max_context_chars"}},
		)
		if current >= 0 && limit > 0 {
			status := displayRuntimePayloadGuardStatus(nestedStringDefault(trimming, "unknown", "status"))
			lines = append(lines, fmt.Sprintf(
				"Trim    [%s]  %s  %s/%s %s · %s",
				formatCommandProgressBar(current, limit, 20),
				percent,
				formatStatusCount(current),
				formatContextLimit(limit),
				unit,
				compactCommandOutput(status, 36),
			))
		}
	}

	if len(lines) == 0 {
		return nil, false
	}
	return lines, true
}

func displayRuntimePayloadGuardStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "unknown"
	}
	return strings.ReplaceAll(status, "_", " ")
}

func formatStatusPercent(used, limit int64) string {
	if used < 0 {
		return "n/a"
	}
	return formatTokenPercent(used, limit)
}

func formatStatusCount(value int64) string {
	if value < 0 {
		return "--"
	}
	return formatTokenCount(value)
}

func resolveDsproxyBinary() string {
	if path, err := exec.LookPath("dsproxy"); err == nil {
		return path
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidates := []string{
			filepath.Join(home, "bin", "dsproxy"),
			filepath.Join(home, ".local", "bin", "dsproxy"),
			filepath.Join(home, ".cargo", "bin", "dsproxy"),
		}
		for _, candidate := range candidates {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	for _, candidate := range []string{"/usr/local/bin/dsproxy", "/usr/bin/dsproxy"} {
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}

	return "dsproxy"
}

func runDsproxyCommand(ctx context.Context, args ...string) string {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	dsproxy := resolveDsproxyBinary()
	cmd := exec.CommandContext(runCtx, dsproxy, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if runCtx.Err() == context.DeadlineExceeded {
		if text != "" {
			return fmt.Sprintf("dsproxy %s timed out.\n%s", strings.Join(args, " "), text)
		}
		return fmt.Sprintf("dsproxy %s timed out.", strings.Join(args, " "))
	}

	if err != nil {
		if text != "" {
			return fmt.Sprintf("dsproxy %s failed: %v\n%s", strings.Join(args, " "), err, text)
		}
		return fmt.Sprintf("dsproxy %s failed: %v", strings.Join(args, " "), err)
	}

	if text == "" {
		return fmt.Sprintf("dsproxy %s completed.", strings.Join(args, " "))
	}
	return text
}

func isDeepSeekRuntimeAgent(name string) bool {
	return name == "deepseek" || name == "deepseek-thinking"
}

type runtimeModelSetter interface {
	SetModel(model string)
}

func (h *Handler) setRunningAgentModel(name, model string) bool {
	h.mu.RLock()
	ag := h.agents[name]
	h.mu.RUnlock()

	setter, ok := ag.(runtimeModelSetter)
	if !ok {
		return false
	}
	setter.SetModel(model)
	return true
}

func persistDeepSeekRuntimeModel(name, model string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	agCfg, ok := cfg.Agents[name]
	if !ok {
		return fmt.Errorf("agent %q is not configured", name)
	}

	agCfg.Model = model
	if agCfg.ModelProvider == "" {
		switch name {
		case "deepseek":
			agCfg.ModelProvider = "deepseek-proxy"
		case "deepseek-thinking":
			agCfg.ModelProvider = "deepseek-thinking-proxy"
		}
	}
	cfg.Agents[name] = agCfg
	return config.Save(cfg)
}

func commandCard(title string, lines ...string) string {
	title = strings.TrimSpace(title)
	parts := make([]string, 0, len(lines)+4)
	if title != "" {
		if strings.HasPrefix(title, "#") {
			parts = append(parts, title)
		} else {
			parts = append(parts, "## "+title)
		}
	}

	body := markdownCommandLines(lines...)
	if len(body) > 0 {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, body...)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func markdownCommandLines(lines ...string) []string {
	out := make([]string, 0, len(lines))
	lastBlank := false
	openFence := markdownFenceSpec{}

	for _, rawLine := range lines {
		lineForFence := strings.TrimRight(rawLine, " \t")
		trimmed := strings.TrimSpace(lineForFence)

		if trimmed == "" {
			if !lastBlank {
				out = append(out, "")
				lastBlank = true
			}
			continue
		}

		if spec, ok := markdownFenceLineSpec(trimmed); ok {
			out = append(out, trimmed)
			if openFence.valid {
				if markdownFenceCanClose(spec, openFence) {
					openFence = markdownFenceSpec{}
				}
			} else {
				openFence = spec
			}
			lastBlank = false
			continue
		}

		if openFence.valid {
			out = append(out, lineForFence)
			lastBlank = false
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "#"),
			strings.HasPrefix(trimmed, "|"),
			strings.HasPrefix(trimmed, "---"),
			strings.HasPrefix(trimmed, ">"),
			strings.HasPrefix(trimmed, "- "):
			out = append(out, trimmed)
		case strings.HasPrefix(trimmed, "• "):
			out = append(out, "- "+strings.TrimSpace(strings.TrimPrefix(trimmed, "• ")))
		case commandSectionLine(trimmed):
			out = append(out, "### "+trimmed)
		default:
			out = append(out, trimmed)
		}
		lastBlank = false
	}

	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func commandSectionLine(line string) bool {
	if line == "" || strings.Contains(line, ":") || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "`") {
		return false
	}
	for _, r := range line {
		return r > 127
	}
	return false
}

func visualCommandFence(title string, lines ...string) []string {
	body := make([]string, 0, len(lines)+1)
	if strings.TrimSpace(title) != "" {
		body = append(body, title)
	}
	body = append(body, lines...)

	fence := markdownFenceForLines(body...)
	out := []string{fence + "text"}
	out = append(out, body...)
	out = append(out, fence)
	return out
}

func markdownTableCell(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	value = strings.ReplaceAll(value, "|", `\|`)
	if value == "" {
		return "--"
	}
	return value
}

func slashInlineCode(value string) string {
	return markdownInlineCode(valueOrUnknown(strings.TrimSpace(value)))
}

func slashBoldField(label, value string) string {
	return "- **" + label + ":** " + value
}

func slashAgentTypeBadge(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "acp":
		return "ACP"
	case "cli":
		return "CLI"
	case "http":
		return "HTTP"
	case "", "none", "unknown":
		return "unknown"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func slashEffortDisplay(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "xhigh":
		return "max"
	case "high", "max", "medium", "low":
		return strings.ToLower(strings.TrimSpace(value))
	case "":
		return "unknown"
	default:
		return strings.TrimSpace(value)
	}
}

func commandStatusNeedsAttention(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" || status == "ok" || status == "updated" || status == "unchanged" || status == "skipped" {
		return false
	}
	return strings.Contains(status, "fail") || strings.Contains(status, "warning") || strings.Contains(status, "error") || strings.Contains(status, "timeout")
}

func slashModelDisplay(agentModel, proxyModel string) string {
	agentModel = strings.TrimSpace(agentModel)
	if agentModel != "" && agentModel != "none" && agentModel != "unknown" {
		return agentModel
	}
	proxyModel = strings.TrimSpace(proxyModel)
	if proxyModel != "" {
		return proxyModel
	}
	return "unknown"
}

func buildCompactStatusPanel(ag agent.Agent, userID string, fallbackWindow int64, proxyRoute, proxyEndpoint, proxyState, balanceSummary string) []string {
	var snapshot agent.TokenUsageSnapshot
	hasSnapshot := false
	if ag != nil {
		if inspector, ok := ag.(agent.TokenUsageInspector); ok {
			if current, ok := inspector.CurrentTokenUsage(userID); ok {
				snapshot = current
				hasSnapshot = true
			}
		}
	}

	window := fallbackWindow
	if window <= 0 && hasSnapshot && snapshot.ModelContextWindow > 0 {
		window = snapshot.ModelContextWindow
	}

	used := int64(0)
	if hasSnapshot && snapshot.Total.TotalTokens > 0 {
		used = snapshot.Total.TotalTokens
	}

	contextLine := fmt.Sprintf(
		"Context  [%s]  %s  %s/%s",
		formatCommandProgressBar(used, window, 20),
		formatTokenPercent(used, window),
		formatTokenCount(maxInt64(used, 0)),
		formatContextLimit(window),
	)

	tokenLine := "Tokens   waiting for Codex usage event"
	if hasSnapshot {
		tokenLine = fmt.Sprintf(
			"Tokens   in %s  cached %s  out %s  reason %s",
			formatTokenCount(snapshot.Total.InputTokens),
			formatTokenCount(snapshot.Total.CachedInputTokens),
			formatTokenCount(snapshot.Total.OutputTokens),
			formatTokenCount(snapshot.Total.ReasoningOutputTokens),
		)
		if snapshot.Last.TotalTokens > 0 {
			tokenLine += fmt.Sprintf(
				"  last %s",
				formatTokenCount(snapshot.Last.TotalTokens),
			)
		}
	}

	balanceSummary = strings.TrimSpace(balanceSummary)
	if balanceSummary == "" {
		balanceSummary = "balance n/a"
	}
	costLine := fmt.Sprintf("%s  %s", "Cost     session~n/a  last~n/a", balanceSummary)

	lines := []string{
		contextLine,
		tokenLine,
		costLine,
	}

	if strings.TrimSpace(proxyRoute) != "" || strings.TrimSpace(proxyEndpoint) != "" || strings.TrimSpace(proxyState) != "" {
		lines = append(lines,
			fmt.Sprintf("Proxy    %s · %s · %s", valueOrUnknown(proxyRoute), valueOrUnknown(proxyEndpoint), valueOrUnknown(proxyState)),
			"Paths    cfg ~/.weclaw/config.json · log ~/.weclaw/weclaw.log",
		)
	}

	return lines
}

func balanceFieldHasData(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "unknown", "--", "null", "none":
		return false
	default:
		return true
	}
}

func boolText(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func compactBalanceSummaryFromText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "balance n/a"
	}

	var payload dsproxyBalanceResponse
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "balance n/a"
	}
	for _, info := range payload.Balance.BalanceInfos {
		total := strings.TrimSpace(info.TotalBalance)
		if balanceFieldHasData(total) {
			currency := strings.TrimSpace(info.Currency)
			if currency == "" {
				currency = "unknown"
			}
			switch strings.ToUpper(currency) {
			case "CNY", "RMB", "CNH":
				return "￥" + total
			case "USD":
				return "$" + total
			default:
				return currency + " " + total
			}
		}
	}
	if len(payload.Balance.BalanceInfos) == 0 {
		return "balance none"
	}
	return "balance n/a"
}

func balancePanelRows(payload dsproxyBalanceResponse) []string {
	if len(payload.Balance.BalanceInfos) == 0 {
		return []string{"Balance  none"}
	}

	currencyWidth := len("Currency")
	for _, info := range payload.Balance.BalanceInfos {
		currency := strings.TrimSpace(info.Currency)
		if currency == "" {
			currency = "unknown"
		}
		if len(currency) > currencyWidth {
			currencyWidth = len(currency)
		}
	}

	rows := []string{fmt.Sprintf("%-*s  %s", currencyWidth, "Currency", "Total")}
	for _, info := range payload.Balance.BalanceInfos {
		currency := strings.TrimSpace(info.Currency)
		if currency == "" {
			currency = "unknown"
		}
		rows = append(rows, fmt.Sprintf("%-*s  %s", currencyWidth, currency, valueOrUnknown(info.TotalBalance)))
	}
	return rows
}

func dsproxyUptimeFromStatus(text string) string {
	var payload struct {
		UptimeSeconds float64 `json:"uptime_seconds"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &payload); err != nil {
		return "unknown"
	}
	if payload.UptimeSeconds <= 0 {
		return "unknown"
	}
	return formatTurnDuration(time.Duration(payload.UptimeSeconds) * time.Second)
}

func runtimeVersionLines(text, publicPrefix, internalPrefix string) (string, string) {
	publicLine := "unknown"
	internalLine := "unknown"
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, strings.ToLower(publicPrefix)):
			publicLine = strings.TrimSpace(trimmed[len(publicPrefix):])
		case strings.HasPrefix(lower, strings.ToLower(internalPrefix)):
			internalLine = strings.TrimSpace(trimmed[len(internalPrefix):])
		}
	}
	return publicLine, internalLine
}

func runCurrentExecutableCommand(ctx context.Context, args ...string) string {
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		exe = "weclaw"
	}
	cmd := exec.CommandContext(runCtx, exe, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if runCtx.Err() == context.DeadlineExceeded {
		if text != "" {
			return fmt.Sprintf("weclaw %s timed out.\n%s", strings.Join(args, " "), text)
		}
		return fmt.Sprintf("weclaw %s timed out.", strings.Join(args, " "))
	}
	if err != nil {
		if text != "" {
			return fmt.Sprintf("weclaw %s failed: %v\n%s", strings.Join(args, " "), err, text)
		}
		return fmt.Sprintf("weclaw %s failed: %v", strings.Join(args, " "), err)
	}
	if text == "" {
		return fmt.Sprintf("weclaw %s completed.", strings.Join(args, " "))
	}
	return text
}

func currentProcessUptime() string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(os.Getpid()), "-o", "etimes=").Output()
	if err != nil {
		return "unknown"
	}
	secondsText := strings.TrimSpace(string(out))
	seconds, err := strconv.ParseInt(secondsText, 10, 64)
	if err != nil || seconds < 0 {
		if secondsText == "" {
			return "unknown"
		}
		return secondsText + "s"
	}
	return formatTurnDuration(time.Duration(seconds) * time.Second)
}

func formatCommandProgressBar(current, total int64, width int) string {
	return formatCommandProgressBarOriginal(current, total, width)
}

func formatCommandProgressBarOriginal(current, total int64, width int) string {
	if width <= 0 {
		return ""
	}
	filled := commandProgressFilledCells(current, total, width)
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func formatCommandProgressBarRightEnd(current, total int64, width int) string {
	if width <= 0 {
		return ""
	}
	filled := commandProgressFilledCells(current, total, width)
	if filled <= 0 {
		return strings.Repeat("─", width)
	}
	if filled >= width {
		return strings.Repeat("━", width)
	}
	return strings.Repeat("━", filled) + "╸" + strings.Repeat("─", width-filled-1)
}

func commandProgressFilledCells(current, total int64, width int) int {
	if width <= 0 || total <= 0 || current <= 0 {
		return 0
	}
	if current >= total {
		return width
	}
	filled := int((current*int64(width) + total/2) / total)
	if filled < 1 {
		return 1
	}
	if filled > width {
		return width
	}
	return filled
}

func compactCommandOutput(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

func commandStatusFromOutput(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "ok"
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "failed") || strings.Contains(lower, "timed out") || strings.Contains(lower, "error") {
		return compactCommandOutput(trimmed, 220)
	}
	if payload, ok := parseJSONMap(trimmed); ok {
		status := nestedStringDefault(payload, "ok", "status")
		if strings.EqualFold(status, "ok") || strings.EqualFold(status, "updated") || strings.EqualFold(status, "unchanged") {
			return "ok"
		}
		return compactCommandOutput(status, 220)
	}
	return "ok"
}

type dsproxyBalanceResponse struct {
	Status  string `json:"status"`
	Balance struct {
		IsAvailable  bool `json:"is_available"`
		BalanceInfos []struct {
			Currency        string `json:"currency"`
			TotalBalance    string `json:"total_balance"`
			GrantedBalance  string `json:"granted_balance"`
			ToppedUpBalance string `json:"topped_up_balance"`
		} `json:"balance_infos"`
	} `json:"balance"`
}

func formatBalanceReply(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return commandCard(
			"💰 Balance",
			"> dsproxy returned an empty balance response.",
		)
	}

	var payload dsproxyBalanceResponse
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return commandCard(
			"💰 Balance",
			slashBoldField("Status", slashInlineCode(commandStatusFromOutput(trimmed))),
			"",
			"```text",
			compactCommandOutput(trimmed, 360),
			"```",
		)
	}

	status := payload.Status
	if status == "" {
		status = "ok"
	}

	lines := []string{
		slashBoldField("Status", slashInlineCode(status)),
		slashBoldField("Available", slashInlineCode(boolText(payload.Balance.IsAvailable))),
		"",
		"```text",
	}
	lines = append(lines, balancePanelRows(payload)...)
	lines = append(lines, "```")

	return commandCard("💰 Balance", lines...)
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func extractCommandValue(text, key string) string {
	for _, marker := range []string{
		"\"" + key + "\"",
		key,
	} {
		idx := strings.Index(text, marker)
		if idx < 0 {
			continue
		}
		rest := text[idx+len(marker):]
		if colon := strings.Index(rest, ":"); colon >= 0 {
			rest = rest[colon+1:]
		} else if eq := strings.Index(rest, "="); eq >= 0 {
			rest = rest[eq+1:]
		}
		rest = strings.TrimSpace(rest)
		rest = strings.Trim(rest, "\", ")
		if comma := strings.IndexAny(rest, ",\n\r"); comma >= 0 {
			rest = rest[:comma]
		}
		rest = strings.TrimSpace(strings.Trim(rest, "\", "))
		if rest != "" && rest != "{" && rest != "}" {
			return rest
		}
	}
	return ""
}

type stoppableAgent interface {
	Stop()
}

func stopAgentIfSupported(name string, ag agent.Agent) {
	if ag == nil {
		return
	}
	stopper, ok := ag.(stoppableAgent)
	if !ok {
		return
	}
	log.Printf("[handler] stopping agent %q before restart", name)
	stopper.Stop()
}

func profileThinkingLine(name string) string {
	if name == "deepseek-thinking" {
		return "• thinking: enabled"
	}
	return "• thinking: disabled"
}

func (h *Handler) restartProfileAgent(ctx context.Context, name, userID string) string {
	if !h.isKnownAgent(name) {
		return commandCard(
			"⚠️ Profile unavailable",
			slashBoldField("Profile", slashInlineCode(name)),
			"",
			"```text",
			"State    not configured",
			"Action   run weclaw start "+name+" once, or check agent detection",
			"```",
		)
	}

	reply := h.switchDefault(ctx, name)

	h.mu.RLock()
	ready := h.defaultName == name && h.agents[name] != nil
	h.mu.RUnlock()
	if !ready {
		return reply
	}

	return commandCard(
		"🔁 Profile updated",
		slashBoldField("Profile", slashInlineCode(name)),
		slashBoldField("Session", "preserved when available"),
		slashBoldField("Thinking", strings.TrimPrefix(profileThinkingLine(name), "• thinking: ")),
	)
}

func (h *Handler) restartCurrentDefaultAgent(ctx context.Context, userID string) string {
	h.mu.RLock()
	name := h.defaultName
	h.mu.RUnlock()

	if name == "" {
		return commandCard(
			"⚠️ Restart",
			"> No default profile is configured.",
		)
	}

	if value, ok := h.runningTurns.Load(userID); ok {
		if state, ok := value.(*runningTurnState); ok {
			state.requestCancel()
		}
	}

	sessionReply := h.resetDefaultSession(ctx, userID)
	return sessionReply + "\n\n" + commandCard(
		"🔄 Restart",
		slashBoldField("Profile", slashInlineCode(name)),
		"",
		"```text",
		"Action   new session created",
		"Config   preserved",
		"```",
	)
}

// sendToDefaultAgent sends the message to the default agent and replies.
func (h *Handler) sendToDefaultAgent(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, text, clientID string) {
	stopTyping := h.startTypingKeepalive(ctx, client, msg.FromUserID, msg.ContextToken)
	defer stopTyping()

	unlock := h.lockUserTurn(msg.FromUserID)
	defer unlock()

	h.mu.RLock()
	defaultName := h.defaultName
	h.mu.RUnlock()

	ag := h.getDefaultAgent()
	var reply string
	var streamedText bool
	if ag != nil {
		h.applyPendingResume(ctx, defaultName, ag, msg.FromUserID)
		turnCtx, turnState, finishTurn := h.beginRunningTurn(ctx, msg.FromUserID, defaultName, text)
		defer finishTurn()

		var err error
		reply, streamedText, err = h.chatWithAgentTracking(turnCtx, ag, msg.FromUserID, text, turnState)
		if err != nil {
			if turnState.wasCancelRequested() && turnCtx.Err() != nil {
				log.Printf("[handler] cancelled default agent turn for %s", msg.FromUserID)
				return
			}
			reply = fmt.Sprintf("Error: %v", err)
			streamedText = false
		}
	} else {
		log.Printf("[handler] agent not ready, using echo mode for %s", msg.FromUserID)
		reply = "[echo] " + text
	}

	h.sendReplyWithMediaOptions(ctx, client, msg, defaultName, reply, clientID, !streamedText, replyTextSingle)
}

// sendToNamedAgent sends the message to a specific agent and replies.
func (h *Handler) sendToNamedAgent(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, name, message, clientID string) {
	stopTyping := h.startTypingKeepalive(ctx, client, msg.FromUserID, msg.ContextToken)
	defer stopTyping()

	unlock := h.lockUserTurn(msg.FromUserID)
	defer unlock()

	ag, agErr := h.getAgent(ctx, name)
	if agErr != nil {
		log.Printf("[handler] agent %q not available: %v", name, agErr)
		reply := fmt.Sprintf("Agent %q is not available: %v", name, agErr)
		SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID)
		return
	}

	h.applyPendingResume(ctx, name, ag, msg.FromUserID)
	turnCtx, turnState, finishTurn := h.beginRunningTurn(ctx, msg.FromUserID, name, message)
	defer finishTurn()

	reply, streamedText, err := h.chatWithAgentTracking(turnCtx, ag, msg.FromUserID, message, turnState)
	if err != nil {
		if turnState.wasCancelRequested() && turnCtx.Err() != nil {
			log.Printf("[handler] cancelled named agent turn for %s", msg.FromUserID)
			return
		}
		reply = fmt.Sprintf("Error: %v", err)
		streamedText = false
	}
	h.sendReplyWithMediaOptions(ctx, client, msg, name, reply, clientID, !streamedText, replyTextSingle)
}

// broadcastToAgents sends the message to multiple agents in parallel.
// Each reply is sent as a separate message with the agent name prefix.
func (h *Handler) broadcastToAgents(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, names []string, message string) {
	stopTyping := h.startTypingKeepalive(ctx, client, msg.FromUserID, msg.ContextToken)
	defer stopTyping()

	unlock := h.lockUserTurn(msg.FromUserID)
	defer unlock()

	type result struct {
		name  string
		reply string
	}

	ch := make(chan result, len(names))

	for _, name := range names {
		go func(n string) {
			ag, err := h.getAgent(ctx, n)
			if err != nil {
				ch <- result{name: n, reply: fmt.Sprintf("Error: %v", err)}
				return
			}
			reply, _, err := h.chatWithAgent(ctx, nil, msg, ag, msg.FromUserID, message)
			if err != nil {
				ch <- result{name: n, reply: fmt.Sprintf("Error: %v", err)}
				return
			}
			ch <- result{name: n, reply: reply}
		}(name)
	}

	// Send replies as they arrive
	for range names {
		r := <-ch
		reply := fmt.Sprintf("[%s] %s", r.name, r.reply)
		clientID := NewClientID()
		h.sendReplyWithMedia(ctx, client, msg, r.name, reply, clientID)
	}
}

// sendReplyWithMedia sends a text reply and any extracted image URLs.
func (h *Handler) sendReplyWithMedia(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, agentName, reply, clientID string) {
	h.sendReplyWithMediaOptions(ctx, client, msg, agentName, reply, clientID, true, replyTextChunked)
}

func (h *Handler) sendReplyWithMediaOptions(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, agentName, reply, clientID string, sendText bool, textMode replyTextMode) bool {
	imageURLs := ExtractImageURLs(reply)
	attachmentPaths := extractLocalAttachmentPaths(reply)
	allowedRoots := h.allowedAttachmentRoots(agentName)
	contextToken := h.latestContextToken(msg.FromUserID, msg.ContextToken)
	textDelivered := !sendText

	var sentPaths []string
	var failedPaths []string
	for _, attachmentPath := range attachmentPaths {
		if !isAllowedAttachmentPath(attachmentPath, allowedRoots) {
			log.Printf("[handler] rejected attachment outside allowed roots for agent %q: %s", agentName, attachmentPath)
			failedPaths = append(failedPaths, attachmentPath)
			continue
		}
		if err := SendMediaFromPath(ctx, client, msg.FromUserID, attachmentPath, contextToken); err != nil {
			log.Printf("[handler] failed to send attachment to %s: %v", msg.FromUserID, err)
			failedPaths = append(failedPaths, attachmentPath)
			continue
		}
		sentPaths = append(sentPaths, attachmentPath)
	}

	reply = rewriteReplyWithAttachmentResults(reply, sentPaths, failedPaths)
	if sendText {
		if textMode == replyTextSingle {
			textDelivered = true
			if err := SendTextReply(ctx, client, msg.FromUserID, reply, contextToken, clientID); err != nil {
				log.Printf("[handler] failed to send final reply to %s: %v", msg.FromUserID, err)
				textDelivered = false
			}
		} else {
			chunks := finalReplyTextChunks(reply)
			textDelivered = true
			for i, chunk := range chunks {
				chunkClientID := clientID
				if i > 0 {
					chunkClientID = NewClientID()
				}
				if err := SendTextReply(ctx, client, msg.FromUserID, chunk, contextToken, chunkClientID); err != nil {
					log.Printf("[handler] failed to send reply chunk %d to %s: %v", i+1, msg.FromUserID, err)
					textDelivered = false
					break
				}
			}
		}
	}

	for _, imgURL := range imageURLs {
		if err := SendMediaFromURL(ctx, client, msg.FromUserID, imgURL, contextToken); err != nil {
			log.Printf("[handler] failed to send image to %s: %v", msg.FromUserID, err)
		}
	}
	return textDelivered
}

func finalReplyTextChunks(reply string) []string {
	return ClawBotMarkdownReplyChunks(reply)
}

func (h *Handler) latestContextToken(userID, fallback string) string {
	if value, ok := h.contextTokens.Load(userID); ok {
		if token, ok := value.(string); ok && token != "" {
			return token
		}
	}
	return fallback
}

func (h *Handler) startTypingKeepalive(ctx context.Context, client *ilink.Client, userID, contextToken string) func() {
	if client == nil || userID == "" {
		return func() {}
	}

	h.mu.RLock()
	sendTyping := h.typingSender
	interval := h.typingEvery
	h.mu.RUnlock()

	if sendTyping == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = 6 * time.Second
	}

	typeCtx, cancel := context.WithCancel(ctx)
	if err := sendTyping(typeCtx, client, userID, contextToken); err != nil {
		log.Printf("[handler] failed to send typing state: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-typeCtx.Done():
				return
			case <-ticker.C:
				if err := sendTyping(typeCtx, client, userID, contextToken); err != nil {
					log.Printf("[handler] failed to refresh typing state: %v", err)
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

func (h *Handler) allowedAttachmentRoots(agentName string) []string {
	roots := []string{defaultAttachmentWorkspace()}

	h.mu.RLock()
	agentDir := h.agentWorkDirs[agentName]
	h.mu.RUnlock()

	if agentDir != "" {
		roots = append(roots, agentDir)
	}

	return roots
}

// chatWithAgent sends a message to an agent and returns the reply, with logging.
func (h *Handler) chatWithAgent(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, ag agent.Agent, userID, message string) (string, bool, error) {
	info := ag.Info()
	log.Printf("[handler] dispatching to agent (%s) for %s", info, userID)

	start := time.Now()
	reply, streamedText, err := h.chatMaybeStream(ctx, client, msg, ag, userID, message)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[handler] agent error (%s, elapsed=%s): %v", info, elapsed, err)
		return "", false, err
	}

	log.Printf("[handler] agent replied (%s, elapsed=%s): %q", info, elapsed, truncate(reply, 100))
	return reply, streamedText, nil
}

func (h *Handler) chatWithAgentWithoutStreaming(ctx context.Context, ag agent.Agent, userID, message string) (string, bool, error) {
	return h.chatWithAgentTracking(ctx, ag, userID, message, nil)
}

func (h *Handler) chatWithAgentTracking(ctx context.Context, ag agent.Agent, userID, message string, turnState *runningTurnState) (string, bool, error) {
	info := ag.Info()
	log.Printf("[handler] dispatching to agent without WeChat streaming (%s) for %s", info, userID)

	start := time.Now()
	if turnState != nil {
		turnState.update("running", "dispatching to "+userFacingAgentLabel("", info))
		turnState.updateSessionID(currentAgentSessionID(ag, userID))
		h.trackAgentSessionIDDuringTurn(ctx, ag, userID, turnState)
	}

	var reply string
	var err error
	if streamingAg, ok := ag.(agent.StreamingAgent); ok && turnState != nil {
		reply, err = streamingAg.ChatStream(ctx, userID, message, func(evt agent.ProgressEvent) error {
			turnState.observeProgress(evt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		})
	} else {
		reply, err = ag.Chat(ctx, userID, message)
	}
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[handler] agent error (%s, elapsed=%s): %v", info, elapsed, err)
		return "", false, err
	}

	if turnState != nil {
		turnState.updateSessionID(currentAgentSessionID(ag, userID))
		turnState.update("done", "final answer ready")
	}
	log.Printf("[handler] agent replied (%s, elapsed=%s): %q", info, elapsed, truncate(reply, 100))
	return reply, false, nil
}

func (h *Handler) chatMaybeStream(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, ag agent.Agent, userID, message string) (string, bool, error) {
	h.mu.RLock()
	cfg := h.streamConfig
	sender := h.streamSender
	pace := h.streamPace
	h.mu.RUnlock()

	streamingAg, ok := ag.(agent.StreamingAgent)
	if !cfg.Enabled || !ok || client == nil || sender == nil {
		reply, err := ag.Chat(ctx, userID, message)
		return reply, false, err
	}

	rawSendStreamText := func(text string) error {
		return sender(ctx, client, msg.FromUserID, text, h.latestContextToken(msg.FromUserID, msg.ContextToken), NewClientID())
	}
	streamSender := newOrderedStreamSenderWithPace(rawSendStreamText, pace)
	progress := newTurnProgressController(streamSender.Send)
	defer progress.Close()

	streamState := struct {
		answerTextSeen      bool
		answerDeliveryError bool
		completionSent      bool
		sentBlocks          map[string]struct{}
	}{sentBlocks: make(map[string]struct{})}
	streamedText := func() bool {
		return streamState.answerTextSeen && !streamState.answerDeliveryError && streamState.completionSent
	}
	sendAssistantComplete := func(evt agent.ProgressEvent) {
		if streamState.answerDeliveryError {
			return
		}
		key := evt.ID
		if key == "" {
			key = strings.TrimSpace(normalizeLineEndings(evt.Text))
		}
		if key != "" {
			if _, ok := streamState.sentBlocks[key]; ok {
				return
			}
		}
		chunks := ClawBotMarkdownReplyChunks(evt.Text)
		if len(chunks) == 0 {
			return
		}
		for _, chunk := range chunks {
			if err := streamSender.Send(chunk); err != nil {
				log.Printf("[handler] failed to send assistant message block: %v", err)
				streamState.answerDeliveryError = true
				return
			}
		}
		streamState.answerTextSeen = true
		if key != "" {
			streamState.sentBlocks[key] = struct{}{}
		}
	}
	handleProgressEvent := func(evt agent.ProgressEvent) {
		if evt.Type == agent.ProgressEventAssistantMessageComplete {
			sendAssistantComplete(evt)
			return
		}
		if evt.Type == agent.ProgressEventAssistantDelta {
			return
		}
		if cfg.ToolEvents {
			progress.Handle(evt)
		}
	}

	type streamResult struct {
		reply string
		err   error
	}
	eventCh := make(chan agent.ProgressEvent, 64)
	doneCh := make(chan streamResult, 1)
	go func() {
		reply, err := streamingAg.ChatStream(ctx, userID, message, func(evt agent.ProgressEvent) error {
			select {
			case eventCh <- evt:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		doneCh <- streamResult{reply: reply, err: err}
	}()

	for {
		select {
		case evt := <-eventCh:
			handleProgressEvent(evt)
		case result := <-doneCh:
			for {
				select {
				case evt := <-eventCh:
					handleProgressEvent(evt)
				default:
					if result.err != nil {
						return "", streamedText(), result.err
					}
					progress.Close()
					if streamState.answerTextSeen && !streamState.answerDeliveryError {
						if err := streamSender.SendCompletion("Done."); err != nil {
							log.Printf("[handler] failed to send assistant stream completion: %v", err)
							streamState.answerDeliveryError = true
						} else {
							streamState.completionSent = true
						}
					}
					return result.reply, streamedText(), nil
				}
			}
		case <-ctx.Done():
			return "", streamedText(), ctx.Err()
		}
	}
}

const (
	streamSendPaceDelay        = 1200 * time.Millisecond
	streamCompletionAttempts   = 3
	streamCompletionRetryDelay = 80 * time.Millisecond
)

type orderedStreamSender struct {
	mu   sync.Mutex
	send func(string) error
	pace time.Duration
	last time.Time
}

func newOrderedStreamSender(send func(string) error) *orderedStreamSender {
	return newOrderedStreamSenderWithPace(send, streamSendPaceDelay)
}

func newOrderedStreamSenderWithPace(send func(string) error, pace time.Duration) *orderedStreamSender {
	if pace < 0 {
		pace = 0
	}
	return &orderedStreamSender{
		send: send,
		pace: pace,
	}
}

func (s *orderedStreamSender) Send(text string) error {
	if s == nil || s.send == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.last.IsZero() {
		if wait := s.pace - time.Since(s.last); wait > 0 {
			time.Sleep(wait)
		}
	}
	err := s.send(text)
	s.last = time.Now()
	return err
}

func (s *orderedStreamSender) SendCompletion(text string) error {
	var err error
	for attempt := 0; attempt < streamCompletionAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(streamCompletionRetryDelay)
		}
		err = s.Send(text)
		if err == nil {
			return nil
		}
	}
	return err
}

func (h *Handler) lockUserTurn(userID string) func() {
	if userID == "" {
		userID = "default"
	}
	value, _ := h.userTurns.LoadOrStore(userID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func compactStreamEvent(evt agent.ProgressEvent) (string, bool) {
	switch evt.Type {
	case agent.ProgressEventToolStart, agent.ProgressEventToolProgress:
		name := compactToolName(evt.Text)
		if !isUsefulToolName(name) {
			return "", false
		}
		return "using " + name, true
	default:
		return "", false
	}
}

func compactToolName(text string) string {
	text = strings.TrimSpace(text)
	for _, prefix := range []string{
		"tool started:",
		"tool progress:",
		"tool completed:",
		"using",
	} {
		if rest, ok := strings.CutPrefix(text, prefix); ok {
			text = strings.TrimSpace(rest)
			break
		}
	}
	if before, _, ok := strings.Cut(text, ":"); ok {
		text = strings.TrimSpace(before)
	}
	if before, _, ok := strings.Cut(text, " ("); ok {
		text = strings.TrimSpace(before)
	}
	return text
}

func isUsefulToolName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	switch lower {
	case "mcptoolcall", "tool", "tools", "status", "started", "completed", "running":
		return false
	}
	if strings.HasPrefix(lower, "call_") || strings.HasPrefix(lower, "toolu_") {
		return false
	}
	if strings.ContainsAny(name, " \t\r\n") {
		return false
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

const (
	maxProgressSendsPerTurn     = 3
	maxProgressToolSendsPerTurn = 2
)

type turnProgressController struct {
	mu        sync.Mutex
	send      func(string) error
	disabled  bool
	sent      int
	toolSent  int
	seenTools map[string]struct{}
}

func newTurnProgressController(send func(string) error) *turnProgressController {
	p := &turnProgressController{
		send:      send,
		seenTools: make(map[string]struct{}),
	}
	return p
}

func (p *turnProgressController) Handle(evt agent.ProgressEvent) {
	text, ok := compactStreamEvent(evt)
	if !ok {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.disabled {
		return
	}
	if _, exists := p.seenTools[text]; exists {
		return
	}
	if p.sent >= maxProgressSendsPerTurn || p.toolSent >= maxProgressToolSendsPerTurn {
		return
	}
	p.seenTools[text] = struct{}{}
	if p.sendLocked(text) {
		p.toolSent++
	}
}

func (p *turnProgressController) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.disabled = true
}

func (p *turnProgressController) sendLocked(text string) bool {
	if p.send == nil {
		return false
	}
	if err := p.send(text); err != nil {
		log.Printf("[handler] failed to send stream progress: %v", err)
		p.disabled = true
		return false
	}
	p.sent++
	return true
}

// switchDefault switches the default agent. Starts it on demand if needed.
// The change is persisted to config file.
func (h *Handler) switchDefault(ctx context.Context, name string) string {
	ag, err := h.getAgent(ctx, name)
	if err != nil {
		log.Printf("[handler] failed to switch default to %q: %v", name, err)
		return commandCard(
			"⚠️ Profile switch failed",
			slashBoldField("Profile", slashInlineCode(name)),
			"",
			"```text",
			"Error    "+compactCommandOutput(err.Error(), 160),
			"```",
		)
	}

	h.mu.Lock()
	old := h.defaultName
	h.defaultName = name
	h.agents[name] = ag
	h.mu.Unlock()

	if h.saveDefault != nil {
		if err := h.saveDefault(name); err != nil {
			log.Printf("[handler] failed to save default agent to config: %v", err)
		} else {
			log.Printf("[handler] saved default agent %q to config", name)
		}
	}

	info := ag.Info()
	log.Printf("[handler] switched default agent: %s -> %s (%s)", old, name, info)
	return commandCard(
		"🔁 Profile updated",
		slashBoldField("Profile", slashInlineCode(userFacingAgentLabel(name, info))),
		"",
		"```text",
		"Previous "+valueOrUnknown(old),
		"Session  preserved when available",
		"Next     send a message or use /status",
		"```",
	)
}

// resetDefaultSession resets the session for the given userID on the default agent.
func (h *Handler) resetDefaultSession(ctx context.Context, userID string) string {
	h.mu.RLock()
	defaultName := h.defaultName
	h.mu.RUnlock()

	ag := h.getDefaultAgent()
	if ag == nil {
		return commandCard(
			"⚠️ Session",
			"> No agent is running.",
		)
	}
	info := ag.Info()
	name := userFacingAgentLabel(defaultName, info)
	sessionID, err := ag.ResetSession(ctx, userID)
	if err != nil {
		log.Printf("[handler] reset session failed for %s: %v", userID, err)
		return commandCard(
			"⚠️ Session",
			"> Failed to create a new session.",
			"",
			"```text",
			"Error    "+compactCommandOutput(err.Error(), 160),
			"```",
		)
	}
	if sessionID != "" {
		h.clearPendingResumeForProfile(defaultName)
		if current := currentAgentSessionID(ag, userID); current != sessionID {
			if resumer, ok := ag.(agent.SessionResumer); ok {
				if err := resumer.ResumeSession(userID, sessionID); err != nil {
					log.Printf("[handler] failed to bind new session %s for %s: %v", sessionID, userID, err)
				}
			}
		}
		h.recordRuntimeSession(defaultName, userID, sessionID)
	}
	lines := []string{
		slashBoldField("Profile", slashInlineCode(name)),
		"",
		"```text",
		"Action   new session created",
	}
	if sessionID != "" {
		lines = append(lines, "Session  "+sessionID)
	}
	lines = append(lines, "```")
	return commandCard("🧵 Session", lines...)
}

func userFacingAgentLabel(preferred string, info agent.AgentInfo) string {
	if label := friendlyAgentName(preferred); label != "" {
		return label
	}
	if label := friendlyAgentName(info.Name); label != "" {
		return label
	}
	if label := friendlyAgentName(info.Command); label != "" {
		return label
	}
	return "agent"
}

func friendlyAgentName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.TrimRight(name, `/\`)
	if strings.ContainsAny(name, `/\`) || filepath.IsAbs(name) {
		name = filepath.Base(strings.ReplaceAll(name, `\`, `/`))
	}
	return name
}

// handleCwd handles the /cwd command. It updates the working directory for all running agents.
func (h *Handler) handleCwd(trimmed string) string {
	arg := strings.TrimSpace(strings.TrimPrefix(trimmed, "/cwd"))
	if arg == "" {
		ag := h.getDefaultAgent()
		if ag == nil {
			return commandCard("📁 Workspace", "- status: No agent running.")
		}
		info := ag.Info()
		return commandCard(
			"📁 Workspace",
			slashBoldField("Agent", slashInlineCode(info.Name)),
			slashBoldField("Cwd", "check agent config"),
		)
	}

	if arg == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			arg = home
		}
	} else if strings.HasPrefix(arg, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			arg = filepath.Join(home, arg[2:])
		}
	}

	absPath, err := filepath.Abs(arg)
	if err != nil {
		return commandCard("⚠️ Workspace", fmt.Sprintf("- error: Invalid path: %v", err))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return commandCard("⚠️ Workspace", "- error: Path not found", "- path: "+absPath)
	}
	if !info.IsDir() {
		return commandCard("⚠️ Workspace", "- error: Not a directory", "- path: "+absPath)
	}

	h.mu.RLock()
	agents := make(map[string]agent.Agent, len(h.agents))
	for name, ag := range h.agents {
		agents[name] = ag
	}
	h.mu.RUnlock()

	for name, ag := range agents {
		ag.SetCwd(absPath)
		log.Printf("[handler] updated cwd for agent %s: %s", name, absPath)
	}

	h.mu.Lock()
	for name := range agents {
		h.agentWorkDirs[name] = absPath
	}
	h.mu.Unlock()

	return commandCard(
		"📁 Workspace",
		slashBoldField("Cwd", slashInlineCode(absPath)),
	)
}

// buildStatus returns a short status string showing the current default agent.
func (h *Handler) buildStatus() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.mu.RLock()
	defaultName := h.defaultName
	h.mu.RUnlock()

	dsproxyStatusArgs := dsproxyStatusArgsForProfile(defaultName)
	dsproxyStatus := runDsproxyCommand(ctx, dsproxyStatusArgs...)
	dsproxyVersionText := runDsproxyCommand(ctx, "--version")
	weclawVersionText := runCurrentExecutableCommand(ctx, "version")

	weclawPublic, weclawInternal := runtimeVersionLines(weclawVersionText, "weclaw public version:", "weclaw internal version:")
	dsproxyPublic, dsproxyInternal := runtimeVersionLines(dsproxyVersionText, "public version:", "internal version:")

	return commandCard(
		"ℹ️ Info",
		"```text",
		"WeClaw   public   "+weclawPublic,
		"         internal "+weclawInternal,
		"         uptime   "+currentProcessUptime(),
		"dsproxy  public   "+dsproxyPublic,
		"         internal "+dsproxyInternal,
		"         uptime   "+dsproxyUptimeFromStatus(dsproxyStatus),
		"```",
	)
}

func buildHelpText() string {
	return strings.Join([]string{
		"## 📖 WeClaw commands",
		"",
		"> Common WeChat-side commands. `/info` is kept as a hidden compatibility alias; use `/status` instead.",
		"",
		"```text",
		"QUICK MAP",
		"status   /status  /now  /cancel",
		"runtime  /model   /effort  /balance",
		"session  /restart /new     /clear",
		"route    @agent   /agent   /cwd",
		"```",
		"",
		"### 🧩 Status",
		"- `/status`: compact runtime dashboard.",
		"- `/now`: current task progress or idle session context.",
		"- `/cancel`: request cancellation for the active turn.",
		"- `/help`: show this help card.",
		"",
		"### ⚙️ DeepSeek runtime",
		"- `/model deepseek-v4-pro|deepseek-v4-flash`: switch model and preserve the session.",
		"- `/effort high|max`: switch reasoning effort and preserve the session.",
		"- `/balance`: show backend account balance when supported.",
		"",
		"### 🧵 Session",
		"- `/restart`: create a new session for the current profile.",
		"- `/new` or `/clear`: create a new session.",
		"- `/profile deepseek|deepseek-thinking`: switch profile.",
		"- `/cwd /path`: change working directory.",
		"",
		"### 💬 Routing",
		"- `@agent` or `/agent`: switch default agent.",
		"- `@agent message` or `/agent message`: send to one agent.",
		"- `@a @b message`: broadcast to multiple agents.",
		"",
		"### 🔗 Aliases",
		"- `/cc` `/cx` `/cs` `/km` `/gm`: claude, codex, cursor, kimi, gemini.",
		"- `/oc` `/ocd` `/pi` `/cp`: openclaw, opencode, pi, copilot.",
		"- `/dr` `/if` `/kr` `/qw`: droid, iflow, kiro, qwen.",
		"",
		"### 🛡️ Safety",
		"- Unknown slash commands are intercepted locally and are not sent to the agent.",
	}, "\n")
}

func extractText(msg ilink.WeixinMessage) string {
	for _, item := range msg.ItemList {
		if item.Type == ilink.ItemTypeText && item.TextItem != nil {
			return item.TextItem.Text
		}
	}
	return ""
}

func extractImage(msg ilink.WeixinMessage) *ilink.ImageItem {
	for _, item := range msg.ItemList {
		if item.Type == ilink.ItemTypeImage && item.ImageItem != nil {
			return item.ImageItem
		}
	}
	return nil
}

func extractVoiceText(msg ilink.WeixinMessage) string {
	for _, item := range msg.ItemList {
		if item.Type == ilink.ItemTypeVoice && item.VoiceItem != nil && item.VoiceItem.Text != "" {
			return item.VoiceItem.Text
		}
	}
	return ""
}

func (h *Handler) handleImageSave(ctx context.Context, client *ilink.Client, msg ilink.WeixinMessage, img *ilink.ImageItem) {
	clientID := NewClientID()
	log.Printf("[handler] received image from %s, saving to %s", msg.FromUserID, h.saveDir)

	// Download image data
	var data []byte
	var err error

	if img.URL != "" {
		// Direct URL download
		data, _, err = downloadFile(ctx, img.URL)
	} else if img.Media != nil && img.Media.EncryptQueryParam != "" {
		// CDN encrypted download
		data, err = DownloadFileFromCDN(ctx, img.Media.EncryptQueryParam, img.Media.AESKey)
	} else {
		log.Printf("[handler] image has no URL or media info from %s", msg.FromUserID)
		return
	}

	if err != nil {
		log.Printf("[handler] failed to download image from %s: %v", msg.FromUserID, err)
		reply := fmt.Sprintf("Failed to save image: %v", err)
		_ = SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID)
		return
	}

	// Detect extension from content
	ext := detectImageExt(data)

	// Generate filename with timestamp
	ts := time.Now().Format("20060102-150405")
	fileName := fmt.Sprintf("%s%s", ts, ext)
	filePath := filepath.Join(h.saveDir, fileName)

	// Ensure save directory exists
	if err := os.MkdirAll(h.saveDir, 0o755); err != nil {
		log.Printf("[handler] failed to create save dir: %v", err)
		return
	}

	// Write image file
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		log.Printf("[handler] failed to write image: %v", err)
		reply := fmt.Sprintf("Failed to save image: %v", err)
		_ = SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID)
		return
	}

	// Write sidecar file
	sidecarPath := filePath + ".sidecar.md"
	sidecarContent := fmt.Sprintf("---\nid: %s\n---\n", uuid.New().String())
	if err := os.WriteFile(sidecarPath, []byte(sidecarContent), 0o644); err != nil {
		log.Printf("[handler] failed to write sidecar: %v", err)
	}

	log.Printf("[handler] saved image to %s (%d bytes)", filePath, len(data))
	reply := fmt.Sprintf("Saved: %s", fileName)
	if err := SendTextReply(ctx, client, msg.FromUserID, reply, msg.ContextToken, clientID); err != nil {
		log.Printf("[handler] failed to send reply to %s: %v", msg.FromUserID, err)
	}
}

func detectImageExt(data []byte) string {
	if len(data) < 4 {
		return ".bin"
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ".png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}
	// GIF: 47 49 46
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return ".gif"
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[8] == 0x57 && data[9] == 0x45 {
		return ".webp"
	}
	// BMP: 42 4D
	if data[0] == 0x42 && data[1] == 0x4D {
		return ".bmp"
	}
	return ".jpg" // default to jpg for WeChat images
}

func buildContextUsageLines(ag agent.Agent, userID, sessionID string, fallbackWindow int64, windowSource string) []string {
	_ = sessionID
	_ = windowSource
	panel := buildCompactStatusPanel(ag, userID, fallbackWindow, "", "", "", "balance n/a")
	return visualCommandFence("", panel...)
}

func formatContextWindowLines(window, used int64) []string {
	return visualCommandFence(
		"",
		fmt.Sprintf(
			"Context  [%s]  %s  %s/%s",
			formatCommandProgressBar(used, window, 20),
			formatTokenPercent(used, window),
			formatTokenCount(maxInt64(used, 0)),
			formatContextLimit(window),
		),
	)
}

func formatContextUsageValue(used, window int64) string {
	if used < 0 {
		used = 0
	}
	return fmt.Sprintf("%s (%s)", formatTokenCount(used), formatTokenPercent(used, window))
}

func formatContextLeftValue(used, window int64) string {
	if window <= 0 {
		return "-- (0.0%)"
	}
	if used < 0 {
		used = 0
	}
	left := window - used
	if left < 0 {
		left = 0
	}
	return fmt.Sprintf("%s (%s)", formatTokenCount(left), formatTokenPercent(left, window))
}

func formatContextLimit(window int64) string {
	if window <= 0 {
		return "--"
	}
	return formatTokenCount(window)
}

func zeroTokenUsageLines(source string) []string {
	if source == "" {
		source = "unavailable"
	}
	return []string{
		"| Metric | Tokens |",
		"| --- | ---: |",
		"| total | 0 |",
		"| input | 0 |",
		"| cached input | 0 |",
		"| output | 0 |",
		"| reasoning output | 0 |",
		"| tools | -- |",
		"| other | -- |",
		"",
		"- total: 0",
		"- input: 0",
		"- cached input: 0",
		"- output: 0",
		"- reasoning output: 0",
		"- tools: --",
		"- other: --",
		"- source: " + source,
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func formatTokenCount(n int64) string {
	if n < 0 {
		return "--"
	}
	if n >= 1000000 {
		return trimFixedDecimal(float64(n)/1000000.0) + "M"
	}
	if n >= 1000 {
		return trimFixedDecimal(float64(n)/1000.0) + "k"
	}
	return fmt.Sprintf("%d", n)
}

func formatTokenPercent(used, window int64) string {
	if window <= 0 {
		return "0.0%"
	}
	if used < 0 {
		used = 0
	}
	if used > window {
		used = window
	}
	return fmt.Sprintf("%.1f%%", float64(used)*100/float64(window))
}

func dsproxyStatusArgsForProfile(profile string) []string {
	if profile == "deepseek-thinking" {
		return []string{"status", "thinking"}
	}
	return []string{"status"}
}

func formatContextWindowLine(window int64) string {
	return "• limit: " + formatContextLimit(window)
}

func unknownContextUsageLines(window int64) []string {
	lines := formatContextWindowLines(window, 0)
	lines = append(lines[:len(lines)-1], "Tokens   waiting for Codex usage event", "```")
	return lines
}

func formatContextUsageLine(used, window int64, hasUsage bool) string {
	if !hasUsage {
		used = 0
	}
	return "• used: " + formatContextUsageValue(used, window)
}

func trimFixedDecimal(value float64) string {
	s := fmt.Sprintf("%.1f", value)
	return strings.TrimSuffix(s, ".0")
}
