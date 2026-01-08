package formatter

import "fmt"

// ANSI color codes for terminal output
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"

	// Foreground colors
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Background colors
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
)

// Color helpers
func Colorize(color, text string) string {
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

func BoldColorize(color, text string) string {
	return fmt.Sprintf("%s%s%s%s", Bold, color, text, Reset)
}

func Title(text string) string {
	return BoldColorize(Cyan, text)
}

func SectionHeader(text string) string {
	return BoldColorize(Blue, text)
}

func Success(text string) string {
	return Colorize(Green, text)
}

func Warning(text string) string {
	return Colorize(Yellow, text)
}

func Error(text string) string {
	return Colorize(Red, text)
}

func Info(text string) string {
	return Colorize(Cyan, text)
}

func Muted(text string) string {
	return Colorize(Gray, text)
}

func ConfidenceBadge(confidence string) string {
	switch confidence {
	case "high":
		return BoldColorize(Green, "● HIGH")
	case "medium":
		return BoldColorize(Yellow, "● MEDIUM")
	case "low":
		return BoldColorize(Red, "● LOW")
	default:
		return BoldColorize(Gray, "● UNKNOWN")
	}
}

func PriorityBadge(priority string) string {
	switch priority {
	case "high", "critical":
		return BoldColorize(Red, "⚠ HIGH")
	case "medium":
		return BoldColorize(Yellow, "◉ MEDIUM")
	case "low":
		return BoldColorize(Green, "○ LOW")
	default:
		return BoldColorize(Gray, "• NORMAL")
	}
}

func SeverityBadge(severity string) string {
	switch severity {
	case "critical":
		return fmt.Sprintf("%s%s %s %s", Bold, BgRed, severity, Reset)
	case "warning":
		return fmt.Sprintf("%s%s %s %s", Bold, BgYellow, severity, Reset)
	case "info":
		return fmt.Sprintf("%s%s %s %s", Bold, BgBlue, severity, Reset)
	default:
		return severity
	}
}
