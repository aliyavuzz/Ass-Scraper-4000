package engine

import (
	"fmt"
	"strings"
)

// ============================================================================
// COLOR CONSTANTS
// ============================================================================

const (
	ColorReset   = "\033[0m"
	ColorCyan    = "\033[36m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorRed     = "\033[31m"
	ColorMagenta = "\033[35m"
	ColorBold    = "\033[1m"
)

// ============================================================================
// LOGGING
// ============================================================================

func LogInfo(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorCyan, stage, ColorReset, fmt.Sprintf(format, args...))
}
func LogSuccess(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorGreen, stage, ColorReset, fmt.Sprintf(format, args...))
}
func LogWarning(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorYellow, stage, ColorReset, fmt.Sprintf(format, args...))
}
func LogError(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorRed, stage, ColorReset, fmt.Sprintf(format, args...))
}
func LogAI(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorMagenta, stage, ColorReset, fmt.Sprintf(format, args...))
}
func LogAction(stage, format string, args ...interface{}) {
	fmt.Printf("%s[%-10s]%s %s\n", ColorCyan, stage, ColorReset, fmt.Sprintf(format, args...))
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func FormatStep(step ActionStep) string {
	switch strings.ToUpper(step.Action) {
	case "CLICK":
		return fmt.Sprintf("CLICK at (%.0f, %.0f)", step.X, step.Y)
	case "TYPE":
		return fmt.Sprintf("TYPE \"%s\" at (%.0f, %.0f)", step.Value, step.X, step.Y)
	case "SELECT":
		return fmt.Sprintf("SELECT \"%s\" at (%.0f, %.0f)", step.Value, step.X, step.Y)
	case "SCROLL":
		return "SCROLL down"
	case "WAIT_URL":
		return fmt.Sprintf("WAIT_URL \"%s\"", step.Value)
	case "WAIT_ELEMENT":
		return fmt.Sprintf("WAIT_ELEMENT \"%s\"", step.Value)
	case "SCREENSHOT":
		return "SCREENSHOT"
	case "HOVER":
		return fmt.Sprintf("HOVER at (%.0f, %.0f)", step.X, step.Y)
	case "UPLOAD":
		return fmt.Sprintf("UPLOAD \"%s\" at (%.0f, %.0f)", step.Value, step.X, step.Y)
	case "SCRAPE":
		return fmt.Sprintf("SCRAPE %s", step.Value)
	case "WAIT_IDLE":
		return "WAIT_IDLE"
	case "PRESS_KEY":
		return fmt.Sprintf("PRESS_KEY \"%s\"", step.Value)
	case "DONE":
		return "DONE"
	default:
		return step.Action
	}
}
