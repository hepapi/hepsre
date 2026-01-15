package ui

import (
	"time"

	"github.com/briandowns/spinner"
)

// ProgressReporter interface for reporting progress during analysis
type ProgressReporter interface {
	Update(message string)
	Stop()
}

// SpinnerProgress implements ProgressReporter using briandowns/spinner
type SpinnerProgress struct {
	spinner *spinner.Spinner
}

// NewSpinnerProgress creates a new spinner-based progress reporter
func NewSpinnerProgress() *SpinnerProgress {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "  "
	s.Color("cyan", "bold")

	return &SpinnerProgress{
		spinner: s,
	}
}

// Start starts the spinner with an initial message
func (sp *SpinnerProgress) Start(message string) {
	sp.spinner.Suffix = "  " + message
	sp.spinner.Start()
}

// Update updates the spinner message
func (sp *SpinnerProgress) Update(message string) {
	sp.spinner.Suffix = "  " + message
}

// Stop stops the spinner
func (sp *SpinnerProgress) Stop() {
	if sp.spinner.Active() {
		sp.spinner.Stop()
	}
}
