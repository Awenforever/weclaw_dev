package messaging

import (
	"strings"
	"testing"
)

func TestWrapMarkdownFenceUsesLongerOuterFence(t *testing.T) {
	content := "```go\nfmt.Println(\"inner\")\n```"
	got := wrapMarkdownFence(content, "markdown")
	if !strings.HasPrefix(got, "````markdown\n```go\n") {
		t.Fatalf("outer fence did not grow beyond nested triple fence:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n````") {
		t.Fatalf("outer fence did not close with four backticks:\n%s", got)
	}
}

func TestVisualCommandFenceUsesDynamicOuterFence(t *testing.T) {
	got := strings.Join(visualCommandFence("", "```go", "fmt.Println(\"inner\")", "```"), "\n")
	if !strings.HasPrefix(got, "````text\n```go\n") {
		t.Fatalf("visualCommandFence did not use four-backtick outer fence:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n````") {
		t.Fatalf("visualCommandFence did not close with matching outer fence:\n%s", got)
	}
}

func TestMarkdownCommandLinesDoesNotCloseOuterFenceOnNestedShortFence(t *testing.T) {
	got := strings.Join(markdownCommandLines("````text", "```go", "    fmt.Println(\"inner\")", "```", "    still indented", "````"), "\n")
	if !strings.Contains(got, "\n    still indented\n") {
		t.Fatalf("nested short fence broke outer fence indentation:\n%s", got)
	}
	if strings.Count(got, "````") != 2 {
		t.Fatalf("outer fence count mismatch:\n%s", got)
	}
}

func TestEnsureBalancedCodeFenceUsesOpeningFenceLength(t *testing.T) {
	input := "````markdown\n```go\nfmt.Println(\"inner\")\n```\n"
	got := ensureBalancedCodeFence(input)
	if !strings.HasSuffix(got, "\n````") {
		t.Fatalf("expected unclosed four-backtick fence to close with four backticks:\n%s", got)
	}
}

func TestClawBotMarkdownReplyChunksPreservesNestedShortFenceInsideLongFence(t *testing.T) {
	input := "Intro.\n\n````markdown\n```go\nfmt.Println(\"inner\")\n```\n````\n\nAfter."
	want := []string{"Intro.", "````markdown\n```go\nfmt.Println(\"inner\")\n```\n````", "After."}
	assertChunks(t, ClawBotMarkdownReplyChunks(input), want)
}

func TestSlashInlineCodeUsesLongerDelimiterWhenValueContainsBackticks(t *testing.T) {
	got := slashInlineCode("use `pip install`")
	if !strings.HasPrefix(got, "``") || !strings.HasSuffix(got, "``") {
		t.Fatalf("inline code did not use double-backtick delimiter: %q", got)
	}
}
