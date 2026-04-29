package messaging

import "strings"

type assistantStreamChunkKind int

const (
	assistantStreamChunkSuppress assistantStreamChunkKind = iota
	assistantStreamChunkProgress
	assistantStreamChunkAnswer
)

type assistantDeltaStreamClassifier struct {
	answerPhase bool
}

func (c *assistantDeltaStreamClassifier) Classify(chunk string) assistantStreamChunkKind {
	if strings.TrimSpace(chunk) == "" {
		return assistantStreamChunkSuppress
	}
	if isFinalAnswerLikeAssistantChunk(chunk) {
		c.answerPhase = true
		return assistantStreamChunkAnswer
	}
	if c.answerPhase {
		return assistantStreamChunkAnswer
	}
	if isOperationalAssistantProgressChunk(chunk) {
		return assistantStreamChunkProgress
	}
	return assistantStreamChunkAnswer
}

func isOperationalAssistantProgressChunk(chunk string) bool {
	line := strings.TrimSpace(firstNonEmptyLine(MarkdownToPlainText(chunk)))
	lower := strings.ToLower(line)
	if line == "" {
		return false
	}

	switch {
	case strings.HasPrefix(line, "I'll ") || strings.HasPrefix(line, "I\u2019ll "):
		return containsAny(lower, []string{"check", "ground", "look", "read", "verify", "inspect", "test", "update"})
	case strings.HasPrefix(line, "I'm ") || strings.HasPrefix(line, "I\u2019m ") || strings.HasPrefix(line, "I am "):
		return containsAny(lower, []string{"checking", "grounding", "reading", "verifying", "inspecting", "testing", "updating", "looking"})
	case strings.HasPrefix(line, "Memory mostly confirms"):
		return true
	case strings.HasPrefix(line, "The current docs split"):
		return true
	}
	return false
}

func isFinalAnswerLikeAssistantChunk(chunk string) bool {
	raw := strings.TrimSpace(normalizeLineEndings(chunk))
	plain := strings.TrimSpace(MarkdownToPlainText(chunk))
	if raw == "" || plain == "" {
		return false
	}

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || isTableLikeLine(trimmed) {
			return true
		}
	}

	first := firstNonEmptyLine(plain)
	lowerFirst := strings.ToLower(first)
	switch {
	case strings.HasPrefix(lowerFirst, "short conclusion"):
		return true
	case strings.HasPrefix(lowerFirst, "you are comparing"):
		return true
	case strings.HasPrefix(lowerFirst, "dimension "):
		return true
	case strings.HasPrefix(lowerFirst, "pros ") || strings.HasPrefix(lowerFirst, "pros:"):
		return true
	case strings.HasPrefix(lowerFirst, "cons ") || strings.HasPrefix(lowerFirst, "cons:"):
		return true
	case strings.HasPrefix(lowerFirst, "complexity ") || strings.HasPrefix(lowerFirst, "complexity:"):
		return true
	case strings.HasPrefix(lowerFirst, "capability ") || strings.HasPrefix(lowerFirst, "capability:"):
		return true
	case isNumberedListLikeLine(first):
		return true
	case isBulletLikeLine(first):
		return true
	case wordCount(plain) > 38:
		return true
	}

	for _, line := range strings.Split(plain, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isNumberedListLikeLine(trimmed) || isBulletLikeLine(trimmed) {
			return true
		}
	}
	return false
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(normalizeLineEndings(text), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isTableLikeLine(line string) bool {
	if strings.Count(line, "|") >= 2 {
		return true
	}
	return strings.HasPrefix(strings.ToLower(line), "dimension ") && strings.Contains(line, "  ")
}

func isBulletLikeLine(line string) bool {
	return strings.HasPrefix(line, "• ") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ")
}

func isNumberedListLikeLine(line string) bool {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) {
		return false
	}
	if line[i] != '.' && line[i] != ')' {
		return false
	}
	return i+1 == len(line) || line[i+1] == ' ' || line[i+1] == '\t'
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func wordCount(text string) int {
	return len(strings.Fields(text))
}

type assistantTextStreamBuffer struct {
	text    string
	pending string
}

func (b *assistantTextStreamBuffer) Push(delta string, maxChunkChars int) []string {
	if delta == "" {
		return nil
	}
	delta = normalizeLineEndings(delta)

	var out []string
	for {
		before, after, found := strings.Cut(delta, "[EOF]")
		if before != "" {
			b.text = normalizeLineEndings(b.text + before)
		}
		if found {
			out = append(out, b.flushBufferedText(false)...)
			delta = after
			continue
		}
		break
	}

	blocks, rest := completeStreamBlocks(b.text)
	if len(blocks) == 0 {
		return append(out, b.flushBySize(maxChunkChars)...)
	}

	last := blocks[len(blocks)-1]
	if strings.HasSuffix(MarkdownToPlainText(last), ":") && (rest == "" || startsListBlock(rest)) {
		blocks = blocks[:len(blocks)-1]
		rest = prependStreamBlock(last, rest)
	}
	if len(blocks) == 0 {
		b.text = rest
		return nil
	}

	b.text = rest
	out = append(out, b.emitMarkdownChunks(strings.Join(blocks, "\n\n"), false)...)
	out = append(out, b.flushBySize(maxChunkChars)...)
	return out
}

func (b *assistantTextStreamBuffer) Flush() []string {
	return b.flushBufferedText(true)
}

func (b *assistantTextStreamBuffer) flushBufferedText(final bool) []string {
	if strings.TrimSpace(b.text) == "" {
		b.text = ""
		if final {
			return b.flushPending()
		}
		return nil
	}
	chunks := b.emitMarkdownChunks(b.text, final)
	b.text = ""
	return chunks
}

func (b *assistantTextStreamBuffer) FlushReadyForInterval() []string {
	if strings.TrimSpace(b.text) == "" {
		b.text = ""
		return nil
	}

	blocks, rest := completeStreamBlocks(b.text)
	if len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		if strings.HasSuffix(MarkdownToPlainText(last), ":") && (rest == "" || startsListBlock(rest)) {
			blocks = blocks[:len(blocks)-1]
			rest = prependStreamBlock(last, rest)
		}
		if len(blocks) > 0 {
			b.text = rest
			return b.emitMarkdownChunks(strings.Join(blocks, "\n\n"), false)
		}
	}

	prefix, rest, ok := splitStreamTextAtLastSentenceBoundary(b.text)
	if !ok || strings.TrimSpace(prefix) == "" {
		return nil
	}
	b.text = rest
	return b.emitMarkdownChunks(prefix, false)
}

func (b *assistantTextStreamBuffer) flushBySize(maxChunkChars int) []string {
	if maxChunkChars <= 0 || runeLen(b.text) < maxChunkChars {
		return nil
	}
	prefix, rest := splitStreamTextAtLimit(b.text, maxChunkChars)
	if strings.TrimSpace(prefix) == "" {
		return nil
	}
	b.text = strings.TrimLeft(rest, " \t\r\n")
	return b.emitMarkdownChunks(prefix, false)
}

func (b *assistantTextStreamBuffer) emitMarkdownChunks(markdown string, final bool) []string {
	chunks := PlainTextReplyChunks(markdown)
	if len(chunks) == 0 {
		if final {
			return b.flushPending()
		}
		return nil
	}

	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if b.pending != "" {
			chunk = joinPendingStreamChunk(b.pending, chunk)
			b.pending = ""
		}
		if shouldHoldStreamChunk(chunk) && !final {
			b.pending = chunk
			continue
		}
		out = append(out, chunk)
	}
	if final {
		out = append(out, b.flushPending()...)
	}
	return out
}

func (b *assistantTextStreamBuffer) flushPending() []string {
	if strings.TrimSpace(b.pending) == "" {
		b.pending = ""
		return nil
	}
	pending := b.pending
	b.pending = ""
	return []string{pending}
}

func shouldHoldStreamChunk(chunk string) bool {
	plain := strings.TrimSpace(MarkdownToPlainText(chunk))
	return strings.HasSuffix(plain, ":") || isStandaloneListFragment(lastNonEmptyStreamLine(plain))
}

func isStandaloneListFragment(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if text == "•" || text == "-" || text == "*" || text == "+" {
		return true
	}
	if len(text) < 2 {
		return false
	}
	suffix := text[len(text)-1]
	if suffix != '.' && suffix != ')' {
		return false
	}
	for _, r := range text[:len(text)-1] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func joinPendingStreamChunk(pending, chunk string) string {
	pending = strings.TrimSpace(pending)
	chunk = strings.TrimSpace(chunk)
	if pending == "" {
		return chunk
	}
	if chunk == "" {
		return pending
	}
	if strings.HasSuffix(pending, ":") && isListBlock(chunk) {
		return pending + "\n" + chunk
	}
	if pendingEndsWithStandaloneListFragment(pending) {
		return attachToPendingStandaloneListFragment(pending, chunk)
	}
	if isStandaloneListFragment(pending) {
		return pending + " " + chunk
	}
	return pending + "\n\n" + chunk
}

func pendingEndsWithStandaloneListFragment(text string) bool {
	return isStandaloneListFragment(lastNonEmptyStreamLine(MarkdownToPlainText(text)))
}

func attachToPendingStandaloneListFragment(pending, chunk string) string {
	lines := strings.Split(strings.TrimRight(pending, " \t\r\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		lines[i] = strings.TrimRight(lines[i], " \t") + " " + chunk
		return strings.Join(lines, "\n")
	}
	return pending + " " + chunk
}

func lastNonEmptyStreamLine(text string) string {
	lines := strings.Split(normalizeLineEndings(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if trimmed := strings.TrimSpace(lines[i]); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func splitStreamTextAtLimit(text string, maxRunes int) (string, string) {
	if maxRunes <= 0 {
		return text, ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text, ""
	}

	if split := lastParagraphBoundaryAtOrBefore(runes, maxRunes); split > 0 {
		return string(runes[:split]), string(runes[split:])
	}
	if split := lastSentenceBoundaryAtOrBefore(runes, maxRunes); split > 0 {
		return string(runes[:split]), string(runes[split:])
	}
	if split := lastWhitespaceBoundaryAtOrBefore(runes, maxRunes); split > 0 {
		return string(runes[:split]), string(runes[split:])
	}

	split := maxRunes
	return string(runes[:split]), string(runes[split:])
}

func splitStreamTextAtLastSentenceBoundary(text string) (string, string, bool) {
	runes := []rune(text)
	split := lastSentenceBoundaryAtOrBefore(runes, len(runes))
	if split == 0 {
		return "", text, false
	}
	return string(runes[:split]), string(runes[split:]), true
}

func lastParagraphBoundaryAtOrBefore(runes []rune, limit int) int {
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := limit; i >= 2; i-- {
		if runes[i-1] != '\n' {
			continue
		}
		j := i - 2
		for j >= 0 && (runes[j] == ' ' || runes[j] == '\t') {
			j--
		}
		if j >= 0 && runes[j] == '\n' {
			return i
		}
	}
	return 0
}

func lastSentenceBoundaryAtOrBefore(runes []rune, limit int) int {
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := limit - 1; i >= 0; i-- {
		if !isSentenceTerminal(runes[i]) {
			continue
		}
		if i+1 == len(runes) || isStreamBoundarySpace(runes[i+1]) {
			return i + 1
		}
	}
	return 0
}

func lastWhitespaceBoundaryAtOrBefore(runes []rune, limit int) int {
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := limit; i > 0; i-- {
		if isStreamBoundarySpace(runes[i-1]) {
			return i
		}
	}
	return 0
}

func isSentenceTerminal(r rune) bool {
	switch r {
	case '.', '!', '?', '。', '！', '？':
		return true
	default:
		return false
	}
}

func isStreamBoundarySpace(r rune) bool {
	return r == '\n' || r == ' ' || r == '\t'
}

func runeLen(text string) int {
	return len([]rune(text))
}

func completeStreamBlocks(text string) ([]string, string) {
	var blocks []string
	start := 0
	for i := 0; i < len(text)-1; i++ {
		if text[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
			j++
		}
		if j < len(text) && text[j] == '\n' {
			block := strings.TrimSpace(text[start:i])
			if block != "" {
				blocks = append(blocks, block)
			}
			for j < len(text) && text[j] == '\n' {
				j++
			}
			start = j
			i = j - 1
		}
	}
	return blocks, text[start:]
}

func prependStreamBlock(block, rest string) string {
	block = strings.TrimSpace(block)
	if rest == "" {
		return block + "\n\n"
	}
	return block + "\n\n" + strings.TrimLeft(rest, "\n")
}

func startsListBlock(text string) bool {
	return isListBlock(MarkdownToPlainText(text))
}
