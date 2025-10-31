//go:build !windows

package jsonrpcipc

import (
	"testing"
)

func TestNormalizeUnixSocketPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "myapp",
			want:  "/tmp/myapp.sock",
		},
		{
			name:  "simple name with sock extension",
			input: "myapp.sock",
			want:  "/tmp/myapp.sock",
		},
		{
			name:  "absolute path",
			input: "/tmp/myapp.sock",
			want:  "/tmp/myapp.sock",
		},
		{
			name:  "relative path with directory",
			input: "./sockets/myapp.sock",
			want:  "./sockets/myapp.sock",
		},
		{
			name:  "name with special chars",
			input: "my-app_123",
			want:  "/tmp/my-app_123.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSocketPath(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSocketPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
