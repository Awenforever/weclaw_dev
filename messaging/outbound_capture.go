package messaging

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const outboundMarkdownCaptureDirEnv = "WECLAW_CAPTURE_OUTBOUND_MARKDOWN_DIR"

var outboundMarkdownCaptureSeq uint64

func captureOutboundMarkdownForDebug(toUserID, clientID, text string) {
	dir := strings.TrimSpace(os.Getenv(outboundMarkdownCaptureDirEnv))
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		log.Printf("[sender] ignoring outbound markdown capture dir because it is not absolute: %q", dir)
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("[sender] failed to create outbound markdown capture dir %q: %v", dir, err)
		return
	}

	seq := atomic.AddUint64(&outboundMarkdownCaptureSeq, 1)
	now := time.Now().UTC()
	sum := sha256.Sum256([]byte(text))
	name := fmt.Sprintf("%s-%06d-%s.txt", now.Format("20060102T150405.000000000Z"), seq, safeCaptureFilename(clientID))
	path := filepath.Join(dir, name)

	var b strings.Builder
	b.WriteString("===== OUTBOUND MARKDOWN CAPTURE =====\n")
	b.WriteString("time_utc=")
	b.WriteString(now.Format(time.RFC3339Nano))
	b.WriteByte('\n')
	b.WriteString("to_user=")
	b.WriteString(toUserID)
	b.WriteByte('\n')
	b.WriteString("client_id=")
	b.WriteString(clientID)
	b.WriteByte('\n')
	b.WriteString("bytes=")
	b.WriteString(fmt.Sprintf("%d", len([]byte(text))))
	b.WriteByte('\n')
	b.WriteString("sha256=")
	b.WriteString(fmt.Sprintf("%x", sum))
	b.WriteString("\n\n")
	b.WriteString("===== OUTBOUND MARKDOWN =====\n")
	b.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		b.WriteByte('\n')
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		log.Printf("[sender] failed to write outbound markdown capture %q: %v", path, err)
		return
	}
	log.Printf("[sender] captured outbound markdown to %s", path)
}

func safeCaptureFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "no-client-id"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "client-id"
	}
	if len(out) > 80 {
		return out[:80]
	}
	return out
}
