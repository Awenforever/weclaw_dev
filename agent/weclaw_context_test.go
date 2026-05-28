package agent

import (
	"strings"
	"testing"
)

func TestMergeWeClawSystemPromptIncludesWechatAttachmentContract(t *testing.T) {
	got := MergeWeClawSystemPrompt("")
	for _, want := range []string{
		"WeClaw runtime context:",
		"active WeChat chat",
		"user-facing artifacts",
		"WECLAW_ARTIFACT",
		"standalone absolute local path",
		"png",
		"docx",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("merged prompt missing %q in:\n%s", want, got)
		}
	}
}

func TestMergeWeClawSystemPromptPreservesConfiguredPrompt(t *testing.T) {
	got := MergeWeClawSystemPrompt("Prefer concise replies.")
	if !strings.Contains(got, "WeClaw runtime context:") {
		t.Fatalf("merged prompt missing WeClaw context: %q", got)
	}
	if !strings.Contains(got, "Prefer concise replies.") {
		t.Fatalf("merged prompt missing configured prompt: %q", got)
	}
}

func TestComposeUserMessageWithSystemPromptKeepsUserMessage(t *testing.T) {
	got := ComposeUserMessageWithSystemPrompt("WeClaw runtime context: use attachments.", "通过微信发给我")
	if !strings.Contains(got, "WeClaw runtime context: use attachments.") {
		t.Fatalf("composed message missing context: %q", got)
	}
	if !strings.Contains(got, "通过微信发给我") {
		t.Fatalf("composed message missing user text: %q", got)
	}
	if !strings.Contains(got, "User message from the current WeChat chat") {
		t.Fatalf("composed message missing delimiter: %q", got)
	}
}

func TestComposeUserMessageWithSystemPromptLeavesRawMessageWhenEmpty(t *testing.T) {
	got := ComposeUserMessageWithSystemPrompt("", "hello")
	if got != "hello" {
		t.Fatalf("ComposeUserMessageWithSystemPrompt empty prompt = %q, want raw message", got)
	}
}
