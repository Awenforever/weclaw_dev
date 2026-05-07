package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fastclaw-ai/weclaw/ilink"
)

func TestExtractImageURLs(t *testing.T) {
	text := "check ![img](https://example.com/a.png) and ![](https://example.com/b.jpg)"
	urls := ExtractImageURLs(text)
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(urls))
	}
	if urls[0] != "https://example.com/a.png" {
		t.Errorf("urls[0] = %q", urls[0])
	}
	if urls[1] != "https://example.com/b.jpg" {
		t.Errorf("urls[1] = %q", urls[1])
	}
}

func TestExtractImageURLs_NoImages(t *testing.T) {
	urls := ExtractImageURLs("just plain text")
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}

func TestExtractImageURLs_RelativeURL(t *testing.T) {
	text := "![img](./local.png)"
	urls := ExtractImageURLs(text)
	if len(urls) != 0 {
		t.Errorf("expected 0 urls for relative path, got %d", len(urls))
	}
}

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/photo.png", "photo.png"},
		{"https://example.com/path/to/report.pdf", "report.pdf"},
		{"https://example.com/file", "file"},
	}
	for _, tt := range tests {
		got := filenameFromURL(tt.url)
		if got != tt.want {
			t.Errorf("filenameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestFilenameFromURL_WithQuery(t *testing.T) {
	got := filenameFromURL("https://example.com/photo.png?token=abc")
	if got != "photo.png" {
		t.Errorf("got %q, want %q", got, "photo.png")
	}
}

func TestStripQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/a?b=c", "https://example.com/a"},
		{"https://example.com/a", "https://example.com/a"},
		{"https://example.com/?x=1&y=2", "https://example.com/"},
	}
	for _, tt := range tests {
		got := stripQuery(tt.input)
		if got != tt.want {
			t.Errorf("stripQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTextSendThenMediaSendArePaced(t *testing.T) {
	const minGap = 40 * time.Millisecond
	const tolerance = 15 * time.Millisecond

	var sentAt []time.Time
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ilink/bot/sendmessage":
			var req ilink.SendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode sendmessage: %v", err)
			}
			sentAt = append(sentAt, time.Now())
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ret":0}`))
		case "/ilink/bot/getuploadurl":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ret":0,"upload_full_url":"` + server.URL + `/cdn/upload"}`))
		case "/cdn/upload":
			w.Header().Set("X-Encrypted-Param", "download-token")
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := ilink.NewClient(&ilink.Credentials{
		BotToken:   "token",
		ILinkBotID: "bot-1",
		BaseURL:    server.URL,
	})
	client.SetSendMessageMinGap(minGap)

	filePath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	if err := SendTextReply(context.Background(), client, "user-1", "hello", "ctx-token", "client-1"); err != nil {
		t.Fatalf("SendTextReply returned error: %v", err)
	}
	if err := SendMediaFromPath(context.Background(), client, "user-1", filePath, "ctx-token"); err != nil {
		t.Fatalf("SendMediaFromPath returned error: %v", err)
	}

	assertMinSendGap(t, sentAt, minGap, tolerance)
}

func assertMinSendGap(t *testing.T, sentAt []time.Time, minGap, tolerance time.Duration) {
	t.Helper()
	if len(sentAt) < 2 {
		t.Fatalf("sentAt has %d entries, want at least 2", len(sentAt))
	}
	for i := 1; i < len(sentAt); i++ {
		gap := sentAt[i].Sub(sentAt[i-1])
		if gap+tolerance < minGap {
			t.Fatalf("gap[%d] = %s, want at least %s within tolerance %s", i-1, gap, minGap, tolerance)
		}
	}
}
