//go:build windows

package jsonrpcipc

import (
	"testing"
)

func TestNormalizeWindowsPipePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "myapp",
			want:  `\\.\pipe\myapp`,
		},
		{
			name:  "already prefixed with pipe",
			input: `\\.\pipe\myapp`,
			want:  `\\.\pipe\myapp`,
		},
		{
			name:  "already prefixed with question pipe",
			input: `\\?\pipe\myapp`,
			want:  `\\?\pipe\myapp`,
		},
		{
			name:  "name with special chars",
			input: "my-app_123",
			want:  `\\.\pipe\my-app_123`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeWindowsPipePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizeWindowsPipePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
