package messaging

import (
	"strings"
	"testing"
)

func TestTokenFirstStatusUnitsKeepOriginalCompactTrimRows(t *testing.T) {
	payload := map[string]any{
		"context_window": map[string]any{
			"display_limit_tokens":          float64(1000000),
			"model_context_window_tokens":   float64(1000000),
			"auto_compact_threshold_tokens": float64(750000),
			"auto_compact_token_limit":      float64(750000),
			"used_tokens_available":         true,
			"used_tokens":                   float64(21577),
		},
		"runtime_payload_guard": map[string]any{
			"compaction": map[string]any{
				"policy":               "adaptive",
				"keep_recent_messages": float64(24),
			},
		},
		"compaction": map[string]any{
			"runtime_trigger_source":   "token_first",
			"estimated_context_tokens": float64(21577),
			"tokens_to_auto_compact":   float64(728423),
			"compacted":                false,
		},
		"token_first_runtime_trim": map[string]any{
			"available":          true,
			"applied":            false,
			"before_tokens":      float64(113),
			"after_tokens":       float64(113),
			"tokens_removed":     float64(0),
			"max_context_tokens": float64(750000),
			"target_met":         true,
		},
	}

	contextLine := formatDsproxyContextLine(payload)
	if !strings.Contains(contextLine, "21.6k/1M") {
		t.Fatalf("context line should use display/model context window as denominator, got: %s", contextLine)
	}

	policyLine := formatDsproxyCompactionPolicySummaryLine(payload)
	if policyLine != "Policy   adaptive · trigger 750k tokens · keep ⤒24 msgs" {
		t.Fatalf("policy line mismatch:\n got: %q", policyLine)
	}

	guardLines := strings.Join(formatDsproxyCompactionLines(payload), "\n")
	for _, want := range []string{
		"Compact [",
		"100.0%  21.6k/21.6k tokens · not triggered",
		"Trim    [",
		"113/113 tokens · not triggered",
	} {
		if !strings.Contains(guardLines, want) {
			t.Fatalf("missing %q in guard lines:\n%s", want, guardLines)
		}
	}
	if strings.Contains(guardLines, "21.6k/750k tokens") {
		t.Fatalf("compact row must use retention semantics, not trigger-threshold semantics:\n%s", guardLines)
	}
}

func TestTokenFirstCompactWithUnavailableTrimStillShowsNoReportTrim(t *testing.T) {
	payload := map[string]any{
		"tokens": map[string]any{
			"last_turn": map[string]any{
				"available": true,
				"total":     float64(21577),
			},
		},
		"compaction": map[string]any{
			"before_tokens": float64(21577),
			"after_tokens":  float64(21577),
			"compacted":     false,
		},
		"runtime_payload_guard": map[string]any{
			"available": true,
			"trimming": map[string]any{
				"available": false,
				"reason":    "runtime_trimming_tokens_unavailable",
			},
			"token_first_runtime_trim": map[string]any{
				"available":      false,
				"before_tokens":  nil,
				"after_tokens":   nil,
				"tokens_removed": float64(0),
				"reason":         "no_runtime_trimming_report_observed",
			},
		},
	}

	got := strings.Join(formatDsproxyCompactionLines(payload), "\n")
	if !strings.Contains(got, "Compact [") || !strings.Contains(got, "21.6k/21.6k tokens · not triggered") {
		t.Fatalf("expected token-first Compact row, got:\n%s", got)
	}
	if !strings.Contains(got, "Trim    [") || !strings.Contains(got, "--/-- chars · no report") {
		t.Fatalf("post-prompt status must keep a Trim no-report row when trim token fields are unavailable, got:\n%s", got)
	}
	if strings.Count(got, "Trim    [") != 1 {
		t.Fatalf("expected exactly one Trim row, got:\n%s", got)
	}
}

func TestTokenFirstTrimMapReadsRuntimePayloadGuardPath(t *testing.T) {
	payload := map[string]any{
		"tokens": map[string]any{
			"last_turn": map[string]any{
				"available": true,
				"total":     float64(21577),
			},
		},
		"compaction": map[string]any{
			"before_tokens": float64(21577),
			"after_tokens":  float64(21577),
			"compacted":     false,
		},
		"runtime_payload_guard": map[string]any{
			"available": true,
			"token_first_runtime_trim": map[string]any{
				"available":      true,
				"before_tokens":  float64(412),
				"after_tokens":   float64(412),
				"tokens_removed": float64(0),
				"applied":        false,
			},
		},
	}

	got := strings.Join(formatDsproxyCompactionLines(payload), "\n")
	if !strings.Contains(got, "Trim    [") || !strings.Contains(got, "412/412 tokens · not triggered") {
		t.Fatalf("expected token-first Trim row from runtime_payload_guard.token_first_runtime_trim, got:\n%s", got)
	}
}

func TestTokenFirstPolicyDoesNotInventCompactTarget(t *testing.T) {
	payload := map[string]any{
		"context_window": map[string]any{
			"auto_compact_threshold_tokens": float64(900000),
			"used_tokens_available":         true,
			"used_tokens":                   float64(1),
		},
		"runtime_payload_guard": map[string]any{
			"compaction": map[string]any{
				"policy":               "adaptive",
				"target_chars":         float64(750000),
				"keep_recent_messages": float64(24),
			},
		},
	}

	got := formatDsproxyCompactionPolicySummaryLine(payload)
	if strings.Contains(got, "target") {
		t.Fatalf("token-first policy must not invent compact target from context window, trigger, or char target; got: %s", got)
	}
	if got != "Policy   adaptive · trigger 900k tokens · keep ⤒24 msgs" {
		t.Fatalf("policy line mismatch: %s", got)
	}
}

func TestTokenFirstStatusUnitsKeepPrePromptGuard(t *testing.T) {
	payload := map[string]any{
		"tokens": map[string]any{
			"cache": map[string]any{
				"last_turn": map[string]any{
					"available":     false,
					"scope":         "current_session",
					"prompt_tokens": float64(0),
				},
				"latest_primary_turn": map[string]any{
					"available":     false,
					"scope":         "current_session",
					"prompt_tokens": float64(0),
				},
				"session": map[string]any{
					"available":     false,
					"scope":         "current_session",
					"prompt_tokens": float64(0),
				},
			},
			"last_turn": map[string]any{
				"available": false,
				"scope":     "current_session",
				"total":     float64(0),
			},
			"latest_primary_turn": map[string]any{
				"available": false,
				"scope":     "current_session",
				"total":     float64(0),
			},
			"session": map[string]any{
				"available": false,
				"scope":     "current_session",
				"total":     float64(0),
			},
		},
		"context_window": map[string]any{
			"auto_compact_threshold_tokens": float64(750000),
			"used_tokens_available":         false,
		},
		"compaction": map[string]any{
			"estimated_context_tokens": float64(21577),
		},
		"token_first_runtime_trim": map[string]any{
			"available":     true,
			"before_tokens": float64(113),
			"after_tokens":  float64(113),
		},
	}
	got := strings.Join(formatDsproxyCompactionLines(payload), "\n")
	if !strings.Contains(got, "no report") || strings.Contains(got, "21577") || strings.Contains(got, "113/113 tokens") {
		t.Fatalf("pre-prompt guard should suppress token-first Compact/Trim values, got:\n%s", got)
	}
}

func TestTokenFirstCompactRetentionUsesAfterOverBeforeTokens(t *testing.T) {
	payload := map[string]any{
		"tokens": map[string]any{
			"last_turn": map[string]any{
				"available": true,
				"total":     float64(1000),
			},
		},
		"compaction": map[string]any{
			"before_tokens": float64(100000),
			"after_tokens":  float64(42000),
			"compacted":     true,
		},
	}
	got := strings.Join(formatDsproxyCompactionLines(payload), "\n")
	if !strings.Contains(got, "42.0%  42k/100k tokens · triggered") {
		t.Fatalf("compact row should display post-compact/raw retention tokens, got:\n%s", got)
	}
}

func TestPolicyDoesNotDisplayAutoCompactMigrationHintInStatus(t *testing.T) {
	payload := map[string]any{
		"context_window": map[string]any{
			"auto_compact_threshold_tokens": float64(750000),
			"auto_compact_policy": map[string]any{
				"available":       true,
				"needs_migration": true,
				"display_label":   "diagnostic label",
				"short_action":    "diagnostic action",
			},
		},
		"runtime_payload_guard": map[string]any{
			"compaction": map[string]any{
				"policy":               "adaptive",
				"keep_recent_messages": float64(24),
			},
		},
	}
	got := formatDsproxyCompactionPolicySummaryLine(payload)
	want := "Policy   adaptive · trigger 750k tokens · keep ⤒24 msgs"
	if got != want {
		t.Fatalf("policy line mismatch:\\n got: %q\\nwant: %q", got, want)
	}
	if strings.Contains(got, "diagnostic") {
		t.Fatalf("policy line must not surface dsproxy repair diagnostics in normal /status: %q", got)
	}
}
