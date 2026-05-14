package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var normalizedInternalTagPattern = regexp.MustCompile(`^p[0-9]+\.[0-9]+\.[0-9]+(?:a[0-9]+)*(?:-[A-Za-z0-9._-]+)?$`)

func TestNormalizedInternalTagPattern(t *testing.T) {
	valid := []string{
		"p0.1.5-topic",
		"p0.1.5a1-topic",
		"p0.1.5a1a3-topic",
		"p0.1.5a33-internal-version-format-policy",
	}
	for _, tag := range valid {
		if !normalizedInternalTagPattern.MatchString(tag) {
			t.Fatalf("expected valid internal tag %q", tag)
		}
	}

	invalid := []string{
		"v0.1p5a33-internal-version-format-policy",
		"v0.1p5-topic",
		"p0.1-topic",
		"p0.1.5a-topic",
		"alpha-work-v0.1p5a33-topic",
	}
	for _, tag := range invalid {
		if normalizedInternalTagPattern.MatchString(tag) {
			t.Fatalf("expected invalid internal tag %q", tag)
		}
	}
}

func TestCurrentInternalTagDocumentationUsesNormalizedFormat(t *testing.T) {
	root := repoRootForInternalTagFormatTest(t)
	checks := []struct {
		path   string
		prefix string
	}{
		{"docs/developer-handbook.md", "- Current internal development tag: `"},
		{"docs/developer-handbook.zh-CN.md", "- 当前内部开发标签：`"},
	}
	for _, tc := range checks {
		text := readRepoFileForInternalTagFormatTest(t, root, tc.path)
		tag := extractBacktickValueAfterPrefixForInternalTagFormatTest(t, text, tc.prefix)
		if !normalizedInternalTagPattern.MatchString(tag) {
			t.Fatalf("%s current internal tag is not normalized: %q", tc.path, tag)
		}
		if strings.HasPrefix(tag, "v") {
			t.Fatalf("%s current internal tag must not start with v: %q", tc.path, tag)
		}
	}
}

func TestFutureFacingDocsDoNotUseLegacyInternalTagExamples(t *testing.T) {
	root := repoRootForInternalTagFormatTest(t)
	checks := []string{
		"README.md",
		".github/workflows/release.yml",
	}
	for _, path := range checks {
		text := readRepoFileForInternalTagFormatTest(t, root, path)
		if strings.Contains(text, "v0.1p") {
			t.Fatalf("%s contains legacy internal tag example v0.1p", path)
		}
		if strings.Contains(text, "alpha-work-v0.1p") {
			t.Fatalf("%s contains legacy alpha-work internal tag example", path)
		}
	}
}

func repoRootForInternalTagFormatTest(t *testing.T) string {
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

func readRepoFileForInternalTagFormatTest(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func extractBacktickValueAfterPrefixForInternalTagFormatTest(t *testing.T, text, prefix string) string {
	t.Helper()
	idx := strings.Index(text, prefix)
	if idx < 0 {
		t.Fatalf("missing prefix %q", prefix)
	}
	rest := text[idx+len(prefix):]
	end := strings.Index(rest, "`")
	if end < 0 {
		t.Fatalf("missing closing backtick after prefix %q", prefix)
	}
	return rest[:end]
}
