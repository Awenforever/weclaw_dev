package cmd

import "testing"

func TestParseStartSelection(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProfile string
		wantResume  string
		wantErr     bool
	}{
		{name: "empty", args: nil, wantProfile: ""},
		{name: "deepseek", args: []string{"deepseek"}, wantProfile: "deepseek"},
		{name: "deepseek thinking", args: []string{"deepseek-thinking"}, wantProfile: "deepseek-thinking"},
		{name: "deepseek resume", args: []string{"deepseek", "resume", "thread-123"}, wantProfile: "deepseek", wantResume: "thread-123"},
		{name: "deepseek thinking resume", args: []string{"deepseek-thinking", "resume", "thread-456"}, wantProfile: "deepseek-thinking", wantResume: "thread-456"},
		{name: "reject thinking alias", args: []string{"thinking"}, wantErr: true},
		{name: "reject non thinking alias", args: []string{"non-thinking"}, wantErr: true},
		{name: "reject resume without profile", args: []string{"resume", "thread-123"}, wantErr: true},
		{name: "reject unsupported action", args: []string{"deepseek", "continue", "thread-123"}, wantErr: true},
		{name: "reject multiple args", args: []string{"deepseek", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStartSelection(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseStartSelection(%v) returned nil error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStartSelection(%v) error = %v", tt.args, err)
			}
			if got.Profile != tt.wantProfile || got.ResumeID != tt.wantResume {
				t.Fatalf("parseStartSelection(%v) = %#v, want profile=%q resume=%q", tt.args, got, tt.wantProfile, tt.wantResume)
			}
		})
	}
}

func TestParseStartProfileCompatibility(t *testing.T) {
	got, err := parseStartProfile([]string{"deepseek-thinking"})
	if err != nil {
		t.Fatalf("parseStartProfile returned error: %v", err)
	}
	if got != "deepseek-thinking" {
		t.Fatalf("parseStartProfile = %q, want deepseek-thinking", got)
	}
}
