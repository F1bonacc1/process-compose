package tui

import (
	"testing"
)

func TestEscapeForAnsiWriter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "ANSI reset before literal bracket not corrupted",
			input: "\x1b[0m] after",
			want:  "\x1b[0m] after",
		},
		{
			name:  "literal tview-like tag is escaped",
			input: "[ButThisWont] visible",
			want:  "[ButThisWont[] visible",
		},
		{
			name:  "structured log with ANSI colors and brackets",
			input: "\x1b[2m2026-01-01T00:00:00Z\x1b[0m [\x1b[32m\x1b[1minfo     \x1b[0m] \x1b[1msome log message\x1b[0m [\x1b[34mmyapp\x1b[0m] \x1b[36mkey\x1b[0m=\x1b[35mvalue\x1b[0m",
			want:  "\x1b[2m2026-01-01T00:00:00Z\x1b[0m [\x1b[32m\x1b[1minfo     \x1b[0m] \x1b[1msome log message\x1b[0m [\x1b[34mmyapp\x1b[0m] \x1b[36mkey\x1b[0m=\x1b[35mvalue\x1b[0m",
		},
		{
			name:  "ANSI sequences preserved",
			input: "\x1b[31mred\x1b[0m",
			want:  "\x1b[31mred\x1b[0m",
		},
		{
			name:  "multiple literal tags escaped",
			input: "[red] and [blue]",
			want:  "[red[] and [blue[]",
		},
		{
			name:  "brackets with spaces not escaped",
			input: "[not a tag because of spaces]",
			want:  "[not a tag because of spaces]",
		},
		{
			name:  "ANSI color followed immediately by literal tag",
			input: "\x1b[0m[SomeTag]",
			want:  "\x1b[0m[SomeTag[]",
		},
		{
			name:  "hex color tag escaped",
			input: "[#ff0000]text",
			want:  "[#ff0000[]text",
		},
		{
			name:  "reset tag escaped",
			input: "[-]reset",
			want:  "[-[]reset",
		},
		{
			name:  "compound style tag escaped",
			input: "[red:-:b]bold red",
			want:  "[red:-:b[]bold red",
		},
		{
			name:  "brackets with underscores not escaped",
			input: "[some_var]",
			want:  "[some_var]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeForAnsiWriter(tt.input)
			if got != tt.want {
				t.Errorf("escapeForAnsiWriter(%q)\n got  %q\n want %q", tt.input, got, tt.want)
			}
		})
	}
}
