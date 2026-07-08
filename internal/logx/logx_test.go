package logx

import "testing"

func TestColorize(t *testing.T) {
	enabled = true
	if got := colorize(colorRed, "boom"); got != "\033[31mboom\033[0m" {
		t.Errorf("expected wrapped ANSI codes, got %q", got)
	}

	enabled = false
	if got := colorize(colorRed, "boom"); got != "boom" {
		t.Errorf("expected plain text when disabled, got %q", got)
	}
}
