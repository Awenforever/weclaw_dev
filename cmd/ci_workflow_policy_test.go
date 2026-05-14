package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIWorkflowDoesNotPublishPreReleases(t *testing.T) {
	root := repoRootForCIWorkflowPolicyTest(t)
	ci := readRepoFileForCIWorkflowPolicyTest(t, root, ".github/workflows/ci.yml")

	forbidden := []string{
		"softprops/action-gh-release",
		"gh release delete",
		"tag=beta-latest",
		"tag=alpha-",
		"alpha-${SAFE_BRANCH}",
		"target_commitish:",
		"prerelease: true",
		"contents: write",
	}
	for _, token := range forbidden {
		if strings.Contains(ci, token) {
			t.Fatalf("CI workflow must not contain pre-release publishing token %q", token)
		}
	}

	required := []string{
		"branches: ['**']",
		"tags-ignore: ['**']",
		"contents: read",
		"actions/upload-artifact@v4",
	}
	for _, token := range required {
		if !strings.Contains(ci, token) {
			t.Fatalf("CI workflow missing expected token %q", token)
		}
	}
}

func TestManualReleaseWorkflowRemainsExplicit(t *testing.T) {
	root := repoRootForCIWorkflowPolicyTest(t)
	release := readRepoFileForCIWorkflowPolicyTest(t, root, ".github/workflows/release.yml")

	required := []string{
		"workflow_dispatch:",
		"public_tag:",
		"internal_tag:",
		"softprops/action-gh-release@v2",
		"tag_name: ${{ inputs.public_tag }}",
		"prerelease: ${{ inputs.prerelease }}",
	}
	for _, token := range required {
		if !strings.Contains(release, token) {
			t.Fatalf("manual release workflow missing expected token %q", token)
		}
	}
}

func repoRootForCIWorkflowPolicyTest(t *testing.T) string {
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

func readRepoFileForCIWorkflowPolicyTest(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
