package messaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureOutboundMarkdownForDebugWritesFinalOutText(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(outboundMarkdownCaptureDirEnv, dir)

	text := "````markdown\n```python\nprint(\"ok\")\n```\n````"
	captureOutboundMarkdownForDebug("user-1", "client/with bad chars", text)

	matches, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		t.Fatalf("glob capture dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("capture file count = %d, want 1, matches=%v", len(matches), matches)
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read capture file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"===== OUTBOUND MARKDOWN CAPTURE =====",
		"to_user=user-1",
		"client_id=client/with bad chars",
		"===== OUTBOUND MARKDOWN =====",
		text,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("capture file missing %q:\n%s", want, got)
		}
	}
}

func TestCaptureOutboundMarkdownForDebugIgnoresRelativeDir(t *testing.T) {
	t.Setenv(outboundMarkdownCaptureDirEnv, "relative-dir")
	captureOutboundMarkdownForDebug("user-1", "client-1", "hello")
	if _, err := os.Stat("relative-dir"); err == nil {
		t.Fatalf("relative capture dir should not be created")
	}
}

func TestSafeCaptureFilename(t *testing.T) {
	got := safeCaptureFilename("client/with bad chars")
	if got != "client_with_bad_chars" {
		t.Fatalf("safeCaptureFilename() = %q", got)
	}
}
