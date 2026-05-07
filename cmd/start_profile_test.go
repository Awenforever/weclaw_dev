package cmd

import "testing"

func TestParseStartProfile(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "empty", args: nil, want: ""},
		{name: "deepseek", args: []string{"deepseek"}, want: "deepseek"},
		{name: "deepseek thinking", args: []string{"deepseek-thinking"}, want: "deepseek-thinking"},
		{name: "reject thinking alias", args: []string{"thinking"}, wantErr: true},
		{name: "reject non thinking alias", args: []string{"non-thinking"}, wantErr: true},
		{name: "reject multiple args", args: []string{"deepseek", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStartProfile(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseStartProfile(%v) returned nil error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStartProfile(%v) error = %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("parseStartProfile(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
