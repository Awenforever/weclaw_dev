package messaging

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
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
			return "Usage: /status", true
		}
		return h.buildStatusDiagnostics(ctx), true

	case "/balance":
		if len(fields) != 1 {
			return "Usage: /balance", true
		}
		return runDsproxyCommand(ctx, "balance"), true

	case "/model":
		if len(fields) != 2 {
			return "Usage: /model deepseek-v4-pro|deepseek-v4-flash", true
		}
		switch fields[1] {
		case "deepseek-v4-pro", "deepseek-v4-flash":
			return runDsproxyCommand(ctx, "config", "set-model", fields[1]), true
		default:
			return "Unsupported model. Allowed values: deepseek-v4-pro, deepseek-v4-flash", true
		}

	case "/effort":
		if len(fields) != 2 {
			return "Usage: /effort medium|high|xhigh", true
		}
		switch fields[1] {
		case "medium", "high", "xhigh":
			return runDsproxyCommand(ctx, "config", "set-effort", fields[1]), true
		default:
			return "Unsupported effort. Allowed values: medium, high, xhigh", true
		}

	case "/profile":
		if len(fields) != 2 {
			return "Usage: /profile deepseek|deepseek-thinking", true
		}
		switch fields[1] {
		case "deepseek", "deepseek-thinking":
			return h.restartProfileAgent(ctx, fields[1], userID), true
		default:
			return "Unsupported profile. Allowed values: deepseek, deepseek-thinking", true
		}

	case "/restart":
		if len(fields) != 1 {
			return "Usage: /restart", true
		}
		return h.restartCurrentDefaultAgent(ctx, userID), true
	}

	return "", false
}

func (h *Handler) buildStatusDiagnostics(ctx context.Context) string {
	parts := []string{h.buildStatus()}

	dsproxyStatus := runDsproxyCommand(ctx, "status")
	if strings.TrimSpace(dsproxyStatus) != "" {
		parts = append(parts, "dsproxy status:\n"+dsproxyStatus)
	}

	dsproxyConfig := runDsproxyCommand(ctx, "config", "show")
	if strings.TrimSpace(dsproxyConfig) != "" {
		parts = append(parts, "dsproxy config:\n"+dsproxyConfig)
	}

	return strings.Join(parts, "\n\n")
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

func (h *Handler) restartProfileAgent(ctx context.Context, name, userID string) string {
	if !h.isKnownAgent(name) {
		return fmt.Sprintf("Profile %q is not configured. Run weclaw start %s once, or ensure codex is installed and auto-detection can register it.", name, name)
	}

	h.mu.Lock()
	oldDefaultName := h.defaultName
	oldDefault := h.agents[oldDefaultName]
	oldTarget := h.agents[name]
	delete(h.agents, name)
	if oldDefaultName != "" && oldDefaultName != name {
		delete(h.agents, oldDefaultName)
	}
	h.mu.Unlock()

	stopAgentIfSupported(name, oldTarget)
	if oldDefaultName != "" && oldDefaultName != name {
		stopAgentIfSupported(oldDefaultName, oldDefault)
	}

	reply := h.switchDefault(ctx, name)
	h.mu.RLock()
	ready := h.defaultName == name && h.agents[name] != nil
	h.mu.RUnlock()
	if !ready {
		return reply
	}

	sessionReply := h.resetDefaultSession(ctx, userID)
	return reply + "\n" + sessionReply
}

func (h *Handler) restartCurrentDefaultAgent(ctx context.Context, userID string) string {
	h.mu.RLock()
	name := h.defaultName
	ag := h.agents[name]
	h.mu.RUnlock()

	if name == "" {
		return "No default agent configured."
	}

	h.mu.Lock()
	delete(h.agents, name)
	h.mu.Unlock()

	stopAgentIfSupported(name, ag)

	reply := h.switchDefault(ctx, name)
	h.mu.RLock()
	ready := h.defaultName == name && h.agents[name] != nil
	h.mu.RUnlock()
	if !ready {
		return reply
	}

	sessionReply := h.resetDefaultSession(ctx, userID)
	return reply + "\n" + sessionReply
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
		var err error
		reply, streamedText, err = h.chatWithAgentWithoutStreaming(ctx, ag, msg.FromUserID, text)
		if err != nil {
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

	reply, streamedText, err := h.chatWithAgentWithoutStreaming(ctx, ag, msg.FromUserID, message)
	if err != nil {
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
	return PlainTextReplyChunks(reply)
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
	info := ag.Info()
	log.Printf("[handler] dispatching to agent without WeChat streaming (%s) for %s", info, userID)

	start := time.Now()
	reply, err := ag.Chat(ctx, userID, message)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[handler] agent error (%s, elapsed=%s): %v", info, elapsed, err)
		return "", false, err
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
		chunks := PlainTextReplyChunks(evt.Text)
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
		return fmt.Sprintf("Failed to switch to %q: %v", name, err)
	}

	h.mu.Lock()
	old := h.defaultName
	h.defaultName = name
	h.agents[name] = ag
	h.mu.Unlock()

	// Persist to config file
	if h.saveDefault != nil {
		if err := h.saveDefault(name); err != nil {
			log.Printf("[handler] failed to save default agent to config: %v", err)
		} else {
			log.Printf("[handler] saved default agent %q to config", name)
		}
	}

	info := ag.Info()
	log.Printf("[handler] switched default agent: %s -> %s (%s)", old, name, info)
	return fmt.Sprintf("Switched default agent to %s.", userFacingAgentLabel(name, info))
}

// resetDefaultSession resets the session for the given userID on the default agent.
func (h *Handler) resetDefaultSession(ctx context.Context, userID string) string {
	h.mu.RLock()
	defaultName := h.defaultName
	h.mu.RUnlock()

	ag := h.getDefaultAgent()
	if ag == nil {
		return "No agent running."
	}
	info := ag.Info()
	name := userFacingAgentLabel(defaultName, info)
	sessionID, err := ag.ResetSession(ctx, userID)
	if err != nil {
		log.Printf("[handler] reset session failed for %s: %v", userID, err)
		return fmt.Sprintf("Failed to reset session: %v", err)
	}
	if sessionID != "" {
		return fmt.Sprintf("Created a new %s session: %s", name, sessionID)
	}
	return fmt.Sprintf("Created a new %s session.", name)
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
		// No path provided — show current cwd of default agent
		ag := h.getDefaultAgent()
		if ag == nil {
			return "No agent running."
		}
		info := ag.Info()
		return fmt.Sprintf("cwd: (check agent config)\nagent: %s", info.Name)
	}

	// Expand ~ to home directory
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

	// Resolve to absolute path
	absPath, err := filepath.Abs(arg)
	if err != nil {
		return fmt.Sprintf("Invalid path: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Sprintf("Path not found: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Sprintf("Not a directory: %s", absPath)
	}

	// Update cwd on all running agents
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

	return fmt.Sprintf("cwd: %s", absPath)
}

// buildStatus returns a short status string showing the current default agent.
func (h *Handler) buildStatus() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.defaultName == "" {
		return "agent: none (echo mode)"
	}

	ag, ok := h.agents[h.defaultName]
	if !ok {
		return fmt.Sprintf("agent: %s (not started)", h.defaultName)
	}

	info := ag.Info()
	return fmt.Sprintf("agent: %s\ntype: %s\nmodel: %s", h.defaultName, info.Type, info.Model)
}

func buildHelpText() string {
	return `Available commands:
@agent or /agent - Switch default agent
@agent msg or /agent msg - Send to a specific agent
@a @b msg - Broadcast to multiple agents
/new or /clear - Start a new session
/cwd /path - Switch workspace directory
/info - Show current agent info
/status - Show current agent and DeepSeek proxy diagnostics
/balance - Show DeepSeek account balance
/model deepseek-v4-pro|deepseek-v4-flash - Set DeepSeek model
/effort medium|high|xhigh - Set DeepSeek reasoning effort
/profile deepseek|deepseek-thinking - Switch existing Codex profile
/restart - Restart the current profile session
/help - Show this help message

Aliases: /cc(claude) /cx(codex) /cs(cursor) /km(kimi) /gm(gemini) /oc(openclaw) /ocd(opencode) /pi(pi) /cp(copilot) /dr(droid) /if(iflow) /kr(kiro) /qw(qwen)`
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
