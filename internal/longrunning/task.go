package longrunning

import (
	"context"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Task represents a long-running operation with progress tracking.
// It provides thread-safe methods for updating progress, handling cancellation,
// and reporting completion or errors.
type Task struct {
	// Identity
	id            string
	progressToken mcp.ProgressToken
	sessionID     string

	// State
	progress     float64
	total        float64
	message      string
	startTime    time.Time
	endTime      *time.Time
	error        error
	cancelled    bool
	cancelReason string

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Progress tracking
	lastNotified time.Time

	// References
	manager *Manager

	// Thread safety
	mu sync.RWMutex
}

// ID returns the task's unique identifier.
// This is typically derived from the progress token.
func (t *Task) ID() string {
	return t.id
}

// SessionID returns the session this task belongs to
func (t *Task) SessionID() string {
	return t.sessionID
}

// Context returns the task's context for cancellation
func (t *Task) Context() context.Context {
	return t.ctx
}

// SetTotal sets the total expected progress value
func (t *Task) SetTotal(total float64) {
	t.mu.Lock()
	t.total = total
	t.mu.Unlock()
}

// UpdateProgress updates the current progress value and optional status message.
// This triggers a progress notification to the client (subject to rate limiting).
// Returns an error if the notification fails to send.
func (t *Task) UpdateProgress(progress float64, message string) error {
	t.mu.Lock()
	t.progress = progress
	if message != "" {
		t.message = message
	}
	total := t.total
	t.mu.Unlock()

	// Send notification through manager
	totalPtr := &total
	if total == 0 {
		totalPtr = nil // Don't send total if not set
	}

	return t.manager.SendProgressNotification(t, progress, totalPtr, message)
}

// IncrementProgress increments progress by 1
func (t *Task) IncrementProgress(message string) error {
	t.mu.Lock()
	t.progress++
	progress := t.progress
	t.mu.Unlock()

	return t.UpdateProgress(progress, message)
}

// Complete marks the task as completed successfully.
// This sends a final progress notification and removes the task from the manager.
func (t *Task) Complete() {
	t.mu.Lock()
	if t.endTime == nil {
		now := time.Now()
		t.endTime = &now
	}
	t.mu.Unlock()

	// Send final progress notification
	t.mu.RLock()
	progress := t.progress
	total := t.total
	message := t.message
	if message == "" {
		message = "Task completed"
	}
	t.mu.RUnlock()

	totalPtr := &total
	if total == 0 {
		totalPtr = &progress // Set total = progress for 100%
	}

	_ = t.manager.SendProgressNotification(t, progress, totalPtr, message)

	// Remove from manager
	t.manager.RemoveTask(t)
}

// CompleteWithError marks the task as failed with the given error.
// This sends an error notification and removes the task from the manager.
func (t *Task) CompleteWithError(err error) {
	t.mu.Lock()
	t.error = err
	if t.endTime == nil {
		now := time.Now()
		t.endTime = &now
	}
	t.mu.Unlock()

	// Send error notification
	t.mu.RLock()
	progress := t.progress
	t.mu.RUnlock()

	_ = t.manager.SendProgressNotification(t, progress, nil, "Error: "+err.Error())

	// Remove from manager
	t.manager.RemoveTask(t)
}

// Cancel cancels the task with the specified reason.
// This cancels the task's context, sends a cancellation notification,
// and removes the task from the manager. Subsequent calls are no-ops.
func (t *Task) Cancel(reason string) {
	t.mu.Lock()
	if t.cancelled {
		t.mu.Unlock()
		return
	}

	t.cancelled = true
	t.cancelReason = reason
	if t.endTime == nil {
		now := time.Now()
		t.endTime = &now
	}
	t.mu.Unlock()

	// Cancel context
	t.cancel()

	// Send cancellation notification
	t.mu.RLock()
	progress := t.progress
	t.mu.RUnlock()

	_ = t.manager.SendProgressNotification(t, progress, nil, "Cancelled: "+reason)

	// Remove from manager
	t.manager.RemoveTask(t)
}

// IsCancelled returns whether the task has been cancelled
func (t *Task) IsCancelled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cancelled
}

// GetProgress returns current progress and total
func (t *Task) GetProgress() (progress, total float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.progress, t.total
}

// GetMessage returns the current progress message
func (t *Task) GetMessage() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.message
}

// GetError returns any error that occurred
func (t *Task) GetError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.error
}

// Duration returns how long the task has been running
func (t *Task) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.endTime != nil {
		return t.endTime.Sub(t.startTime)
	}
	return time.Since(t.startTime)
}

// IsComplete returns whether the task has finished (successfully or not)
func (t *Task) IsComplete() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.endTime != nil
}
