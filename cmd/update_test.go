package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseAssetName(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{goos: "linux", goarch: "amd64", want: "weclaw_linux_amd64"},
		{goos: "darwin", goarch: "arm64", want: "weclaw_darwin_arm64"},
		{goos: "windows", goarch: "amd64", want: "weclaw_windows_amd64.exe"},
	}

	for _, tc := range tests {
		if got := releaseAssetName(tc.goos, tc.goarch); got != tc.want {
			t.Fatalf("releaseAssetName(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestShouldOfferUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "same release", current: "v0.1.1-alpha", latest: "v0.1.1-alpha", want: false},
		{name: "new release", current: "v0.1.1-alpha", latest: "v0.1.2-alpha", want: true},
		{name: "dev build", current: "dev", latest: "v0.1.2-alpha", want: false},
		{name: "empty latest", current: "v0.1.1-alpha", latest: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldOfferUpdate(tc.current, tc.latest); got != tc.want {
				t.Fatalf("shouldOfferUpdate(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}

func TestUpdateCheckDue(t *testing.T) {
	now := time.Date(2026, 5, 11, 21, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		state updateCheckState
		want  bool
	}{
		{name: "never checked", state: updateCheckState{}, want: true},
		{name: "recent check", state: updateCheckState{LastCheckedAt: now.Add(-time.Hour).Format(time.RFC3339)}, want: false},
		{name: "old check", state: updateCheckState{LastCheckedAt: now.Add(-25 * time.Hour).Format(time.RFC3339)}, want: true},
		{name: "bad timestamp", state: updateCheckState{LastCheckedAt: "bad"}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := updateCheckDue(tc.state, now); got != tc.want {
				t.Fatalf("updateCheckDue(%+v) = %v, want %v", tc.state, got, tc.want)
			}
		})
	}
}

func TestManagedProcessPIDsForExecutableOnlyMatchesSameExecutable(t *testing.T) {
	target := t.TempDir() + "/weclaw"
	other := t.TempDir() + "/weclaw"

	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write target executable: %v", err)
	}
	if err := os.WriteFile(other, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write other executable: %v", err)
	}

	live := []managedProcess{
		{PID: 101, Args: []string{target, "start", "-f"}},
		{PID: 202, Args: []string{other, "start", "-f"}},
		{PID: 303, Args: []string{"", "start", "-f"}},
	}

	got := managedProcessPIDsForExecutable(live, target)
	if len(got) != 1 || got[0] != 101 {
		t.Fatalf("managedProcessPIDsForExecutable() = %v, want [101]", got)
	}
}

func TestSameExecutablePathRejectsDifferentPaths(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "weclaw")
	other := filepath.Join(dir, "other-weclaw")

	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write target executable: %v", err)
	}
	if err := os.WriteFile(other, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write other executable: %v", err)
	}

	if !sameExecutablePath(target, target) {
		t.Fatal("sameExecutablePath should accept the same path")
	}
	if sameExecutablePath(other, target) {
		t.Fatal("sameExecutablePath should reject a different executable")
	}
}
