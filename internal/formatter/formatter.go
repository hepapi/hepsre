package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/emirozbir/micro-sre/internal/models"
)

const (
	divider      = "═══════════════════════════════════════════════════════════════════════════════"
	sectionBreak = "───────────────────────────────────────────────────────────────────────────────"
)

type Formatter struct {
	useColors bool
}

func NewFormatter(useColors bool) *Formatter {
	return &Formatter{
		useColors: useColors,
	}
}

func (f *Formatter) FormatAnalysisResult(result *models.AnalysisResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString(Colorize(Cyan, divider))
	sb.WriteString("\n")
	sb.WriteString(Title("  🔍 MICRO-SRE INCIDENT ANALYSIS REPORT"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Cyan, divider))
	sb.WriteString("\n\n")

	// Alert Summary
	f.writeAlertSummary(&sb, result.Alert)

	// Root Cause
	f.writeRootCause(&sb, result.Analysis)

	// Timeline
	if len(result.Analysis.Timeline) > 0 {
		f.writeTimeline(&sb, result.Analysis.Timeline)
	}

	// Evidence
	f.writeEvidence(&sb, result.Analysis.Evidence)

	// Recommendations
	if len(result.Analysis.Recommendations) > 0 {
		f.writeRecommendations(&sb, result.Analysis.Recommendations)
	}

	// Collection Stats
	f.writeCollectionStats(&sb, result.CollectedData)

	// Footer
	sb.WriteString("\n")
	sb.WriteString(Colorize(Cyan, divider))
	sb.WriteString("\n")

	return sb.String()
}

func (f *Formatter) writeAlertSummary(sb *strings.Builder, alert models.AlertSummary) {
	sb.WriteString(SectionHeader("📋 ALERT SUMMARY"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	if alert.Name != "" && alert.Name != "Alert" {
		sb.WriteString(fmt.Sprintf("  Alert Name:  %s\n", BoldColorize(White, alert.Name)))
	}
	if alert.Severity != "" {
		sb.WriteString(fmt.Sprintf("  Severity:    %s\n", SeverityBadge(alert.Severity)))
	}
	sb.WriteString(fmt.Sprintf("  Namespace:   %s\n", Info(alert.Namespace)))
	sb.WriteString(fmt.Sprintf("  Pod:         %s\n", Info(alert.Pod)))
	sb.WriteString(fmt.Sprintf("  Started At:  %s\n", Muted(alert.StartedAt.Format(time.RFC3339))))
	sb.WriteString("\n")
}

func (f *Formatter) writeRootCause(sb *strings.Builder, analysis models.Analysis) {
	sb.WriteString(SectionHeader("🎯 ROOT CAUSE ANALYSIS"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Confidence:  %s\n", ConfidenceBadge(analysis.Confidence)))
	sb.WriteString(fmt.Sprintf("  Root Cause:  %s\n\n", BoldColorize(Yellow, analysis.RootCause)))

	if analysis.Reasoning != "" {
		sb.WriteString(Colorize(Gray, "  Detailed Reasoning:"))
		sb.WriteString("\n")
		sb.WriteString(f.indentText(analysis.Reasoning, "    "))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (f *Formatter) writeTimeline(sb *strings.Builder, timeline []models.TimelineEvent) {
	sb.WriteString(SectionHeader("⏰ EVENT TIMELINE"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	for i, event := range timeline {
		timeStr := event.Timestamp.Format("15:04:05")
		sb.WriteString(fmt.Sprintf("  %s %s %s\n",
			Colorize(Magenta, timeStr),
			Colorize(Gray, "│"),
			BoldColorize(White, event.Event),
		))

		if event.Details != "" {
			sb.WriteString(fmt.Sprintf("  %s %s %s\n",
				Muted(strings.Repeat(" ", len(timeStr))),
				Colorize(Gray, "└─"),
				Muted(event.Details),
			))
		}

		if i < len(timeline)-1 {
			sb.WriteString(fmt.Sprintf("  %s %s\n",
				strings.Repeat(" ", len(timeStr)),
				Colorize(Gray, "│"),
			))
		}
	}
	sb.WriteString("\n")
}

func (f *Formatter) writeEvidence(sb *strings.Builder, evidence models.Evidence) {
	hasEvidence := len(evidence.Logs) > 0 || len(evidence.Events) > 0

	if !hasEvidence {
		return
	}

	sb.WriteString(SectionHeader("🔎 EVIDENCE"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	// Log Evidence
	if len(evidence.Logs) > 0 {
		sb.WriteString(BoldColorize(White, "  Key Log Entries:"))
		sb.WriteString("\n\n")

		for i, log := range evidence.Logs {
			timeStr := log.Timestamp.Format("15:04:05")
			sb.WriteString(fmt.Sprintf("    %s. %s %s\n",
				Colorize(Yellow, fmt.Sprintf("%d", i+1)),
				Colorize(Magenta, timeStr),
				Muted("→"),
			))

			// Indent and colorize log line
			logLine := strings.TrimSpace(log.Line)
			if strings.Contains(strings.ToLower(logLine), "error") ||
				strings.Contains(strings.ToLower(logLine), "fatal") {
				sb.WriteString(fmt.Sprintf("       %s\n", Error(logLine)))
			} else if strings.Contains(strings.ToLower(logLine), "warn") {
				sb.WriteString(fmt.Sprintf("       %s\n", Warning(logLine)))
			} else {
				sb.WriteString(fmt.Sprintf("       %s\n", logLine))
			}

			if log.Container != "" {
				sb.WriteString(fmt.Sprintf("       %s\n", Muted(fmt.Sprintf("Container: %s", log.Container))))
			}
			sb.WriteString("\n")
		}
	}

	// Event Evidence
	if len(evidence.Events) > 0 {
		sb.WriteString(BoldColorize(White, "  Related Kubernetes Events:"))
		sb.WriteString("\n\n")

		for i, event := range evidence.Events {
			timeStr := event.Timestamp.Format("15:04:05")
			eventType := event.Type
			if eventType == "Warning" {
				eventType = Warning("Warning")
			} else if eventType == "Normal" {
				eventType = Success("Normal")
			}

			sb.WriteString(fmt.Sprintf("    %s. %s [%s] %s\n",
				Colorize(Yellow, fmt.Sprintf("%d", i+1)),
				Colorize(Magenta, timeStr),
				eventType,
				BoldColorize(White, event.Reason),
			))
			sb.WriteString(fmt.Sprintf("       %s\n\n", Muted(event.Message)))
		}
	}
}

func (f *Formatter) writeRecommendations(sb *strings.Builder, recommendations []models.Recommendation) {
	sb.WriteString(SectionHeader("💡 RECOMMENDATIONS"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	for i, rec := range recommendations {
		sb.WriteString(fmt.Sprintf("  %s. %s %s\n",
			Colorize(Yellow, fmt.Sprintf("%d", i+1)),
			PriorityBadge(rec.Priority),
			BoldColorize(White, rec.Action),
		))

		if rec.Details != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", Muted(rec.Details)))
		}

		if rec.Command != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", Muted("Command:")))
			sb.WriteString(fmt.Sprintf("     %s\n", Colorize(Green, fmt.Sprintf("$ %s", rec.Command))))
		}
		sb.WriteString("\n")
	}
}

func (f *Formatter) writeCollectionStats(sb *strings.Builder, data models.CollectedData) {
	sb.WriteString(SectionHeader("📊 DATA COLLECTION STATS"))
	sb.WriteString("\n")
	sb.WriteString(Colorize(Gray, sectionBreak))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Log Lines:    %s\n", Info(fmt.Sprintf("%d", data.LogLines))))
	sb.WriteString(fmt.Sprintf("  Events:       %s\n", Info(fmt.Sprintf("%d", data.EventsCount))))
	sb.WriteString(fmt.Sprintf("  Time Range:   %s\n", Info(data.TimeRange)))
	sb.WriteString("\n")
}

func (f *Formatter) indentText(text string, indent string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			result.WriteString(indent)
			result.WriteString(line)
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
