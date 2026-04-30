package cmd

import "testing"

func TestIsManagedWeclawProcess(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "managed foreground child",
			args: []string{"/usr/local/bin/weclaw", "start", "-f"},
			want: true,
		},
		{
			name: "managed foreground child long flag",
			args: []string{"/tmp/weclaw-dev", "start", "--foreground"},
			want: true,
		},
		{
			name: "background starter is not managed child",
			args: []string{"/usr/local/bin/weclaw", "start"},
			want: false,
		},
		{
			name: "status command is not managed child",
			args: []string{"/usr/local/bin/weclaw", "status"},
			want: false,
		},
		{
			name: "other binary is ignored",
			args: []string{"/usr/bin/bash", "start", "-f"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isManagedWeclawProcess(tc.args); got != tc.want {
				t.Fatalf("isManagedWeclawProcess(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
