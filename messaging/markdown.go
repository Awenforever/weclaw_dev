package messaging

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// Code blocks: strip fences, keep code content for internal plain-text classification.
	reCodeBlock = regexp.MustCompile("(?s)```[^\n]*\n?(.*?)```")
	// Inline code: strip backticks, keep content for internal plain-text classification.
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	// Images: remove entirely for internal plain-text classification.
	reImage = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	// Links: keep display text only for internal plain-text classification.
	reLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	// Table separator rows: remove for internal plain-text classification.
	reTableSep = regexp.MustCompile(`(?m)^\|[\s:|\-]+\|$`)
	// Table rows: convert pipe-delimited to space-delimited for internal plain-text classification.
	reTableRow = regexp.MustCompile(`(?m)^\|(.+)\|$`)
	// Headers: remove # prefix for internal plain-text classification.
	reHeader = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	// Bold: **text** or __text__
	reBold = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	// Italic: *text* or _text_
	reItalic = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)|(?:^|[^_])_([^_]+)_(?:[^_]|$)`)
	// Strikethrough: ~~text~~
	reStrike = regexp.MustCompile(`~~(.+?)~~`)
	// Blockquote: > prefix
	reBlockquote = regexp.MustCompile(`(?m)^>\s?`)
	// Horizontal rule
	reHR = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	// Unordered list markers: -, *, +
	reUL = regexp.MustCompile(`(?m)^(\s*)[-*+]\s+`)
	// Ordered list markers: 1. or 1)
	reOL = regexp.MustCompile(`(?m)^(\s*)(\d+)[.)]\s+`)
	// Optional local output markers after an EOF marker.
	reMarkerOnlyTail = regexp.MustCompile(`^(?:\s|\[NEW LINE\]|\[PARAGRAPH\]|\[ITEM\]|\[EOF\])*$`)
)

// MarkdownForClawBot preserves Markdown that ClawBot can render while normalizing
// WeClaw-local stream markers and bullet notation into Markdown-friendly text.
// It is intentionally not a Markdown-to-plain-text downgrade.
func MarkdownForClawBot(text string) string {
	result := normalizeLineEndings(text)
	result = applyClawBotMarkdownMarkers(result)
	result = normalizeEnglishQuoteCharacters(result)
	result = normalizeClawBotMarkdownListMarkers(result)
	result = cleanupMarkdownSpacing(result)
	result = ensureBalancedCodeFence(result)
	return strings.TrimSpace(result)
}

// ClawBotMarkdownReplyChunks converts a final reply to Markdown-first chunks.
// Chunks preserve code fences, tables, links, emphasis, blockquotes and headings.
func ClawBotMarkdownReplyChunks(markdown string) []string {
	rich := MarkdownForClawBot(markdown)
	if rich == "" {
		return nil
	}

	blocks := splitMarkdownBlocks(rich)
	chunks := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if len(chunks) > 0 && strings.HasSuffix(strings.TrimSpace(chunks[len(chunks)-1]), ":") && isMarkdownListBlock(block) {
			chunks[len(chunks)-1] += "\n" + block
			continue
		}
		chunks = append(chunks, block)
	}
	return chunks
}

// MarkdownToPlainText converts markdown to readable plain text for internal
// classification, progress previews and conservative fallback logic.
func MarkdownToPlainText(text string) string {
	result := normalizeLineEndings(text)
	result = applyPlainTextMarkers(result)
	result = normalizeEnglishQuoteCharacters(result)

	// Code blocks: strip fences, keep code content
	result = reCodeBlock.ReplaceAllStringFunc(result, func(match string) string {
		parts := reCodeBlock.FindStringSubmatch(match)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
		return match
	})

	// Images: remove entirely
	result = reImage.ReplaceAllString(result, "")

	// Links: keep display text only
	result = reLink.ReplaceAllString(result, "$1")

	// Table separator rows: remove
	result = reTableSep.ReplaceAllString(result, "")

	// Table rows: pipe-delimited to space-delimited
	result = reTableRow.ReplaceAllStringFunc(result, func(match string) string {
		parts := reTableRow.FindStringSubmatch(match)
		if len(parts) > 1 {
			cells := strings.Split(parts[1], "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			return strings.Join(cells, "  ")
		}
		return match
	})

	// Headers: remove # prefix
	result = reHeader.ReplaceAllString(result, "")

	// Bold
	result = reBold.ReplaceAllStringFunc(result, func(match string) string {
		parts := reBold.FindStringSubmatch(match)
		if parts[1] != "" {
			return parts[1]
		}
		return parts[2]
	})

	// Strikethrough
	result = reStrike.ReplaceAllString(result, "$1")

	// Blockquote
	result = reBlockquote.ReplaceAllString(result, "")

	// Horizontal rule -> empty line
	result = reHR.ReplaceAllString(result, "")

	// Unordered list: replace markers with "• "
	result = reUL.ReplaceAllString(result, "${1}• ")

	// Ordered list: normalize ")" markers to "." markers
	result = reOL.ReplaceAllString(result, "${1}${2}. ")

	result = normalizeInlineBullets(result)

	// Inline code: strip backticks (do after code blocks)
	result = reInlineCode.ReplaceAllString(result, "$1")

	result = cleanupPlainTextSpacing(result)

	return strings.TrimSpace(result)
}

// PlainTextReplyChunks is kept for internal fallback callers and tests.
// User-visible WeChat text should prefer ClawBotMarkdownReplyChunks.
func PlainTextReplyChunks(markdown string) []string {
	plain := MarkdownToPlainText(markdown)
	if plain == "" {
		return nil
	}

	paragraphs := strings.Split(plain, "\n\n")
	chunks := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if len(chunks) > 0 && strings.HasSuffix(chunks[len(chunks)-1], ":") && isListBlock(paragraph) {
			chunks[len(chunks)-1] += "\n" + paragraph
			continue
		}
		chunks = append(chunks, paragraph)
	}
	return chunks
}

func normalizeLineEndings(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func applyPlainTextMarkers(text string) string {
	if idx := strings.Index(text, "[EOF]"); idx >= 0 {
		tail := text[idx+len("[EOF]"):]
		if reMarkerOnlyTail.MatchString(tail) {
			text = text[:idx]
		} else {
			text = text[:idx] + tail
		}
	}
	text = strings.ReplaceAll(text, "[PARAGRAPH]", "\n\n")
	text = strings.ReplaceAll(text, "[NEW LINE]", "\n")
	text = strings.ReplaceAll(text, "[ITEM]", "• ")
	text = strings.ReplaceAll(text, "[EOF]", "")
	return text
}

func applyClawBotMarkdownMarkers(text string) string {
	if idx := strings.Index(text, "[EOF]"); idx >= 0 {
		tail := text[idx+len("[EOF]"):]
		if reMarkerOnlyTail.MatchString(tail) {
			text = text[:idx]
		} else {
			text = text[:idx] + tail
		}
	}
	text = strings.ReplaceAll(text, "[PARAGRAPH]", "\n\n")
	text = strings.ReplaceAll(text, "[NEW LINE]", "\n")
	text = strings.ReplaceAll(text, "[ITEM]", "- ")
	text = strings.ReplaceAll(text, "[EOF]", "")
	return text
}

func normalizeEnglishQuoteCharacters(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if containsHan(line) {
			continue
		}
		line = strings.ReplaceAll(line, "“", `"`)
		line = strings.ReplaceAll(line, "”", `"`)
		line = strings.ReplaceAll(line, "‘", "'")
		line = strings.ReplaceAll(line, "’", "'")
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func containsHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func normalizeClawBotMarkdownListMarkers(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false

	for _, line := range lines {
		trimmedLeft := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmedLeft)]
		if strings.HasPrefix(trimmedLeft, "```") {
			out = append(out, line)
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmedLeft, "• ") {
			out = append(out, indent+"- "+strings.TrimSpace(strings.TrimPrefix(trimmedLeft, "• ")))
			continue
		}
		if strings.ContainsRune(line, '•') {
			out = append(out, splitInlineMarkdownBulletLine(line)...)
			continue
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func splitInlineMarkdownBulletLine(line string) []string {
	parts := strings.Split(line, "•")
	out := make([]string, 0, len(parts))
	if prefix := strings.TrimRight(parts[0], " \t"); strings.TrimSpace(prefix) != "" {
		out = append(out, prefix)
	}
	for _, part := range parts[1:] {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, "- "+item)
	}
	if len(out) == 0 {
		return []string{line}
	}
	return out
}

func cleanupMarkdownSpacing(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 1 {
				out = append(out, "")
			}
			continue
		}
		blankCount = 0
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func ensureBalancedCodeFence(text string) string {
	openFence := markdownFenceSpec{}
	for _, line := range strings.Split(text, "\n") {
		spec, ok := markdownFenceLineSpec(line)
		if !ok {
			continue
		}
		if openFence.valid {
			if markdownFenceCanClose(spec, openFence) {
				openFence = markdownFenceSpec{}
			}
			continue
		}
		openFence = spec
	}
	if openFence.valid {
		return strings.TrimRight(text, "\n") + "\n" + markdownFenceText(openFence)
	}
	return text
}

func splitMarkdownBlocks(text string) []string {
	lines := strings.Split(text, "\n")
	blocks := make([]string, 0)
	current := make([]string, 0)
	openFence := markdownFenceSpec{}

	flush := func() {
		block := strings.TrimSpace(strings.Join(current, "\n"))
		if block != "" {
			blocks = append(blocks, block)
		}
		current = current[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		current = append(current, line)

		if spec, ok := markdownFenceLineSpec(trimmed); ok {
			if openFence.valid {
				if markdownFenceCanClose(spec, openFence) {
					openFence = markdownFenceSpec{}
				}
			} else {
				openFence = spec
			}
			continue
		}

		if openFence.valid {
			continue
		}

		if trimmed == "" {
			flush()
		}
	}
	flush()
	return blocks
}

func isMarkdownListBlock(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return isBulletLikeLine(trimmed) || isNumberedListLikeLine(trimmed)
	}
	return false
}

func isListBlock(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isBulletLikeLine(trimmed) {
			return true
		}
		dot := strings.Index(trimmed, ". ")
		if dot > 0 {
			allDigits := true
			for _, r := range trimmed[:dot] {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
		return false
	}
	return false
}

func normalizeInlineBullets(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.ContainsRune(line, '•') {
			out = append(out, line)
			continue
		}
		out = append(out, splitInlineBulletLine(line)...)
	}
	return strings.Join(out, "\n")
}

func splitInlineBulletLine(line string) []string {
	parts := strings.Split(line, "•")
	out := make([]string, 0, len(parts))
	if prefix := strings.TrimRight(parts[0], " \t"); strings.TrimSpace(prefix) != "" {
		out = append(out, prefix)
	}
	for _, part := range parts[1:] {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, "• "+item)
	}
	if len(out) == 0 {
		return []string{line}
	}
	return out
}

func cleanupPlainTextSpacing(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 1 {
				out = append(out, "")
			}
			continue
		}
		blankCount = 0
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
