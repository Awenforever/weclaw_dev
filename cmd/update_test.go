package cmd

import (
	"github.com/fastclaw-ai/weclaw/runtime_state"
	"os"
	"path/filepath"
	"strings"
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
		{name: "local test build", current: "v0.1p4-local-test", latest: "v0.1.2-alpha", want: false},
		{name: "internal tag build", current: "v0.1p4-session-resume-status-docs", latest: "v0.1.2-alpha", want: false},
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

func TestIsRetriableHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{name: "ok", status: 200, want: false},
		{name: "not found", status: 404, want: false},
		{name: "rate limited", status: 429, want: true},
		{name: "server error", status: 500, want: true},
		{name: "bad gateway", status: 502, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetriableHTTPStatus(tc.status); got != tc.want {
				t.Fatalf("isRetriableHTTPStatus(%d) = %v, want %v", tc.status, got, tc.want)
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

func TestStageReplacementBinaryCreatesFileInTargetDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src-weclaw")
	dst := filepath.Join(dir, "weclaw")
	if err := os.WriteFile(src, []byte("new-binary"), 0o755); err != nil {
		t.Fatalf("write src: %v", err)
	}
	staged, err := stageReplacementBinary(src, dst)
	if err != nil {
		t.Fatalf("stageReplacementBinary returned error: %v", err)
	}
	defer os.Remove(staged)

	if filepath.Dir(staged) != dir {
		t.Fatalf("staged dir = %q, want %q", filepath.Dir(staged), dir)
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("read staged: %v", err)
	}
	if string(data) != "new-binary" {
		t.Fatalf("staged data = %q, want new-binary", data)
	}
}

func TestUpgradeRestartResumeSelectionUsesRuntimeState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := runtime_state.SetDefaultProfile("deepseek-thinking"); err != nil {
		t.Fatalf("SetDefaultProfile returned error: %v", err)
	}
	if err := runtime_state.UpsertSession("deepseek-thinking", "user-1", "thread-upgrade-resume"); err != nil {
		t.Fatalf("UpsertSession returned error: %v", err)
	}

	profile, sessionID := upgradeRestartResumeSelection(nil)
	if profile != "deepseek-thinking" || sessionID != "thread-upgrade-resume" {
		t.Fatalf("upgradeRestartResumeSelection() = %q/%q, want deepseek-thinking/thread-upgrade-resume", profile, sessionID)
	}
}

func TestIsAlphaReleaseVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "v0.1.4-alpha", want: true},
		{version: "v1.2.3-alpha", want: true},
		{version: "v0.1.4", want: false},
		{version: "v0.1.4-beta", want: false},
		{version: "v0.1p5a9-upgrade-resume-status-clarity", want: false},
		{version: "0.1.4-alpha", want: false},
	}
	for _, tc := range tests {
		if got := isAlphaReleaseVersion(tc.version); got != tc.want {
			t.Fatalf("isAlphaReleaseVersion(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}

func TestUpgradeCommandExposesAlphaFlag(t *testing.T) {
	if upgradeCmd.Flags().Lookup("alpha") == nil {
		t.Fatal("upgrade --alpha flag is missing")
	}
}

func TestVersionOutputIncludesPublicAndInternalMetadata(t *testing.T) {
	oldVersion := Version
	oldPublicCommit := PublicCommit
	oldInternalVersion := InternalVersion
	oldInternalCommit := InternalCommit
	t.Cleanup(func() {
		Version = oldVersion
		PublicCommit = oldPublicCommit
		InternalVersion = oldInternalVersion
		InternalCommit = oldInternalCommit
	})

	Version = "v0.1.4-alpha"
	PublicCommit = "3460e0741f29f2e13f1451f995a6f8a000a89caa"
	InternalVersion = "v0.1p5a11-version-metadata-dual-output"
	InternalCommit = "3460e0741f29f2e13f1451f995a6f8a000a89caa"

	got := versionOutput("linux", "amd64")
	for _, want := range []string{
		"weclaw public version: v0.1.4-alpha | 3460e07 (linux/amd64)",
		"weclaw internal version: v0.1p5a11-version-metadata-dual-output | 3460e07 (linux/amd64)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("versionOutput() = %q, want %q", got, want)
		}
	}
}

func TestReleaseMetadataLDFlagsDocumentRequiredFields(t *testing.T) {
	got := strings.Join(releaseMetadataLDFlags(), "\n")
	for _, want := range []string{
		"cmd.Version=<public-release-tag>",
		"cmd.PublicCommit=<public-release-commit>",
		"cmd.InternalVersion=<internal-p-tag>",
		"cmd.InternalCommit=<internal-p-commit>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("releaseMetadataLDFlags() = %q, want %q", got, want)
		}
	}
}
