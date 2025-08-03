package longrunning

import (
	"fmt"
	"time"
)

// ProgressReporter provides convenient methods for reporting progress
type ProgressReporter struct {
	task             *Task
	updateInterval   time.Duration
	lastUpdate       time.Time
	pendingMessage   string
	pendingProgress  float64
	hasPendingUpdate bool
}

// NewProgressReporter creates a progress reporter for a task
func NewProgressReporter(task *Task) *ProgressReporter {
	return &ProgressReporter{
		task:           task,
		updateInterval: 100 * time.Millisecond, // Default rate limit
	}
}

// SetUpdateInterval sets the minimum interval between progress updates
func (r *ProgressReporter) SetUpdateInterval(interval time.Duration) {
	r.updateInterval = interval
}

// ReportProgress reports progress with automatic rate limiting
func (r *ProgressReporter) ReportProgress(progress float64, message string) error {
	now := time.Now()

	// Store pending update
	r.pendingProgress = progress
	r.pendingMessage = message
	r.hasPendingUpdate = true

	// Check if we should send now
	if now.Sub(r.lastUpdate) >= r.updateInterval {
		return r.flush()
	}

	return nil
}

// ReportPercentage reports progress as a percentage of total
func (r *ProgressReporter) ReportPercentage(current, total float64, message string) error {
	if total <= 0 {
		return r.ReportProgress(current, message)
	}

	percentage := (current / total) * 100
	fullMessage := fmt.Sprintf("[%.1f%%] %s", percentage, message)

	return r.ReportProgress(current, fullMessage)
}

// ReportItems reports progress for processing items
func (r *ProgressReporter) ReportItems(current, total int, itemName string) error {
	message := fmt.Sprintf("Processing %s %d of %d", itemName, current, total)
	return r.ReportPercentage(float64(current), float64(total), message)
}

// ReportStep reports a named step in a multi-step process
func (r *ProgressReporter) ReportStep(step, totalSteps int, stepName string) error {
	message := fmt.Sprintf("Step %d/%d: %s", step, totalSteps, stepName)
	return r.ReportPercentage(float64(step), float64(totalSteps), message)
}

// ReportBytes reports progress for byte operations (downloads, uploads, etc.)
func (r *ProgressReporter) ReportBytes(current, total int64) error {
	message := fmt.Sprintf("%s / %s", formatBytes(current), formatBytes(total))
	return r.ReportPercentage(float64(current), float64(total), message)
}

// Flush forces any pending update to be sent
func (r *ProgressReporter) Flush() error {
	if r.hasPendingUpdate {
		return r.flush()
	}
	return nil
}

// flush sends the pending update
func (r *ProgressReporter) flush() error {
	if !r.hasPendingUpdate {
		return nil
	}

	err := r.task.UpdateProgress(r.pendingProgress, r.pendingMessage)
	r.lastUpdate = time.Now()
	r.hasPendingUpdate = false

	return err
}

// Complete flushes any pending updates and marks the task complete
func (r *ProgressReporter) Complete() {
	_ = r.Flush()
	r.task.Complete()
}

// CompleteWithError flushes updates and marks the task as failed
func (r *ProgressReporter) CompleteWithError(err error) {
	_ = r.Flush()
	r.task.CompleteWithError(err)
}

// formatBytes formats byte counts in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// StepTracker helps track progress through a series of steps
type StepTracker struct {
	reporter    *ProgressReporter
	totalSteps  int
	currentStep int
}

// NewStepTracker creates a tracker for step-based progress
func NewStepTracker(task *Task, totalSteps int) *StepTracker {
	task.SetTotal(float64(totalSteps))
	return &StepTracker{
		reporter:    NewProgressReporter(task),
		totalSteps:  totalSteps,
		currentStep: 0,
	}
}

// NextStep advances to the next step and reports progress
func (s *StepTracker) NextStep(stepName string) error {
	s.currentStep++
	return s.reporter.ReportStep(s.currentStep, s.totalSteps, stepName)
}

// Complete marks all steps as done
func (s *StepTracker) Complete() {
	s.reporter.Complete()
}

// ItemProcessor helps process a collection of items with progress
type ItemProcessor struct {
	reporter   *ProgressReporter
	totalItems int
	processed  int
	itemName   string
}

// NewItemProcessor creates a processor for item-based progress
func NewItemProcessor(task *Task, totalItems int, itemName string) *ItemProcessor {
	task.SetTotal(float64(totalItems))
	return &ItemProcessor{
		reporter:   NewProgressReporter(task),
		totalItems: totalItems,
		processed:  0,
		itemName:   itemName,
	}
}

// ProcessItem reports progress for one processed item
func (p *ItemProcessor) ProcessItem() error {
	p.processed++
	return p.reporter.ReportItems(p.processed, p.totalItems, p.itemName)
}

// ProcessItemWithName reports progress with a specific item name
func (p *ItemProcessor) ProcessItemWithName(name string) error {
	p.processed++
	message := fmt.Sprintf("Processing %s: %s (%d of %d)",
		p.itemName, name, p.processed, p.totalItems)
	return p.reporter.ReportPercentage(float64(p.processed), float64(p.totalItems), message)
}

// Complete marks all items as processed
func (p *ItemProcessor) Complete() {
	p.reporter.Complete()
}
