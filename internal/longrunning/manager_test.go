package longrunning

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMCPServer for testing
type MockMCPServer struct {
	notifications []interface{}
}

func (m *MockMCPServer) SendNotification(notification interface{}) error {
	m.notifications = append(m.notifications, notification)
	return nil
}

func TestTaskCreation(t *testing.T) {
	// Create manager with mock server
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	// Create task
	ctx := context.Background()
	progressToken := mcp.ProgressToken("test-token-123")
	sessionID := "session-123"

	task, taskCtx := manager.StartTask(ctx, progressToken, sessionID)

	// Verify task properties
	assert.NotNil(t, task)
	assert.Equal(t, "test-token-123", task.ID())
	assert.Equal(t, sessionID, task.SessionID())
	assert.NotNil(t, taskCtx)
	assert.False(t, task.IsComplete())
	assert.False(t, task.IsCancelled())

	// Verify task is registered
	assert.Equal(t, 1, manager.GetActiveTaskCount())
	assert.Equal(t, 1, manager.GetSessionTaskCount(sessionID))

	retrievedTask := manager.GetTask(progressToken)
	assert.Equal(t, task, retrievedTask)

	// Complete task
	task.Complete()

	// Verify task is removed
	assert.Equal(t, 0, manager.GetActiveTaskCount())
	assert.Nil(t, manager.GetTask(progressToken))
}

func TestProgressUpdates(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	// Set total
	task.SetTotal(100)

	// Update progress
	err := task.UpdateProgress(25, "Processing first batch")
	assert.NoError(t, err)

	progress, total := task.GetProgress()
	assert.Equal(t, float64(25), progress)
	assert.Equal(t, float64(100), total)
	assert.Equal(t, "Processing first batch", task.GetMessage())

	// Increment progress
	err = task.IncrementProgress("Processing item")
	assert.NoError(t, err)

	progress, _ = task.GetProgress()
	assert.Equal(t, float64(26), progress)
}

func TestTaskCancellation(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	task, taskCtx := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	// Cancel task
	task.Cancel("User requested cancellation")

	// Verify cancellation
	assert.True(t, task.IsCancelled())
	assert.True(t, task.IsComplete())

	// Context should be cancelled
	select {
	case <-taskCtx.Done():
		// Good
	default:
		t.Fatal("Task context should be cancelled")
	}

	// Task should be removed from manager
	assert.Equal(t, 0, manager.GetActiveTaskCount())
}

func TestSessionTaskManagement(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	sessionID := "session-123"

	// Create multiple tasks for session
	task1, _ := manager.StartTask(ctx, mcp.ProgressToken("task1"), sessionID)
	task2, _ := manager.StartTask(ctx, mcp.ProgressToken("task2"), sessionID)
	task3, _ := manager.StartTask(ctx, mcp.ProgressToken("task3"), sessionID)

	// Verify all tasks are tracked
	assert.Equal(t, 3, manager.GetSessionTaskCount(sessionID))

	// Cancel all session tasks
	manager.CancelSessionTasks(sessionID)

	// Verify all tasks are cancelled
	assert.True(t, task1.IsCancelled())
	assert.True(t, task2.IsCancelled())
	assert.True(t, task3.IsCancelled())

	// Verify tasks are removed
	assert.Equal(t, 0, manager.GetSessionTaskCount(sessionID))
	assert.Equal(t, 0, manager.GetActiveTaskCount())
}

func TestProgressReporter(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")
	task.SetTotal(100)

	reporter := NewProgressReporter(task)
	reporter.SetUpdateInterval(10 * time.Millisecond)

	// Report progress
	err := reporter.ReportPercentage(25, 100, "Quarter done")
	assert.NoError(t, err)

	// Report items
	err = reporter.ReportItems(5, 10, "files")
	assert.NoError(t, err)

	// Report bytes
	err = reporter.ReportBytes(1024*1024, 10*1024*1024)
	assert.NoError(t, err)

	// Complete
	reporter.Complete()
	assert.True(t, task.IsComplete())
}

func TestStepTracker(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	tracker := NewStepTracker(task, 5)

	// Process steps
	steps := []string{
		"Initialize",
		"Load data",
		"Process",
		"Validate",
		"Complete",
	}

	for _, step := range steps {
		err := tracker.NextStep(step)
		assert.NoError(t, err)
	}

	tracker.Complete()
	assert.True(t, task.IsComplete())
}

func TestItemProcessor(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	processor := NewItemProcessor(task, 3, "users")

	// Process items
	err := processor.ProcessItem()
	assert.NoError(t, err)

	err = processor.ProcessItemWithName("John Doe")
	assert.NoError(t, err)

	err = processor.ProcessItemWithName("Jane Smith")
	assert.NoError(t, err)

	processor.Complete()
	assert.True(t, task.IsComplete())
}

func TestCancellationHandler(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	handler := NewCancellationHandler(manager)

	// Create task
	ctx := context.Background()
	requestID := mcp.NewRequestId("request-123")
	progressToken := mcp.ProgressToken("request-123")
	task, _ := manager.StartTask(ctx, progressToken, "session")

	// Create cancellation notification
	// FIX: Construct the notification with the correct, strongly-typed parameter struct.
	notification := mcp.CancelledNotification{
		Notification: mcp.Notification{
			Method: "notifications/cancelled",
			Params: mcp.NotificationParams{
				AdditionalFields: map[string]any{
					"requestId": "request-123",
					"reason":    "User cancelled",
				},
			},
		},
		Params: mcp.CancelledNotificationParams{
			RequestId: requestID,
			Reason:    "User cancelled",
		},
	}

	// Handle cancellation
	err := handler.Handle(notification.Notification)
	assert.NoError(t, err)

	// Verify task is cancelled
	assert.True(t, task.IsCancelled())
}

func TestWithProgress(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	// Test with progress token
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Meta: &mcp.Meta{
				ProgressToken: "test-token",
			},
		},
	}

	ctx := context.Background()
	taskCtx, task, hasProgress := WithProgress(ctx, req, manager, "session")

	assert.True(t, hasProgress)
	assert.NotNil(t, task)
	assert.NotNil(t, taskCtx)

	// Test without progress token
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{},
	}

	taskCtx2, task2, hasProgress2 := WithProgress(ctx, req2, manager, "session")

	assert.False(t, hasProgress2)
	assert.Nil(t, task2)
	assert.Equal(t, ctx, taskCtx2)
}

func TestRunWithProgress(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)

	// Test function that uses progress
	testFunc := func(ctx context.Context, task *Task) (*mcp.CallToolResult, error) {
		if task != nil {
			task.SetTotal(2)
			_ = task.UpdateProgress(1, "Half done")
			_ = task.UpdateProgress(2, "Complete")
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Success",
				},
			},
		}, nil
	}

	// With progress token
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Meta: &mcp.Meta{
				ProgressToken: "test-token",
			},
		},
	}

	ctx := context.Background()
	result, err := RunWithProgress(ctx, req, manager, "session", testFunc)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	// Without progress token
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{},
	}

	result2, err2 := RunWithProgress(ctx, req2, manager, "session", testFunc)

	require.NoError(t, err2)
	assert.NotNil(t, result2)
}
