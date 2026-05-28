package messaging

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var supportedAttachmentExts = []string{
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	".zip", ".txt", ".csv",
	".png", ".jpg", ".jpeg", ".gif", ".webp",
	".mp4", ".mov",
}

type weclawArtifactBlock struct {
	start int
	end   int
	path  string
	send  bool
}

func defaultAttachmentWorkspace() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(os.TempDir())
	}
	return filepath.Join(home, ".weclaw", "workspace")
}

func extractLocalAttachmentPaths(text string) []string {
	var paths []string
	seen := make(map[string]struct{})

	addPath := func(candidate string) {
		candidate = normalizeAttachmentCandidate(candidate)
		if candidate == "" || !filepath.IsAbs(candidate) {
			return
		}
		if !isSupportedAttachmentPath(candidate) {
			return
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		paths = append(paths, candidate)
	}

	for _, block := range parseWeClawArtifactBlocks(text) {
		if block.send {
			addPath(block.path)
		}
	}

	for _, line := range strings.Split(text, "\n") {
		candidate := strings.TrimSpace(line)
		if isWeClawArtifactControlLine(candidate) {
			continue
		}
		addPath(candidate)
	}

	return paths
}

func isAllowedAttachmentPath(path string, allowedRoots []string) bool {
	cleanPath, err := canonicalizePath(path, true)
	if err != nil {
		return false
	}
	for _, root := range allowedRoots {
		if root == "" {
			continue
		}
		cleanRoot, err := canonicalizePath(root, false)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(cleanRoot, cleanPath)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return true
		}
	}
	return false
}

func rewriteReplyWithAttachmentResults(reply string, sentPaths, failedPaths []string) string {
	sentMap := make(map[string]string, len(sentPaths))
	for _, path := range sentPaths {
		sentMap[path] = "Sent attachment: " + filepath.Base(path)
	}
	failedMap := make(map[string]string, len(failedPaths))
	for _, path := range failedPaths {
		failedMap[path] = "Attachment send failed: " + filepath.Base(path)
	}

	blocks := parseWeClawArtifactBlocks(reply)
	blocksByStart := make(map[int]weclawArtifactBlock, len(blocks))
	for _, block := range blocks {
		blocksByStart[block.start] = block
	}

	lines := strings.Split(reply, "\n")
	var rewrittenLines []string
	emittedFailures := make(map[string]struct{})
	for i := 0; i < len(lines); i++ {
		if block, ok := blocksByStart[i]; ok {
			if replacement, ok := sentMap[block.path]; ok {
				rewrittenLines = append(rewrittenLines, replacement)
			} else if replacement, ok := failedMap[block.path]; ok {
				rewrittenLines = append(rewrittenLines, replacement)
				emittedFailures[block.path] = struct{}{}
			} else {
				rewrittenLines = append(rewrittenLines, lines[block.start:block.end+1]...)
			}
			i = block.end
			continue
		}
		trimmed := strings.TrimSpace(lines[i])
		if replacement, ok := sentMap[trimmed]; ok {
			rewrittenLines = append(rewrittenLines, replacement)
			continue
		}
		rewrittenLines = append(rewrittenLines, lines[i])
	}

	rewritten := strings.Join(rewrittenLines, "\n")
	var failureLines []string
	seenFailures := make(map[string]struct{})
	for _, path := range failedPaths {
		if _, ok := seenFailures[path]; ok {
			continue
		}
		seenFailures[path] = struct{}{}
		if _, ok := emittedFailures[path]; ok {
			continue
		}
		failureLines = append(failureLines, "Attachment send failed: "+filepath.Base(path))
	}
	if len(failureLines) == 0 {
		return rewritten
	}
	if strings.TrimSpace(rewritten) == "" {
		return strings.Join(failureLines, "\n")
	}
	return rewritten + "\n" + strings.Join(failureLines, "\n")
}

func parseWeClawArtifactBlocks(text string) []weclawArtifactBlock {
	lines := strings.Split(text, "\n")
	var blocks []weclawArtifactBlock
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !isWeClawArtifactStart(line) {
			continue
		}
		block := weclawArtifactBlock{start: i, end: i, send: true}
		if inlinePath := strings.TrimSpace(artifactMarkerInlineValue(line)); inlinePath != "" {
			block.path = inlinePath
		}
		for j := i + 1; j < len(lines); j++ {
			inner := strings.TrimSpace(lines[j])
			if isWeClawArtifactEnd(inner) {
				block.end = j
				break
			}
			if inner == "" {
				block.end = j - 1
				break
			}
			if key, value, ok := parseArtifactKeyValue(inner); ok {
				switch key {
				case "path", "file":
					block.path = value
				case "send":
					block.send = artifactSendEnabled(value)
				}
			}
			block.end = j
		}
		block.path = normalizeAttachmentCandidate(block.path)
		blocks = append(blocks, block)
		i = block.end
	}
	return blocks
}

func artifactMarkerInlineValue(line string) string {
	if idx := strings.Index(line, ":"); idx >= 0 {
		return strings.TrimSpace(line[idx+1:])
	}
	return ""
}

func parseArtifactKeyValue(line string) (string, string, bool) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	if line == "" {
		return "", "", false
	}
	idxEq := strings.Index(line, "=")
	idxColon := strings.Index(line, ":")
	idx := -1
	if idxEq >= 0 && idxColon >= 0 {
		idx = min(idxEq, idxColon)
	} else if idxEq >= 0 {
		idx = idxEq
	} else if idxColon >= 0 {
		idx = idxColon
	}
	if idx < 0 {
		return "", "", false
	}
	key := strings.ToLower(strings.TrimSpace(line[:idx]))
	value := strings.TrimSpace(line[idx+1:])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, normalizeAttachmentCandidate(value), true
}

func artifactSendEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "no", "0", "off":
		return false
	default:
		return true
	}
}

func normalizeAttachmentCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, "`\"'")
	if strings.HasPrefix(candidate, "file://") {
		candidate = strings.TrimPrefix(candidate, "file://")
	}
	if candidate == "" {
		return ""
	}
	return filepath.Clean(candidate)
}

func isWeClawArtifactControlLine(line string) bool {
	if isWeClawArtifactStart(line) || isWeClawArtifactEnd(line) {
		return true
	}
	key, _, ok := parseArtifactKeyValue(line)
	if !ok {
		return false
	}
	switch key {
	case "path", "file", "type", "title", "send":
		return true
	default:
		return false
	}
}

func isWeClawArtifactStart(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	return upper == "WECLAW_ARTIFACT" || strings.HasPrefix(upper, "WECLAW_ARTIFACT:")
}

func isWeClawArtifactEnd(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	return upper == "END_WECLAW_ARTIFACT" || upper == "WECLAW_ARTIFACT_END"
}

func isSupportedAttachmentPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(supportedAttachmentExts, ext)
}

func canonicalizePath(path string, mustExist bool) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if realPath, err := filepath.EvalSymlinks(absPath); err == nil {
		return filepath.Clean(realPath), nil
	} else if mustExist {
		return "", err
	}
	return filepath.Clean(absPath), nil
}
