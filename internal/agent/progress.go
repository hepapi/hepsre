package agent

import "github.com/emirozbir/micro-sre/internal/ui"

// NoOpProgressReporter is a no-op implementation for JSON/API modes
type NoOpProgressReporter struct{}

func (n *NoOpProgressReporter) Update(message string) {}
func (n *NoOpProgressReporter) Stop()                 {}

// Ensure NoOpProgressReporter implements ui.ProgressReporter
var _ ui.ProgressReporter = (*NoOpProgressReporter)(nil)
