package messaging

import (
	"os"
	"strings"
	"testing"

	"github.com/fastclaw-ai/weclaw/agent"
)

func TestSlashHelpIsEnglishCompactAndHidesInfo(t *testing.T) {
	got := buildHelpText()
	for _, forbidden := range []string{"常用命令", "查看当前", "推理强度", "："} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("help text contains non-English or Chinese punctuation token %q in:\n%s", forbidden, got)
		}
	}
	if strings.Contains(got, "`/info`:") || strings.Contains(got, "status   /status  /now  /cancel  /info") {
		t.Fatalf("/info should not be listed as a primary command:\n%s", got)
	}
	for _, want := range []string{"QUICK MAP", "/status", "/now", "/model", "/effort", "/balance", "Unknown slash commands"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help text missing %q in:\n%s", want, got)
		}
	}
}

func TestBalanceReplyUsesSingleTableAndHidesEmptyGranted(t *testing.T) {
	got := formatBalanceReply(`{"status":"ok","balance":{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"12.34","granted_balance":"99.99","topped_up_balance":"88.88"}]}}`)
	for _, forbidden := range []string{"| Currency |", "Granted", "Topped-up", "- CNY total:", "- CNY granted:", "- CNY topped-up:"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("balance reply contains table or hidden column token %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"## 💰 Balance", "Currency  Total", "CNY       12.34"} {
		if !strings.Contains(got, want) {
			t.Fatalf("balance reply missing %q in:\n%s", want, got)
		}
	}
}

func TestUnknownSlashCommandCardIsCompactAndSuggestsCancel(t *testing.T) {
	got := unknownSlashCommandCard("/cancle")
	for _, want := range []string{"## ⚠️ Unknown slash command", "Not sent to agent", "`/cancle`", "`/cancel`"} {
		if !strings.Contains(got, want) {
			t.Fatalf("unknown command reply missing %q in:\n%s", want, got)
		}
	}
}

func TestCompactContextPanelDoesNotExposeSourceOrMetricTable(t *testing.T) {
	got := strings.Join(buildCompactStatusPanel(nil, "user-1", 1000000, "thinking", "127.0.0.1:8001", "reachable", "￥12.34"), "\n")
	for _, forbidden := range []string{"source:", "Metric", "Limit", "Used", "Left", "tools", "other", "Granted", "Paths"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("compact status panel contains old verbose token %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"Context", "Tokens", "Cost     session n/a  last n/a", "￥12.34", "Proxy    thinking · 127.0.0.1:8001 · reachable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("compact status panel missing %q in:\n%s", want, got)
		}
	}
}

func TestVisualCommandFenceAllowsNoTitle(t *testing.T) {
	got := strings.Join(visualCommandFence("", "Context  [░░░░]  0.0%  0/1M"), "\n")
	if strings.Contains(got, "```text\n\n") {
		t.Fatalf("empty fence title produced a blank title line:\n%s", got)
	}
	if !strings.Contains(got, "```text\nContext") {
		t.Fatalf("fence did not start directly with content:\n%s", got)
	}
}

func TestSlashCommandOutputPreviewSnapshot(t *testing.T) {
	out := os.Getenv("WECLAW_SLASH_PREVIEW_OUT")
	if out == "" {
		t.Skip("WECLAW_SLASH_PREVIEW_OUT not set")
	}

	preview := strings.Join([]string{
		"# p0.1.5a45 slash command preview",
		"",
		"## /help",
		buildHelpText(),
		"",
		"## /status panel core",
		strings.Join(append([]string{"## 🧩 Status", "- **Profile:** `deepseek-thinking` `ACP`", "- **Model:** `deepseek-v4-flash` `high`", "- **Session:** `thread-preview`", ""}, visualCommandFence("", buildCompactStatusPanel(nil, "user-1", 1000000, "thinking", "127.0.0.1:8001", "reachable", "￥12.34")...)...), "\n"),
		"",
		"## /balance",
		formatBalanceReply(`{"status":"ok","balance":{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"12.34","granted_balance":"","topped_up_balance":""}]}}`),
		"",
		"## unknown",
		unknownSlashCommandCard("/cancle"),
	}, "\n")

	if err := os.WriteFile(out, []byte(preview), 0o644); err != nil {
		t.Fatalf("write preview: %v", err)
	}
}

func TestMarkdownCommandLinesPreservesFenceIndentation(t *testing.T) {
	got := strings.Join(markdownCommandLines("```text", "Paths    cfg ~/.weclaw/config.json", "         log ~/.weclaw/weclaw.log", "```"), "\n")
	if !strings.Contains(got, "\n         log ~/.weclaw/weclaw.log\n") {
		t.Fatalf("fenced indentation was not preserved:\n%s", got)
	}
}

func TestCompactStatusPanelUsesShorterProgressBar(t *testing.T) {
	got := strings.Join(buildCompactStatusPanel(nil, "user-1", 1000000, "", "", "", "balance n/a"), "\n")
	start := strings.Index(got, "[")
	end := strings.Index(got, "]")
	if start < 0 || end <= start {
		t.Fatalf("progress bar not found in:\n%s", got)
	}
	bar := got[start+1 : end]
	if gotLen := len([]rune(bar)); gotLen != 20 {
		t.Fatalf("progress bar length = %d, want 20 in:\n%s", gotLen, got)
	}
}

func TestWorkspaceReplyDoesNotDuplicateCwdRows(t *testing.T) {
	h := NewHandler(nil, nil)
	h.SetDefaultAgent("deepseek", &runtimeControlTestAgent{info: agent.AgentInfo{Name: "deepseek", Type: "acp"}})

	got := h.handleCwd("/cwd")
	for _, forbidden := range []string{"| Field | Value |", "- cwd:", "- agent:"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("workspace reply contains duplicate token %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"## 📁 Workspace", "Agent:** `deepseek`", "Cwd:** check agent config"} {
		if !strings.Contains(got, want) {
			t.Fatalf("workspace reply missing %q in:\n%s", want, got)
		}
	}
}

func TestCompactBalanceSummaryFromText(t *testing.T) {
	got := compactBalanceSummaryFromText(`{"status":"ok","balance":{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"12.34","granted_balance":"99.99","topped_up_balance":"88.88"}]}}`)
	if got != "￥12.34" {
		t.Fatalf("compactBalanceSummaryFromText() = %q, want ￥12.34", got)
	}
}
