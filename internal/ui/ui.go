// Package ui provides minimal ANSI color helpers and output utilities
// for netpilot. Colors degrade gracefully when stdout is not a TTY.
package ui

import (
	"fmt"
	"os"
)

var enabled = isTTY(os.Stdout)

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
)

func paint(code, s string) string {
	if !enabled {
		return s
	}
	return code + s + reset
}

func Bold(s string) string   { return paint(bold, s) }
func Red(s string) string    { return paint(red, s) }
func Green(s string) string  { return paint(green, s) }
func Yellow(s string) string { return paint(yellow, s) }
func Blue(s string) string   { return paint(blue, s) }
func Cyan(s string) string   { return paint(cyan, s) }
func Gray(s string) string   { return paint(gray, s) }

// Success prints a green check line.
func Success(format string, a ...any) {
	fmt.Printf(" %s %s\n", Green("✓"), fmt.Sprintf(format, a...))
}

// Fail prints a red cross line.
func Fail(format string, a ...any) {
	fmt.Printf(" %s %s\n", Red("✗"), fmt.Sprintf(format, a...))
}

// Info prints a blue info line.
func Info(format string, a ...any) {
	fmt.Printf(" %s %s\n", Blue("●"), fmt.Sprintf(format, a...))
}

// Warn prints a yellow warning line.
func Warn(format string, a ...any) {
	fmt.Printf(" %s %s\n", Yellow("▲"), fmt.Sprintf(format, a...))
}

// Dim prints a gray detail line.
func Dim(format string, a ...any) {
	fmt.Println(Gray("  " + fmt.Sprintf(format, a...)))
}
