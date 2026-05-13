package runtime_state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMostRecentSessionForDefaultProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SetDefaultProfile("deepseek-thinking"); err != nil {
		t.Fatalf("SetDefaultProfile returned error: %v", err)
	}
	if err := UpsertSession("deepseek", "user-1", "thread-deepseek"); err != nil {
		t.Fatalf("UpsertSession deepseek returned error: %v", err)
	}
	if err := UpsertSession("deepseek-thinking", "user-2", "thread-thinking"); err != nil {
		t.Fatalf("UpsertSession thinking returned error: %v", err)
	}

	profile, sessionID, ok := MostRecentSessionForDefaultProfile()
	if !ok {
		t.Fatal("MostRecentSessionForDefaultProfile ok = false")
	}
	if profile != "deepseek-thinking" || sessionID != "thread-thinking" {
		t.Fatalf("got profile=%q session=%q, want deepseek-thinking/thread-thinking", profile, sessionID)
	}
}

func TestMostRecentLogThreadForPIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".weclaw"), 0o700); err != nil {
		t.Fatalf("create weclaw dir: %v", err)
	}
	logText := "" +
		"2026/05/12 12:00:00 [acp] new thread created (pid=111, thread=thread-old, conversation=user-1)\n" +
		"2026/05/12 12:01:00 [acp] reusing thread (pid=222, thread=thread-target, conversation=user-2)\n" +
		"2026/05/12 12:02:00 [acp] reusing thread (pid=333, thread=thread-other, conversation=user-3)\n"
	if err := os.WriteFile(LogPath(), []byte(logText), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	hint, ok := MostRecentLogThreadForPIDs("deepseek-thinking", []int{222})
	if !ok {
		t.Fatal("MostRecentLogThreadForPIDs ok = false")
	}
	if hint.Profile != "deepseek-thinking" || hint.UserID != "user-2" || hint.SessionID != "thread-target" {
		t.Fatalf("hint = %#v, want deepseek-thinking/user-2/thread-target", hint)
	}
}

func TestMostRecentSessionForProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := UpsertSession("deepseek", "user-1", "thread-deepseek"); err != nil {
		t.Fatalf("UpsertSession deepseek returned error: %v", err)
	}
	if err := UpsertSession("deepseek-thinking", "user-2", "thread-thinking"); err != nil {
		t.Fatalf("UpsertSession thinking returned error: %v", err)
	}

	hint, ok := MostRecentSessionForProfile("deepseek-thinking")
	if !ok {
		t.Fatal("MostRecentSessionForProfile ok = false")
	}
	if hint.Profile != "deepseek-thinking" || hint.UserID != "user-2" || hint.SessionID != "thread-thinking" {
		t.Fatalf("hint = %#v, want deepseek-thinking/user-2/thread-thinking", hint)
	}
}
