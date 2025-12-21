package ui

import (
	"fmt"
	"os"
	"time"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

// Symbols
const (
	SymbolCheck   = "✓"
	SymbolCross   = "✗"
	SymbolArrow   = "→"
	SymbolDot     = "•"
	SymbolWarning = "⚠"
	SymbolInfo    = "ℹ"
	SymbolGame    = "🎮"
	SymbolServer  = "🖥"
	SymbolDocker  = "🐳"
)

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// color wraps text in ANSI color codes if terminal
func color(c, text string) string {
	if !isTerminal() {
		return text
	}
	return c + text + Reset
}

// Success prints a success message
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color(Green, SymbolCheck), msg)
}

// Error prints an error message
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color(Red, SymbolCross), msg)
}

// Warning prints a warning message
func Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color(Yellow, SymbolWarning), msg)
}

// Info prints an info message
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color(Blue, SymbolInfo), msg)
}

// Step prints a step being performed
func Step(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", color(Cyan, SymbolArrow), msg)
}

// Title prints a bold title
func Title(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if isTerminal() {
		fmt.Printf("%s%s%s\n", Bold, msg, Reset)
	} else {
		fmt.Println(msg)
	}
}

// Detail prints an indented detail line
func Detail(label, value string) {
	fmt.Printf("   %s %s\n", color(Gray, label+":"), value)
}

// Spinner represents a loading spinner
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	done     chan bool
	running  bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		done:     make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	if !isTerminal() {
		fmt.Printf("%s %s...\n", SymbolArrow, s.message)
		return
	}
	s.running = true
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r%s %s %s", color(Cyan, s.frames[i]), s.message, color(Gray, "..."))
				i = (i + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner with a result
func (s *Spinner) Stop(success bool) {
	if s.running {
		s.done <- true
		s.running = false
	}
	if isTerminal() {
		fmt.Print("\r\033[K") // Clear line
	}
	if success {
		Success("%s", s.message)
	} else {
		Error("%s", s.message)
	}
}

// StopWithMessage stops the spinner with a custom message
func (s *Spinner) StopWithMessage(success bool, message string) {
	if s.running {
		s.done <- true
		s.running = false
	}
	if isTerminal() {
		fmt.Print("\r\033[K") // Clear line
	}
	if success {
		Success("%s", message)
	} else {
		Error("%s", message)
	}
}

// Table prints a formatted table
func Table(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print headers
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], color(Bold, h))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		for j := 0; j < widths[i]; j++ {
			fmt.Print("─")
		}
		fmt.Print("  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s  ", widths[i], cell)
			}
		}
		fmt.Println()
	}
}

// Box prints text in a box
func Box(title, content string) {
	width := len(title) + 4
	if len(content) > width-4 {
		width = len(content) + 4
	}

	// Top border
	fmt.Print("╭")
	for i := 0; i < width; i++ {
		fmt.Print("─")
	}
	fmt.Println("╮")

	// Title
	fmt.Printf("│ %s%s%s", Bold, title, Reset)
	for i := 0; i < width-len(title)-1; i++ {
		fmt.Print(" ")
	}
	fmt.Println("│")

	// Content
	if content != "" {
		fmt.Printf("│ %s", content)
		for i := 0; i < width-len(content)-1; i++ {
			fmt.Print(" ")
		}
		fmt.Println("│")
	}

	// Bottom border
	fmt.Print("╰")
	for i := 0; i < width; i++ {
		fmt.Print("─")
	}
	fmt.Println("╯")
}
