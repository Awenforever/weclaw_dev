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

func withDsproxyCommandRunner(t *testing.T, runner func(context.Context, ...string) string) {
	t.Helper()
	oldRunner := dsproxyCommandRunner
	oldAllow := dsproxyCommandRunnerAllowExternalInTests
	dsproxyCommandRunner = runner
	dsproxyCommandRunnerAllowExternalInTests = true
	t.Cleanup(func() {
		dsproxyCommandRunner = oldRunner
		dsproxyCommandRunnerAllowExternalInTests = oldAllow
	})
}

func sampleWeClawTelemetryJSON() string {
	return `{
  "status": "ok",
  "profile": "deepseek-thinking",
  "model": {
    "effective_model": "deepseek-v4-flash",
    "codex_model": "glm-5.1",
    "model_conflict": true,
    "display_hint": null,
    "diagnostic_hint": "Codex profile model differs from forced upstream model; dsproxy effective_model is authoritative.",
    "user_visible": false
  },
  "effort": {
    "user_facing": "max",
    "deepseek_reasoning_effort": "max",
    "codex_model_reasoning_effort": "xhigh"
  },
  "context_window": {
    "effective_safe_window_tokens": 750000,
    "used_tokens": null,
    "used_tokens_available": false,
    "used_tokens_source": "not_reported",
    "used_tokens_reason": "context_used_tokens_not_reported_by_codex_or_provider",
    "source": "codex_profile.model_auto_compact_token_limit",
    "is_estimated": false
  },
  "tokens": {
    "last_turn": {
      "available": true,
      "summary": {
        "total_tokens": 50236
      }
    },
    "session_total": {
      "available": true,
      "summary": {
        "total_tokens": 9113787
      }
    },
    "auxiliary_model_calls": {
      "available": true,
      "summary": {
        "total_tokens": 648175
      }
    }
  },
  "cost": {
    "available": true,
    "currency": "USD",
    "is_estimated": true,
    "last_turn_estimated_cost": 0.0001540728,
    "session_estimated_cost": 0.5992581266,
    "auxiliary_estimated_cost": 0.0093673598,
    "usage_available": true,
    "pricing_available": true,
    "pricing_stale": false,
    "reason": null,
    "missing": []
  },
  "balance": {
    "available": true,
    "status": "ok",
    "provider": "deepseek",
    "currency": "CNY",
    "amount": 5.83,
    "display": "5.83 CNY",
    "reason": null,
    "action": null
  },
  "compaction": {
    "available": true,
    "unit": "chars",
    "runtime_context": {
      "compaction": {
        "last_report": {
          "before_chars": 58,
          "effective_trigger_chars": 1250000,
          "effective_target_chars": 750000,
          "keep_recent_messages": 24,
          "policy_decision": {
            "policy": "adaptive"
          },
          "reason": "not_triggered"
        }
      },
      "trimming": {
        "last_report": {
          "before_chars": 219,
          "max_context_chars": 1500000,
          "chars_removed": 0
        }
      }
    }
  }
}`
}

func sampleWeClawTelemetryRound3JSON() string {
	return strings.TrimSuffix(sampleWeClawTelemetryJSON(), "\n}") + `,
  "diagnostics": {
    "available": true,
    "user_visible": false,
    "degraded_fields": [
      {
        "path": "context_window.model_catalog",
        "reason": "model_catalog_entry_not_found",
        "action": "add the effective model to the model catalog or repair the managed Codex profile"
      },
      {
        "path": "semantic_compaction.rollout",
        "reason": "semantic_payload_compaction_not_safe_to_enable",
        "action": "keep semantic payload compaction disabled until blockers clear"
      }
    ],
    "warnings": [
      "model_conflict_hidden_from_normal_status"
    ],
    "actions": [
      "keep semantic payload compaction disabled until blockers clear"
    ]
  },
  "context_window": {
    "display_limit_tokens": 750000,
    "effective_safe_window_tokens": 750000,
    "used_tokens": 87,
    "used_tokens_available": true,
    "used_tokens_is_estimated": true,
    "used_tokens_precision": "estimated_current_context_from_latest_upstream_prompt_tokens",
    "used_tokens_source": "dsproxy_usage_ledger.latest_turn.by_purpose.primary.prompt_tokens",
    "remaining_tokens_estimate": 749913,
    "latest_upstream_prompt_tokens": {
      "available": true,
      "value": 87,
      "unit": "tokens",
      "is_estimated_for_context_window": true,
      "precision": "provider_reported_prompt_tokens_for_latest_upstream_model_call",
      "source": "dsproxy_usage_ledger.latest_turn.by_purpose.primary.prompt_tokens"
    },
    "limit_explanation": {
      "display_limit_tokens": 750000,
      "display_limit_source": "codex_profile.model_auto_compact_token_limit",
      "display_limit_reason": "codex_profile_auto_compact_token_limit",
      "auto_compact_token_limit": 750000,
      "model_context_window_tokens": 1000000,
      "unit": "tokens"
    }
  },
  "tokens": {
    "taxonomy": {
      "version": 3,
      "precision": {
        "provider_usage_totals": "exact_provider_reported",
        "purpose_attribution": "exact_dsproxy_call_purpose",
        "prompt_subcategory_split": "not_reported_by_provider_without_tokenizer",
        "context_window_used_tokens": "estimated_current_context_from_latest_upstream_prompt_tokens"
      }
    },
    "attribution": {
      "provider_usage_totals": {
        "available": true
      },
      "purpose_attribution": {
        "available": true,
        "known_purposes": [
          "primary",
          "tool_bridge",
          "liveness_retry",
          "compaction",
          "semantic_audit"
        ]
      },
      "prompt_subcategory_split": {
        "available": false,
        "reason": "provider_usage_is_aggregate_without_prompt_subcategory_breakdown",
        "action": "display prompt subcategory splits as unavailable until dsproxy adds an audited tokenizer or a provider-backed per-segment ledger"
      },
      "context_window_used_tokens": {
        "available": false,
        "estimate_field": "context_window.latest_upstream_prompt_tokens",
        "estimate_precision": "estimated_current_context_from_latest_upstream_prompt_tokens",
        "action": "use context_window.used_tokens when context_window.used_tokens_available is true; otherwise display an unavailable marker; never derive current context usage from session totals"
      }
    },
    "prompt_subcategory_split": {
      "available": false,
      "reason": "provider_usage_is_aggregate_without_prompt_subcategory_breakdown",
      "action": "display prompt subcategory splits as unavailable until dsproxy adds an audited tokenizer or a provider-backed per-segment ledger"
    },
    "last_turn": {
      "available": true,
      "summary": {
        "total_tokens": 50236
      }
    },
    "session_total": {
      "available": true,
      "summary": {
        "total_tokens": 9113787
      }
    },
    "auxiliary_model_calls": {
      "available": true,
      "summary": {
        "total_tokens": 648175
      }
    }
  },
  "cost": {
    "available": true,
    "currency": "USD",
    "is_estimated": true,
    "last_turn_estimated_cost": 0.0001540728,
    "session_estimated_cost": 0.5992747866,
    "auxiliary_estimated_cost": 0.0093673598,
    "usage_available": true,
    "pricing_available": true,
    "pricing_source_kind": "bundled_official_docs_snapshot",
    "pricing_source_trust": "bundled_official_docs_snapshot",
    "pricing_source_url": "https://api-docs.deepseek.com/quick_start/pricing",
    "pricing_updated_at": "2026-05-17T00:00:00Z",
    "official_pricing_available": false
  },
  "pricing": {
    "available": true,
    "source": "project_default_pricing_config",
    "source_kind": "bundled_official_docs_snapshot",
    "source_trust": "bundled_official_docs_snapshot",
    "source_url": "https://api-docs.deepseek.com/quick_start/pricing",
    "official_reference_url": "https://api-docs.deepseek.com/quick_start/pricing",
    "updated_at": "2026-05-17T00:00:00Z",
    "snapshot_created_at": "2026-05-17T00:00:00Z",
    "prices": {
      "input_cache_hit": 0.0028,
      "input_cache_miss": 0.14,
      "output": 0.28
    },
    "pricing_source_state": {
      "cost_uses_current_prices": true,
      "current_prices_are_bundled_official_snapshot": true,
      "current_prices_are_external_config": false,
      "current_prices_are_official_live_cache": false,
      "must_display_source_label": true
    },
    "official_source": {
      "available": false,
      "source_kind": "official_docs_html",
      "source_url": "https://api-docs.deepseek.com/quick_start/pricing",
      "reason": "official_pricing_cache_not_available_for_active_status",
      "requires_refresh": true
    }
  },
  "semantic_compaction": {
    "rollout": {
      "safe_to_enable_payload_compaction": false,
      "current_payload_mode": "dry_run",
      "blockers": [
        "semantic_audit_event_missing",
        "semantic_policy_dry_run_event_missing",
        "semantic_payload_compaction_event_missing"
      ],
      "missing_events": [
        "semantic_audit",
        "semantic_policy_dry_run",
        "semantic_payload_compaction"
      ],
      "action": "keep semantic payload compaction disabled until blockers clear; use debug semantic selftest and canary checks for validation"
    }
  }
}`
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
	for _, want := range []string{
		"## 🧩 Status",
		"**Profile:**",
		"**Model:**",
		"**Session:**",
		"Context  [",
		"Tokens",
		"Cost     session n/a  last n/a",
		"Proxy    default · 127.0.0.1:8000",
		"Contract unavailable",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	if strings.Contains(reply, "📊 Context window") || strings.Contains(reply, "🔌 Proxy") || strings.Contains(reply, "📁 Paths") {
		t.Fatalf("status reply = %q, should not contain old verbose section headings", reply)
	}
}

func TestRuntimeControlRestartWithoutDefault(t *testing.T) {
	h := NewHandler(nil, nil)

	reply, ok := h.handleRuntimeControl(context.Background(), "/restart", "user-1")
	if !ok {
		t.Fatal("/restart should be intercepted")
	}
	if !strings.Contains(reply, "No default profile is configured") {
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
	for _, want := range []string{
		"## 🔁 Profile updated",
		"Profile:** `deepseek`",
		"Session:** preserved when available",
		"Thinking:** disabled",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply = %q, want %q", reply, want)
		}
	}
	for _, old := range []string{"## 🧵 Session", "Next     send a message", "Restart  use /restart"} {
		if strings.Contains(reply, old) {
			t.Fatalf("reply = %q, should not contain noisy token %q", reply, old)
		}
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
	for _, want := range []string{
		"## 🔁 Profile updated",
		"Profile:** `deepseek-thinking`",
		"Session:** preserved when available",
		"Thinking:** enabled",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply = %q, want %q", reply, want)
		}
	}
	for _, old := range []string{"## 🧵 Session", "Next     send a message", "Restart  use /restart"} {
		if strings.Contains(reply, old) {
			t.Fatalf("reply = %q, should not contain noisy token %q", reply, old)
		}
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
	if !strings.Contains(reply, "Config   preserved") {
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

func TestRuntimeControlEffortUsesDsproxyProfileContract(t *testing.T) {
	var gotArgs []string
	withDsproxyCommandRunner(t, func(ctx context.Context, args ...string) string {
		gotArgs = append([]string(nil), args...)
		return `{"status":"ok","effort":{"user_facing":"max","deepseek_reasoning_effort":"max","codex_model_reasoning_effort":"xhigh"}}`
	})

	h := NewHandler(nil, nil)
	h.defaultName = "deepseek-thinking"

	reply, ok := h.handleRuntimeControl(context.Background(), "/effort max", "user-1")
	if !ok {
		t.Fatal("/effort should be intercepted")
	}
	wantArgs := []string{"profile", "set-effort", "deepseek-thinking", "max", "--json"}
	if strings.Join(gotArgs, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("dsproxy args = %#v, want %#v", gotArgs, wantArgs)
	}
	for _, want := range []string{
		"## ✅ Effort updated",
		"Profile:** `deepseek-thinking`",
		"Effort:** `max`",
		"Applied through dsproxy profile contract",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply = %q, want %q", reply, want)
		}
	}
	for _, forbidden := range []string{"Codex profile", ".codex", "xhigh"} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("reply = %q, should not contain %q", reply, forbidden)
		}
	}
}

func TestRuntimeControlEffortOnlySupportsDeepSeekProfiles(t *testing.T) {
	h := NewHandler(nil, nil)
	h.defaultName = "codex"

	reply, ok := h.handleRuntimeControl(context.Background(), "/effort max", "user-1")
	if !ok {
		t.Fatal("/effort should be intercepted")
	}
	for _, want := range []string{
		"## ⛔ DeepSeek only",
		"`/effort` is only supported",
		"`deepseek`",
		"`deepseek-thinking`",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply = %q, want %q", reply, want)
		}
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
	if !strings.Contains(reply, "`thread-now-123`") {
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

func TestRuntimeControlNowIdleAppliesPendingResumeBeforeEnsure(t *testing.T) {
	ag := &runtimeControlTestAgent{
		info:            agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		ensureSessionID: "thread-created-unexpected",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)
	h.SetPendingResume("deepseek-thinking", "thread-resume-now")

	reply, ok := h.handleRuntimeControl(context.Background(), "/now", "user-1")
	if !ok {
		t.Fatal("/now should be intercepted")
	}
	if ag.currentSessionID != "thread-resume-now" {
		t.Fatalf("currentSessionID = %q, want thread-resume-now", ag.currentSessionID)
	}
	if ag.ensureCalls != 0 {
		t.Fatalf("ensureCalls = %d, want 0 because pending resume should be applied before ensure", ag.ensureCalls)
	}
	if !strings.Contains(reply, "`thread-resume-now`") {
		t.Fatalf("reply = %q, want compact resumed session ID", reply)
	}
	if strings.Contains(reply, "thread-created-unexpected") {
		t.Fatalf("reply = %q, should not show newly ensured session", reply)
	}
}

func TestRuntimeControlStatusAppliesPendingResumeBeforeEnsure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(home+"/.codex", 0o755); err != nil {
		t.Fatalf("create codex dir: %v", err)
	}
	if err := os.WriteFile(home+"/.codex/config.toml", []byte("[profiles.deepseek-thinking]\nmodel_context_window = 1000000\n"), 0o644); err != nil {
		t.Fatalf("write codex config: %v", err)
	}

	ag := &runtimeControlTestAgent{
		info:            agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		ensureSessionID: "thread-created-unexpected",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)
	h.SetPendingResume("deepseek-thinking", "thread-resume-status")

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted")
	}
	if ag.currentSessionID != "thread-resume-status" {
		t.Fatalf("currentSessionID = %q, want thread-resume-status", ag.currentSessionID)
	}
	if ag.ensureCalls != 0 {
		t.Fatalf("ensureCalls = %d, want 0 because pending resume should be applied before ensure", ag.ensureCalls)
	}
	if !strings.Contains(reply, "`thread-resume-status`") {
		t.Fatalf("reply = %q, want compact resumed session ID", reply)
	}
	if strings.Contains(reply, "thread-created-unexpected") {
		t.Fatalf("reply = %q, should not show newly ensured session", reply)
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
	if !strings.Contains(reply, "Running  no") {
		t.Fatalf("reply = %q, want compact idle status", reply)
	}
	if !strings.Contains(reply, "`thread-idle-123`") {
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
	if !strings.Contains(reply, "Running  no") {
		t.Fatalf("reply = %q, want compact idle status", reply)
	}
	if !strings.Contains(reply, "`thread-ensure-123`") {
		t.Fatalf("reply = %q, want ensured session ID", reply)
	}
}

func TestRuntimeControlStatusUsesDsproxyTelemetryContract(t *testing.T) {
	withDsproxyCommandRunner(t, func(ctx context.Context, args ...string) string {
		wantArgs := []string{"status", "thinking", "--weclaw-json"}
		if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
			t.Fatalf("dsproxy args = %#v, want %#v", args, wantArgs)
		}
		return sampleWeClawTelemetryJSON()
	})

	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "stale-agent-model"},
		currentSessionID: "thread-telemetry-1",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted")
	}
	for _, want := range []string{
		"## 🧩 Status",
		"Profile:** `deepseek-thinking` `ACP`",
		"Model:** `deepseek-v4-flash` `max`",
		"Session:** `thread-telemetry-1`",
		"—/750k",
		"Tokens   last 50.2k  session 9.1M  aux 648.2k",
		"EstCost session $0.5993  last $0.000154  aux $0.009367",
		"Balance  5.83 CNY",
		"Compact [",
		"58/1.2M chars · not triggered",
		"Trim    [",
		"219/1.5M chars · removed 0",
		"Proxy    thinking · 127.0.0.1:8001 · reachable",
		"Paths    cfg ~/.weclaw/config.json · log ~/.weclaw/weclaw.log",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	for _, forbidden := range []string{"stale-agent-model", "source:", "tools:", "other:", "last turn id:", "Model    codex", "codex_profile.model_auto_compact_token_limit", "missing usage_attribution", "balance_client_unavailable", "9.1M/750k", "100.0%"} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("status reply = %q, should not contain %q", reply, forbidden)
		}
	}
}

func TestRuntimeControlStatusShowsRound3CompactSummary(t *testing.T) {
	withDsproxyCommandRunner(t, func(ctx context.Context, args ...string) string {
		wantArgs := []string{"status", "thinking", "--weclaw-json"}
		if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
			t.Fatalf("dsproxy args = %#v, want %#v", args, wantArgs)
		}
		return sampleWeClawTelemetryRound3JSON()
	})

	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "stale-agent-model"},
		currentSessionID: "thread-round3-compact",
	}
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek-thinking", ag)

	reply, ok := h.handleRuntimeControl(context.Background(), "/status", "user-1")
	if !ok {
		t.Fatal("/status should be intercepted")
	}
	for _, want := range []string{
		"## 🧩 Status",
		"Profile:** `deepseek-thinking` `ACP`",
		"Model:** `deepseek-v4-flash` `max`",
		"Session:** `thread-round3-compact`",
		"Context  [",
		"Tokens   last 50.2k  session 9.1M  aux 648.2k",
		"EstCost session $0.5993  last $0.000154  aux $0.009367",
		"Balance  5.83 CNY",
		"87/750k",
		"Pricing  hit $0.0028/M miss $0.14/M out $0.28/M · updated 2026-05-17",
		"Policy   adaptive · trigger 1.2M chars · target 750k · keep 24",
		"Compact [",
		"Proxy    thinking · 127.0.0.1:8001 · reachable",
		"Paths    cfg ~/.weclaw/config.json · log ~/.weclaw/weclaw.log",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	for _, forbidden := range []string{
		"Status Debug",
		"Diag     degraded",
		"Attrib v3",
		"Semantic safe",
		"user_tokens",
		"assistant_history_tokens",
		"tool_tokens",
		"environment_tokens",
		"runtime_tokens",
		"semantic payload compaction enabled",
		"Pricing  default config",
		"refresh yes",
	} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("status reply = %q, should not contain %q", reply, forbidden)
		}
	}
}

func TestRuntimeControlStatusDebugAliasesAreRemoved(t *testing.T) {
	h := NewHandler(nil, nil)

	for _, command := range []string{"/status verbose", "/status debug", "/status noisy"} {
		reply, ok := h.handleRuntimeControl(context.Background(), command, "user-1")
		if !ok {
			t.Fatalf("%s should be intercepted as usage", command)
		}
		for _, want := range []string{"## ℹ️ Usage", "Show the compact runtime dashboard.", "/status"} {
			if !strings.Contains(reply, want) {
				t.Fatalf("%s usage reply = %q, want %q", command, reply, want)
			}
		}
		for _, forbidden := range []string{"/status verbose", "/status debug", "Status Debug", "Diagnostics"} {
			if strings.Contains(reply, forbidden) {
				t.Fatalf("%s usage reply = %q, should not contain %q", command, reply, forbidden)
			}
		}
	}
}

func TestDsproxyContextUsedTokensRequiresExplicitAvailability(t *testing.T) {
	payload, ok := parseJSONMap(sampleWeClawTelemetryJSON())
	if !ok {
		t.Fatal("sample telemetry JSON should parse")
	}
	line := formatDsproxyContextLine(payload)
	if !strings.Contains(line, "—/750k") {
		t.Fatalf("context line = %q, want unavailable used-token marker", line)
	}
	if strings.Contains(line, "9.1M/750k") || strings.Contains(line, "100.0%") {
		t.Fatalf("context line = %q, must not use session_total as context used tokens", line)
	}

	payload["context_window"].(map[string]any)["used_tokens_available"] = true
	payload["context_window"].(map[string]any)["used_tokens"] = float64(375000)
	line = formatDsproxyContextLine(payload)
	for _, want := range []string{"50.0%", "375k/750k"} {
		if !strings.Contains(line, want) {
			t.Fatalf("context line = %q, want %q", line, want)
		}
	}
}

func TestDsproxyContextLineMarksEstimatedUsage(t *testing.T) {
	payload, ok := parseJSONMap(sampleWeClawTelemetryRound3JSON())
	if !ok {
		t.Fatal("sample round4 telemetry JSON should parse")
	}
	line := formatDsproxyContextLine(payload)
	for _, want := range []string{"0.0%", "87/750k"} {
		if !strings.Contains(line, want) {
			t.Fatalf("context line = %q, want %q", line, want)
		}
	}
	if strings.Contains(line, "9.1M/750k") {
		t.Fatalf("context line = %q, must not use session_total as context used tokens", line)
	}
}

func TestDsproxyPricingSummaryShowsSnapshotPrices(t *testing.T) {
	payload, ok := parseJSONMap(sampleWeClawTelemetryRound3JSON())
	if !ok {
		t.Fatal("sample round4 telemetry JSON should parse")
	}
	line := formatDsproxyPricingSummaryLine(payload)
	for _, want := range []string{
		"Pricing  hit $0.0028/M",
		"miss $0.14/M",
		"out $0.28/M",
		"updated 2026-05-17",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("pricing line = %q, want %q", line, want)
		}
	}
	for _, forbidden := range []string{"default config", "refresh yes", "Pricing  bundled official snapshot ·"} {
		if strings.Contains(line, forbidden) {
			t.Fatalf("pricing line = %q, should not contain %q", line, forbidden)
		}
	}
}

func TestDsproxyRuntimePayloadGuardLinesUseRealtimeChars(t *testing.T) {
	payload := map[string]any{
		"runtime_payload_guard": map[string]any{
			"available": true,
			"unit":      "chars",
			"compaction": map[string]any{
				"available":     true,
				"current_chars": float64(60),
				"trigger_chars": float64(1250000),
				"usage_ratio":   0.000048,
				"status":        "not_triggered",
			},
			"trimming": map[string]any{
				"available":         true,
				"current_chars":     float64(174),
				"max_context_chars": float64(1500000),
				"usage_ratio":       0.000116,
				"status":            "not_triggered",
			},
		},
		"compaction": map[string]any{
			"available": true,
			"unit":      "chars",
			"runtime_context": map[string]any{
				"compaction": map[string]any{
					"config": map[string]any{
						"trigger_chars": float64(900000),
					},
					"last_report": map[string]any{
						"exists": false,
					},
				},
				"trimming": map[string]any{
					"config": map[string]any{
						"max_context_chars": float64(1500000),
					},
					"last_report": map[string]any{
						"exists": false,
					},
				},
			},
		},
	}
	lines := formatDsproxyCompactionLines(payload)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Compact [",
		"0.0%  60/1.2M chars · not triggered",
		"Trim    [",
		"0.0%  174/1.5M chars · not triggered",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("runtime payload guard lines = %q, want %q", joined, want)
		}
	}
	for _, forbidden := range []string{"no report", "0/-- chars", "--/900k chars"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("runtime payload guard lines = %q, should not contain %q", joined, forbidden)
		}
	}
}

func TestDsproxyCompactionLinesFallbackToConfigWhenReportsMissing(t *testing.T) {
	payload := map[string]any{
		"compaction": map[string]any{
			"available": true,
			"unit":      "chars",
			"runtime_context": map[string]any{
				"compaction": map[string]any{
					"config": map[string]any{
						"trigger_chars": float64(900000),
					},
					"last_report": map[string]any{
						"exists": false,
					},
				},
				"trimming": map[string]any{
					"config": map[string]any{
						"max_context_chars": float64(1500000),
					},
					"last_report": map[string]any{
						"exists": false,
					},
				},
			},
		},
	}
	lines := formatDsproxyCompactionLines(payload)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Compact [",
		"n/a  --/900k chars · no report",
		"Trim    [",
		"n/a  --/1.5M chars · no report",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("compaction lines = %q, want %q", joined, want)
		}
	}
	if strings.Contains(joined, "0/-- chars") {
		t.Fatalf("compaction lines = %q, should not contain invalid 0/-- chars", joined)
	}
}

func TestRuntimeControlStatusReportsTokenUsage(t *testing.T) {
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
		"## 🧩 Status",
		"Profile:** `deepseek-thinking` `ACP`",
		"Model:** `deepseek-v4-pro`",
		"Session:** `thread-usage-1`",
		"Context  [",
		"16.9%",
		"43.6k/258.4k",
		"Tokens   in 43.2k  cached 1.2k  out 350  reason 17  last 22.3k",
		"Cost     session n/a  last n/a",
		"Contract unavailable",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	for _, old := range []string{"📊 Context window", "limit:", "left:", "source:", "tools:", "other:", "last turn id:"} {
		if strings.Contains(reply, old) {
			t.Fatalf("status reply = %q, should not contain old verbose token %q", reply, old)
		}
	}
}

func TestRuntimeControlStatusShowsFallbackContextWindowWhileUsageIsWaiting(t *testing.T) {
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
		"## 🧩 Status",
		"Session:** `thread-waiting-1`",
		"Context  [",
		"0.0%",
		"0/--",
		"Tokens   waiting for Codex usage event",
		"Cost     session n/a  last n/a",
		"Contract unavailable",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	for _, old := range []string{"limit:", "left:", "source:", "tools:", "other:"} {
		if strings.Contains(reply, old) {
			t.Fatalf("status reply = %q, should not contain old verbose token %q", reply, old)
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

func TestFormatContextWindowLines(t *testing.T) {
	got := strings.Join(formatContextWindowLines(258400, 12920), "\n")
	for _, want := range []string{
		"```text",
		"Context  [",
		"5.0%",
		"12.9k/258.4k",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatContextWindowLines = %q, want %q", got, want)
		}
	}
	for _, old := range []string{"limit:", "left:", "CONTEXT WINDOW", "| Metric |"} {
		if strings.Contains(got, old) {
			t.Fatalf("formatContextWindowLines = %q, should not contain old token %q", got, old)
		}
	}
}

func TestRuntimeControlStatusFallsBackToSnapshotWindowWhenConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ag := &runtimeControlTestAgent{
		info:             agent.AgentInfo{Name: "deepseek-thinking", Type: "acp", Model: "deepseek-v4-pro"},
		currentSessionID: "thread-snapshot-window",
		tokenUsageOK:     true,
		tokenUsage: agent.TokenUsageSnapshot{
			ThreadID:           "thread-snapshot-window",
			TurnID:             "turn-snapshot-window",
			ModelContextWindow: 258400,
			Total: agent.TokenUsageBreakdown{
				TotalTokens:  12920,
				InputTokens:  12000,
				OutputTokens: 920,
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
		"Session:** `thread-snapshot-window`",
		"Context  [",
		"5.0%",
		"12.9k/258.4k",
		"Tokens   in 12k  cached 0  out 920  reason 0",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status reply = %q, want %q", reply, want)
		}
	}
	for _, old := range []string{"used: unknown", "left: unknown", "source:", "last turn id:"} {
		if strings.Contains(reply, old) {
			t.Fatalf("status reply = %q, should not contain old token %q", reply, old)
		}
	}
}

func TestCommandCardUsesRealNewlines(t *testing.T) {
	got := commandCard("🧩 Agent", "- profile: deepseek-thinking", "Context  [", "| Metric | Value |", "| --- | --- |", "| Used | 4.4% |")
	if strings.Contains(got, `\n`) {
		t.Fatalf("commandCard() leaked literal backslash-n: %q", got)
	}
	for _, want := range []string{
		"## 🧩 Agent\n\n- profile: deepseek-thinking",
		"Context  [",
		"| Metric | Value |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("commandCard() = %q, want %q", got, want)
		}
	}
}

func TestFormatContextWindowLinesUsesVisualPanel(t *testing.T) {
	got := strings.Join(formatContextWindowLines(258400, 12920), "\n")
	for _, want := range []string{
		"```text\nContext  [",
		"5.0%",
		"12.9k/258.4k",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatContextWindowLines = %q, want %q", got, want)
		}
	}
	for _, old := range []string{"CONTEXT WINDOW", "| Used | 12.9k (5.0%) |", "limit:", "left:"} {
		if strings.Contains(got, old) {
			t.Fatalf("formatContextWindowLines = %q, should not contain old token %q", got, old)
		}
	}
	if strings.Contains(got, `\n`) {
		t.Fatalf("formatContextWindowLines leaked literal backslash-n: %q", got)
	}
}

func TestBuildHelpTextUsesDisplayEffects(t *testing.T) {
	text := buildHelpText()
	for _, want := range []string{
		"## 📖 WeClaw commands",
		"Common WeChat-side commands",
		"```text\nQUICK MAP",
		"### 🧩 Status",
		"- `/status`:",
		"### ⚙️ DeepSeek runtime",
		"### 🛡️ Safety",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("buildHelpText() = %q, want %q", text, want)
		}
	}
	if strings.Contains(text, "| Command | Action |") {
		t.Fatalf("buildHelpText() should avoid wide mobile tables, got %q", text)
	}
}
