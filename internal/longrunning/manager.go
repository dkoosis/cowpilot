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

// Manager handles all long-running tasks in the MCP server
type Manager struct {
	server       *server.MCPServer
	tasks        map[string]*Task           // Progress token -> Task
	sessionTasks map[string]map[string]bool // Session ID -> Set of task IDs
	mu           sync.RWMutex

	// Configuration
	minNotificationInterval time.Duration
}

// NewManager creates a new task manager
func NewManager(mcpServer *server.MCPServer) *Manager {
	return &Manager{
		server:                  mcpServer,
		tasks:                   make(map[string]*Task),
		sessionTasks:            make(map[string]map[string]bool),
		minNotificationInterval: 100 * time.Millisecond, // Default rate limit
	}
}

// StartTask creates and registers a new tracked task
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

// GetTask retrieves a task by progress token
func (m *Manager) GetTask(progressToken mcp.ProgressToken) *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id := fmt.Sprintf("%v", progressToken)
	return m.tasks[id]
}

// RemoveTask unregisters a task
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

// CancelSessionTasks cancels all tasks for a session
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
	// Parse cancellation params
	params, ok := notification.Params.(mcp.CancelledNotificationParams)
	if !ok {
		log.Printf("Invalid cancellation notification params: %T", notification.Params)
		return
	}

	// Find task by request ID
	// Note: In a real implementation, we'd need to map request IDs to progress tokens
	// For now, we'll treat the request ID as the progress token
	progressToken := mcp.ProgressToken(params.RequestID)

	task := m.GetTask(progressToken)
	if task == nil {
		log.Printf("No task found for cancellation request: %s", params.RequestID)
		return
	}

	// Cancel the task
	reason := params.Reason
	if reason == "" {
		reason = "Cancelled by client"
	}
	task.Cancel(reason)
}

// SendProgressNotification sends a progress update to the client
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

	// Create progress notification
	params := mcp.ProgressNotificationParams{
		ProgressToken: task.progressToken,
		Progress:      progress,
		Total:         total,
	}

	notification := mcp.ProgressNotification{
		Notification: mcp.Notification{
			Method: "notifications/progress",
			Params: params,
		},
	}

	// Send to client
	// Note: The actual sending mechanism depends on the transport (SSE, WebSocket, etc.)
	// The mcp-go library should provide a way to send notifications
	// For now, we'll log it
	percentage := 100.0
	if total != nil && *total > 0 {
		percentage = (progress / *total) * 100
	} else if progress > 0 {
		percentage = (progress / progress) * 100 // 100% if no total set
	}
	log.Printf("Progress notification for task %s: %.1f%% - %s",
		task.id, percentage, message)

	// TODO: Implement actual notification sending when mcp-go supports it
	// m.server.SendNotificationToClient(notification)

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
