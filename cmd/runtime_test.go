package cmd

import "testing"

func TestIsWeclawProcess(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "installed weclaw binary",
			args: []string{"/usr/local/bin/weclaw", "status"},
			want: true,
		},
		{
			name: "temporary weclaw binary",
			args: []string{"/tmp/weclaw-dev", "start", "-f"},
			want: true,
		},
		{
			name: "non-weclaw binary",
			args: []string{"/usr/bin/bash", "start", "-f"},
			want: false,
		},
		{
			name: "empty args",
			args: nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isWeclawProcess(tc.args); got != tc.want {
				t.Fatalf("isWeclawProcess(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

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
			name: "managed deepseek thinking foreground child",
			args: []string{"/usr/local/bin/weclaw", "start", "deepseek-thinking", "-f"},
			want: true,
		},
		{
			name: "managed resume foreground child",
			args: []string{"/usr/local/bin/weclaw", "start", "deepseek-thinking", "resume", "thread-1", "-f"},
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

func TestManagedProcessPIDsSortsAndCompacts(t *testing.T) {
	got := managedProcessPIDs([]managedProcess{
		{PID: 42},
		{PID: 7},
		{PID: 42},
		{PID: 0},
	})
	if formatPIDs(got) != "7, 42" {
		t.Fatalf("managedProcessPIDs/formatPIDs = %q, want 7, 42", formatPIDs(got))
	}
}

func TestFormatPIDsEmpty(t *testing.T) {
	if got := formatPIDs(nil); got != "" {
		t.Fatalf("formatPIDs(nil) = %q, want empty", got)
	}
}
