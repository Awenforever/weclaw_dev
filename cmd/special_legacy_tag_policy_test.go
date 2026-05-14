package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecialLegacyTagsAreArchivedInDevelopmentLog(t *testing.T) {
	root := repoRootForSpecialLegacyTagPolicyTest(t)
	devlog := readRepoFileForSpecialLegacyTagPolicyTest(t, root, "docs/development-log.md")

	required := []string{
		"`v0.1d-codex-model-provider-thread-fix` -> `p0.1.0a1-codex-model-provider-thread-fix`",
		"`v0.1e-slash-output-polish` -> `p0.1.0a2-slash-output-polish`",
		"`v0.1f-acp-raw-stdout-log` -> `p0.1.0a3-acp-raw-stdout-log`",
		"`v0.1f1-handoff-notes` -> `p0.1.0a4-handoff-notes`",
	}
	for _, token := range required {
		if !strings.Contains(devlog, token) {
			t.Fatalf("development log missing special legacy archive mapping %q", token)
		}
	}
}

func TestFutureFacingDocsDoNotUseSpecialLegacyTags(t *testing.T) {
	root := repoRootForSpecialLegacyTagPolicyTest(t)
	files := []string{
		"README.md",
		"README_CN.md",
		".github/workflows/ci.yml",
		".github/workflows/release.yml",
		"docs/developer-handbook.md",
		"docs/developer-handbook.zh-CN.md",
	}
	forbidden := []string{
		"v0.1d-codex-model-provider-thread-fix",
		"v0.1e-slash-output-polish",
		"v0.1f-acp-raw-stdout-log",
		"v0.1f1-handoff-notes",
	}
	for _, rel := range files {
		text := readRepoFileForSpecialLegacyTagPolicyTest(t, root, rel)
		for _, token := range forbidden {
			if strings.Contains(text, token) {
				t.Fatalf("%s contains future-facing special legacy tag %q", rel, token)
			}
		}
	}
}

func repoRootForSpecialLegacyTagPolicyTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

func readRepoFileForSpecialLegacyTagPolicyTest(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
