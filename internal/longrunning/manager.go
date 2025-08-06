package longrunning

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Manager handles all long-running tasks in the MCP server.
// It provides task lifecycle management, progress tracking, and session-based cleanup.
type Manager struct {
	server       *server.MCPServer
	tasks        map[string]*Task           // Progress token -> Task
	sessionTasks map[string]map[string]bool // Session ID -> Set of task IDs
	mu           sync.RWMutex

	// Configuration
	minNotificationInterval time.Duration
}

// NewManager creates a new task manager for handling long-running operations.
// The mcpServer parameter is stored for future notification sending when supported.
func NewManager(mcpServer *server.MCPServer) *Manager {
	return &Manager{
		server:                  mcpServer,
		tasks:                   make(map[string]*Task),
		sessionTasks:            make(map[string]map[string]bool),
		minNotificationInterval: 100 * time.Millisecond, // Default rate limit
	}
}

// StartTask creates and registers a new tracked task with progress tracking.
// Returns the created task and a cancellable context for the operation.
// The task is automatically registered with the manager and tracked by session.
func (m *Manager) StartTask(ctx context.Context, progressToken mcp.ProgressToken, sessionID string) (*Task, context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create task with cancellable context
	taskCtx, cancel := context.WithCancel(ctx)

	task := &Task{
		id:            fmt.Sprintf("%v", progressToken), // Convert token to string ID
		progressToken: progressToken,
		sessionID:     sessionID,
		ctx:           taskCtx,
		cancel:        cancel,
		startTime:     time.Now(),
		manager:       m,
		lastNotified:  time.Time{},
	}

	// Register task
	m.tasks[task.id] = task

	// Track by session
	if m.sessionTasks[sessionID] == nil {
		m.sessionTasks[sessionID] = make(map[string]bool)
	}
	m.sessionTasks[sessionID][task.id] = true

	log.Printf("Started task %s for session %s", task.id, sessionID)

	return task, taskCtx
}

// GetTask retrieves a task by its progress token.
// Returns nil if no task exists with the given token.
func (m *Manager) GetTask(progressToken mcp.ProgressToken) *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id := fmt.Sprintf("%v", progressToken)
	return m.tasks[id]
}

// RemoveTask unregisters a task from the manager.
// This also removes the task from session tracking and cleans up empty sessions.
func (m *Manager) RemoveTask(task *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tasks, task.id)

	// Remove from session tracking
	if sessionTasks, exists := m.sessionTasks[task.sessionID]; exists {
		delete(sessionTasks, task.id)
		if len(sessionTasks) == 0 {
			delete(m.sessionTasks, task.sessionID)
		}
	}

	log.Printf("Removed task %s", task.id)
}

// CancelSessionTasks cancels all tasks associated with a given session ID.
// This is typically called when a client disconnects or a session ends.
func (m *Manager) CancelSessionTasks(sessionID string) {
	m.mu.RLock()
	taskIDs := make([]string, 0)
	if sessionTasks, exists := m.sessionTasks[sessionID]; exists {
		for taskID := range sessionTasks {
			taskIDs = append(taskIDs, taskID)
		}
	}
	m.mu.RUnlock()

	// Cancel tasks outside of lock
	for _, taskID := range taskIDs {
		m.mu.RLock()
		task := m.tasks[taskID]
		m.mu.RUnlock()

		if task != nil {
			task.Cancel("Session ended")
		}
	}
}

// HandleCancellation processes cancellation notifications from clients
func (m *Manager) HandleCancellation(notification mcp.Notification) {
	// Safely type assert AdditionalFields to map
	additionalFields, ok := notification.Params.AdditionalFields.(map[string]interface{})
	if !ok || additionalFields == nil {
		log.Printf("Invalid cancellation notification: AdditionalFields is not a map or is nil")
		return
	}

	rawRequestID, ok1 := additionalFields["requestId"]
	requestID, ok2 := rawRequestID.(string)

	if !ok1 || !ok2 {
		log.Printf("Invalid cancellation notification: requestId is missing or not a string")
		return
	}

	progressToken := mcp.ProgressToken(requestID)
	task := m.GetTask(progressToken)
	if task == nil {
		log.Printf("No task found for cancellation request: %s", requestID)
		return
	}

	var reason string
	if rawReason, ok := additionalFields["reason"]; ok {
		reason, _ = rawReason.(string)
	}
	if reason == "" {
		reason = "Cancelled by client"
	}
	task.Cancel(reason)
}

// SendProgressNotification sends a progress update notification to the client.
// Implements rate limiting to avoid overwhelming clients with updates.
// Returns nil if the notification was sent or skipped due to rate limiting.
func (m *Manager) SendProgressNotification(task *Task, progress float64, total *float64, message string) error {
	// Check rate limiting
	now := time.Now()
	task.mu.Lock()
	if now.Sub(task.lastNotified) < m.minNotificationInterval {
		task.mu.Unlock()
		return nil // Skip this notification
	}
	task.lastNotified = now
	task.mu.Unlock()

	percentage := 100.0
	if total != nil && *total > 0 {
		percentage = (progress / *total) * 100
	} else if progress > 0 && total == nil {
		log.Printf("Progress notification for task %s: %.1f - %s",
			task.id, progress, message)
		return nil
	}
	log.Printf("Progress notification for task %s: %.1f%% - %s",
		task.id, percentage, message)

	// TODO(vcto): Implement actual notification sending when mcp-go supports it

	return nil
}

// SetMinNotificationInterval configures the rate limiting for progress notifications
func (m *Manager) SetMinNotificationInterval(interval time.Duration) {
	m.minNotificationInterval = interval
}

// GetActiveTaskCount returns the number of active tasks
func (m *Manager) GetActiveTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tasks)
}

// GetSessionTaskCount returns the number of active tasks for a session
func (m *Manager) GetSessionTaskCount(sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionTasks, exists := m.sessionTasks[sessionID]; exists {
		return len(sessionTasks)
	}
	return 0
}
