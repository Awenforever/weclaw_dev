package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastclaw-ai/weclaw/runtime_state"
)

func TestParseStartSelection(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProfile string
		wantResume  string
		wantErr     bool
	}{
		{name: "empty", args: nil, wantProfile: ""},
		{name: "deepseek", args: []string{"deepseek"}, wantProfile: "deepseek"},
		{name: "deepseek thinking", args: []string{"deepseek-thinking"}, wantProfile: "deepseek-thinking"},
		{name: "deepseek resume", args: []string{"deepseek", "resume", "thread-123"}, wantProfile: "deepseek", wantResume: "thread-123"},
		{name: "deepseek resume latest", args: []string{"deepseek", "resume"}, wantProfile: "deepseek"},
		{name: "deepseek thinking resume", args: []string{"deepseek-thinking", "resume", "thread-456"}, wantProfile: "deepseek-thinking", wantResume: "thread-456"},
		{name: "deepseek thinking resume latest", args: []string{"deepseek-thinking", "resume"}, wantProfile: "deepseek-thinking"},
		{name: "reject thinking alias", args: []string{"thinking"}, wantErr: true},
		{name: "reject non thinking alias", args: []string{"non-thinking"}, wantErr: true},
		{name: "reject resume without profile", args: []string{"resume", "thread-123"}, wantErr: true},
		{name: "reject unsupported action", args: []string{"deepseek", "continue", "thread-123"}, wantErr: true},
		{name: "reject multiple args", args: []string{"deepseek", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStartSelection(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseStartSelection(%v) returned nil error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStartSelection(%v) error = %v", tt.args, err)
			}
			if got.Profile != tt.wantProfile || got.ResumeID != tt.wantResume {
				t.Fatalf("parseStartSelection(%v) = %#v, want profile=%q resume=%q", tt.args, got, tt.wantProfile, tt.wantResume)
			}
		})
	}
}

func TestParseStartProfileCompatibility(t *testing.T) {
	got, err := parseStartProfile([]string{"deepseek-thinking"})
	if err != nil {
		t.Fatalf("parseStartProfile returned error: %v", err)
	}
	if got != "deepseek-thinking" {
		t.Fatalf("parseStartProfile = %q, want deepseek-thinking", got)
	}
}

func TestResolveStartResumeIDUsesMostRecentACPThreadForProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := runtime_state.UpsertSession("deepseek", "user-1", "thread-deepseek"); err != nil {
		t.Fatalf("UpsertSession deepseek returned error: %v", err)
	}
	if err := runtime_state.UpsertSession("deepseek-thinking", "user-2", "thread-thinking"); err != nil {
		t.Fatalf("UpsertSession thinking returned error: %v", err)
	}

	got, err := resolveStartResumeID("deepseek-thinking", "", true)
	if err != nil {
		t.Fatalf("resolveStartResumeID returned error: %v", err)
	}
	if got != "thread-thinking" {
		t.Fatalf("resolveStartResumeID = %q, want thread-thinking", got)
	}
}

func TestResolveStartResumeIDErrorsWhenNoRecentACPThread(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveStartResumeID("deepseek-thinking", "", true)
	if err == nil {
		t.Fatalf("resolveStartResumeID returned nil error with got=%q", got)
	}
	if !strings.Contains(err.Error(), "no recent ACP session") {
		t.Fatalf("error = %q, want no recent ACP session", err)
	}
}

func TestTrimBackgroundLogKeepsNewestLinesInOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weclaw.log")
	oldLine := "2026/05/13 10:00:00 old " + strings.Repeat("x", 80) + "\n"
	midLine := "2026/05/13 10:01:00 mid " + strings.Repeat("y", 80) + "\n"
	newLine1 := "2026/05/13 10:02:00 new-1 " + strings.Repeat("z", 80) + "\n"
	newLine2 := "2026/05/13 10:03:00 new-2 " + strings.Repeat("q", 80) + "\n"
	if err := os.WriteFile(path, []byte(oldLine+midLine+newLine1+newLine2), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	const maxLogBytes int64 = 380
	if err := trimBackgroundLog(path, maxLogBytes); err != nil {
		t.Fatalf("trimBackgroundLog returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	text := string(data)
	if int64(len(data)) > maxLogBytes {
		t.Fatalf("trimmed log size = %d, want <= %d", len(data), maxLogBytes)
	}
	if !strings.Contains(text, "log truncated") {
		t.Fatalf("trimmed log missing truncation marker: %q", text)
	}
	if strings.Contains(text, "old ") {
		t.Fatalf("trimmed log retained oldest line: %q", text)
	}
	idx1 := strings.Index(text, "new-1")
	idx2 := strings.Index(text, "new-2")
	if idx1 < 0 || idx2 < 0 || idx1 >= idx2 {
		t.Fatalf("newest log lines not retained in order: %q", text)
	}
}
