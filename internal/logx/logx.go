// Package logx prints colorful, informative status output for the lumina CLI.
package logx

import (
	"fmt"
	"os"
)

const (
	colorRed    = "31"
	colorGreen  = "32"
	colorYellow = "33"
	colorBlue   = "34"
	colorCyan   = "36"
	colorBold   = "1"
)

// enabled reports whether ANSI color codes should be emitted. Disabled when
// NO_COLOR is set, TERM=dumb, or stdout is not a terminal (e.g. piped/redirected).
var enabled = shouldColor()

func shouldColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func colorize(code, s string) string {
	if !enabled {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Section prints a bold banner marking a new phase of work (e.g. "Build all").
func Section(format string, a ...any) {
	title := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stdout, colorize(colorBold, "== "+title+" =="))
}

// Step announces an action about to be taken (e.g. "Compiling PDF...").
func Step(format string, a ...any) {
	fmt.Fprintln(os.Stdout, colorize(colorCyan, "→")+" "+fmt.Sprintf(format, a...))
}

// Info prints a neutral, informational line.
func Info(format string, a ...any) {
	fmt.Fprintln(os.Stdout, colorize(colorBlue, "•")+" "+fmt.Sprintf(format, a...))
}

// Success announces that an action completed successfully.
func Success(format string, a ...any) {
	fmt.Fprintln(os.Stdout, colorize(colorGreen, "✓")+" "+fmt.Sprintf(format, a...))
}

// Warn prints a non-fatal warning to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, colorize(colorYellow, "⚠")+" "+fmt.Sprintf(format, a...))
}

// Error prints a failure message to stderr.
func Error(format string, a ...any) {
	fmt.Fprintln(os.Stderr, colorize(colorRed, "✗")+" "+fmt.Sprintf(format, a...))
}
