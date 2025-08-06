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

func TestTaskManager_RegistersAndRemovesTask_When_LifecycleIsManaged(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	progressToken := mcp.ProgressToken("test-token-123")
	sessionID := "session-123"

	task, taskCtx := manager.StartTask(ctx, progressToken, sessionID)

	assert.NotNil(t, task)
	assert.Equal(t, "test-token-123", task.ID())
	assert.Equal(t, sessionID, task.SessionID())
	assert.NotNil(t, taskCtx)
	assert.False(t, task.IsComplete())
	assert.False(t, task.IsCancelled())
	assert.Equal(t, 1, manager.GetActiveTaskCount())
	assert.Equal(t, 1, manager.GetSessionTaskCount(sessionID))
	retrievedTask := manager.GetTask(progressToken)
	assert.Equal(t, task, retrievedTask)

	task.Complete()

	assert.Equal(t, 0, manager.GetActiveTaskCount())
	assert.Nil(t, manager.GetTask(progressToken))
}

func TestTask_UpdatesProgress_When_ProgressIsReported(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	task.SetTotal(100)
	err := task.UpdateProgress(25, "Processing first batch")
	assert.NoError(t, err)

	progress, total := task.GetProgress()
	assert.Equal(t, float64(25), progress)
	assert.Equal(t, float64(100), total)
	assert.Equal(t, "Processing first batch", task.GetMessage())

	err = task.IncrementProgress("Processing item")
	assert.NoError(t, err)

	progress, _ = task.GetProgress()
	assert.Equal(t, float64(26), progress)
}

func TestTask_CancelsAndCleansUp_When_CancelIsCalled(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	task, taskCtx := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")

	task.Cancel("User requested cancellation")

	assert.True(t, task.IsCancelled())
	assert.True(t, task.IsComplete())

	select {
	case <-taskCtx.Done():
		// Good
	default:
		t.Fatal("Task context should be cancelled")
	}

	assert.Equal(t, 0, manager.GetActiveTaskCount())
}

func TestTaskManager_CancelsAllTasks_When_SessionIsTerminated(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	sessionID := "session-123"

	task1, _ := manager.StartTask(ctx, mcp.ProgressToken("task1"), sessionID)
	task2, _ := manager.StartTask(ctx, mcp.ProgressToken("task2"), sessionID)
	task3, _ := manager.StartTask(ctx, mcp.ProgressToken("task3"), sessionID)

	assert.Equal(t, 3, manager.GetSessionTaskCount(sessionID))

	manager.CancelSessionTasks(sessionID)

	assert.True(t, task1.IsCancelled())
	assert.True(t, task2.IsCancelled())
	assert.True(t, task3.IsCancelled())
	assert.Equal(t, 0, manager.GetSessionTaskCount(sessionID))
	assert.Equal(t, 0, manager.GetActiveTaskCount())
}

func TestProgressReporter_ReportsFormattedProgress_When_HelperMethodsAreUsed(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")
	task.SetTotal(100)

	reporter := NewProgressReporter(task)
	reporter.SetUpdateInterval(10 * time.Millisecond)

	err := reporter.ReportPercentage(25, 100, "Quarter done")
	assert.NoError(t, err)
	err = reporter.ReportItems(5, 10, "files")
	assert.NoError(t, err)
	err = reporter.ReportBytes(1024*1024, 10*1024*1024)
	assert.NoError(t, err)

	reporter.Complete()
	assert.True(t, task.IsComplete())
}

func TestStepTracker_ReportsProgress_When_NavigatingSteps(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")
	tracker := NewStepTracker(task, 5)

	steps := []string{"Initialize", "Load data", "Process", "Validate", "Complete"}
	for _, step := range steps {
		err := tracker.NextStep(step)
		assert.NoError(t, err)
	}

	tracker.Complete()
	assert.True(t, task.IsComplete())
}

func TestItemProcessor_ReportsProgress_When_ProcessingItems(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	task, _ := manager.StartTask(ctx, mcp.ProgressToken("test"), "session")
	processor := NewItemProcessor(task, 3, "users")

	err := processor.ProcessItem()
	assert.NoError(t, err)
	err = processor.ProcessItemWithName("John Doe")
	assert.NoError(t, err)
	err = processor.ProcessItemWithName("Jane Smith")
	assert.NoError(t, err)

	processor.Complete()
	assert.True(t, task.IsComplete())
}

func TestCancellationHandler_CancelsTask_When_NotificationIsReceived(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	handler := NewCancellationHandler(manager)
	ctx := context.Background()
	requestID := mcp.NewRequestId("request-123")
	progressToken := mcp.ProgressToken("request-123")
	task, _ := manager.StartTask(ctx, progressToken, "session")

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

	err := handler.Handle(notification.Notification)
	assert.NoError(t, err)
	assert.True(t, task.IsCancelled())
}

func TestRunWithProgress_HandlesBothCases_When_TokenIsPresentOrAbsent(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()
	sessionID := "session"

	// Test function that uses progress
	testFunc := func(ctx context.Context, task *Task) (*mcp.CallToolResult, error) {
		if task != nil {
			task.SetTotal(2)
			_ = task.UpdateProgress(1, "Half done")
			_ = task.UpdateProgress(2, "Complete")
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Success"}},
		}, nil
	}

	t.Run("RunWithProgress_CreatesTask_When_TokenIsPresent", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Meta: &mcp.Meta{ProgressToken: "test-token"},
			},
		}
		result, err := RunWithProgress(ctx, req, manager, sessionID, testFunc)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("RunWithProgress_RunsSynchronously_When_TokenIsAbsent", func(t *testing.T) {
		req2 := mcp.CallToolRequest{Params: mcp.CallToolParams{}}
		result2, err2 := RunWithProgress(ctx, req2, manager, sessionID, testFunc)
		require.NoError(t, err2)
		assert.NotNil(t, result2)
	})
}
