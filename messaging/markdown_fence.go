package messaging

import "strings"

type markdownFenceSpec struct {
	marker rune
	length int
	valid  bool
}

func longestRuneRun(text string, target rune) int {
	maxRun := 0
	current := 0
	for _, r := range text {
		if r == target {
			current++
			if current > maxRun {
				maxRun = current
			}
			continue
		}
		current = 0
	}
	return maxRun
}

func markdownFenceForContent(content string) string {
	n := longestRuneRun(content, '`') + 1
	if n < 3 {
		n = 3
	}
	return strings.Repeat("`", n)
}

func markdownFenceForLines(lines ...string) string {
	return markdownFenceForContent(strings.Join(lines, "\n"))
}

func markdownFenceLineSpec(line string) (markdownFenceSpec, bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return markdownFenceSpec{}, false
	}
	first := rune(trimmed[0])
	if first != '`' && first != '~' {
		return markdownFenceSpec{}, false
	}
	count := 0
	for _, r := range trimmed {
		if r != first {
			break
		}
		count++
	}
	if count < 3 {
		return markdownFenceSpec{}, false
	}
	return markdownFenceSpec{marker: first, length: count, valid: true}, true
}

func markdownFenceCanClose(candidate, opener markdownFenceSpec) bool {
	return opener.valid && candidate.valid && candidate.marker == opener.marker && candidate.length >= opener.length
}

func markdownFenceText(spec markdownFenceSpec) string {
	if !spec.valid || spec.length < 3 {
		return "```"
	}
	return strings.Repeat(string(spec.marker), spec.length)
}

func sanitizeFenceInfo(info string) string {
	info = strings.TrimSpace(info)
	if info == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range info {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == '+':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wrapMarkdownFence(content, info string) string {
	fence := markdownFenceForContent(content)
	info = sanitizeFenceInfo(info)
	var b strings.Builder
	if info != "" {
		b.WriteString(fence)
		b.WriteString(info)
		b.WriteByte('\n')
	} else {
		b.WriteString(fence)
		b.WriteByte('\n')
	}
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(fence)
	return b.String()
}

func markdownInlineCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "unknown"
	}
	n := longestRuneRun(value, '`') + 1
	if n < 1 {
		n = 1
	}
	ticks := strings.Repeat("`", n)
	content := value
	if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") || strings.HasPrefix(content, " ") || strings.HasSuffix(content, " ") {
		content = " " + content + " "
	}
	return ticks + content + ticks
}
