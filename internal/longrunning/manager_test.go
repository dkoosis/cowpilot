package longrunning

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskManagerLifecycle(t *testing.T) {
	t.Logf("Importance: This suite validates the fundamental lifecycle of asynchronous tasks (start, track, complete, cancel), which is critical for supporting long-running, non-blocking operations in tools.")
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()

	t.Run("registers and removes a task correctly", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the manager can correctly track a task from creation to completion, preventing memory leaks and orphaned operations.")
		progressToken := mcp.ProgressToken("test-token-123")
		sessionID := "session-123"

		task, _ := manager.StartTask(ctx, progressToken, sessionID)
		require.NotNil(t, task, "Task should be created")
		assert.Equal(t, 1, manager.GetActiveTaskCount(), "Active task count should be 1 after start")

		task.Complete()
		assert.Equal(t, 0, manager.GetActiveTaskCount(), "Active task count should be 0 after completion")
		assert.Nil(t, manager.GetTask(progressToken), "Completed task should be removed from manager")
	})

	t.Run("cancels a task and its context", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that client- or server-initiated cancellations correctly propagate and terminate the running operation, freeing up resources.")
		task, taskCtx := manager.StartTask(ctx, mcp.ProgressToken("cancellation-test"), "session-cancel")

		task.Cancel("User requested cancellation")
		assert.True(t, task.IsCancelled(), "Task should be marked as cancelled")
		assert.True(t, task.IsComplete(), "Cancelled task should be marked as complete")
		assert.Equal(t, 0, manager.GetActiveTaskCount(), "Cancelled task should be removed from active count")

		select {
		case <-taskCtx.Done():
			// This is the expected outcome
		default:
			t.Fatal("Task context should be cancelled after task.Cancel() is called")
		}
	})

	t.Run("cancels all tasks for a given session", func(t *testing.T) {
		t.Logf("  > Why it's important: A critical cleanup feature. Ensures that when a user session ends (e.g., client disconnects), all associated long-running tasks are terminated to prevent resource leaks.")
		sessionID := "session-to-cancel"
		task1, _ := manager.StartTask(ctx, mcp.ProgressToken("s-task1"), sessionID)
		task2, _ := manager.StartTask(ctx, mcp.ProgressToken("s-task2"), sessionID)

		require.Equal(t, 2, manager.GetSessionTaskCount(sessionID), "Should have 2 tasks for the session")

		manager.CancelSessionTasks(sessionID)
		assert.True(t, task1.IsCancelled(), "Task 1 should be cancelled")
		assert.True(t, task2.IsCancelled(), "Task 2 should be cancelled")
		assert.Equal(t, 0, manager.GetSessionTaskCount(sessionID), "Session task count should be 0 after cancellation")
		assert.Equal(t, 0, manager.GetActiveTaskCount(), "Total active task count should be 0")
	})

	t.Run("handles cancellation notifications from clients", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies the server correctly processes the 'notifications/cancelled' message from the MCP protocol, allowing clients to stop operations they initiated.")
		handler := NewCancellationHandler(manager)
		progressToken := mcp.ProgressToken("request-to-cancel-123")
		task, _ := manager.StartTask(ctx, progressToken, "session-notify-cancel")

		notification := mcp.Notification{
			Method: "notifications/cancelled",
			Params: mcp.NotificationParams{
				AdditionalFields: map[string]any{
					"requestId": "request-to-cancel-123",
					"reason":    "Client side cancellation",
				},
			},
		}

		err := handler.Handle(notification)
		require.NoError(t, err)
		assert.True(t, task.IsCancelled(), "Task should be cancelled via notification")
	})
}

func TestProgressReporting(t *testing.T) {
	t.Logf("Importance: This suite tests the progress reporting helpers (ProgressReporter, StepTracker, ItemProcessor), which provide a structured and rate-limited way for tools to communicate progress to clients.")
	mcpServer := server.NewMCPServer("test", "1.0")
	manager := NewManager(mcpServer)
	ctx := context.Background()

	t.Run("ProgressReporter correctly formats and sends updates", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the base progress reporting mechanism works, providing the foundation for all other progress helpers.")
		task, _ := manager.StartTask(ctx, mcp.ProgressToken("progress-reporter-test"), "session-pr")
		task.SetTotal(100)
		reporter := NewProgressReporter(task)

		err := reporter.ReportPercentage(50, 100, "Halfway there")
		require.NoError(t, err)

		progress, total := task.GetProgress()
		assert.Equal(t, float64(50), progress)
		assert.Equal(t, float64(100), total)
		assert.Contains(t, task.GetMessage(), "[50.0%] Halfway there")
	})

	t.Run("StepTracker correctly reports stepped progress", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies a common use case for progress reporting: a multi-step process. This ensures clear, sequential updates can be sent.")
		task, _ := manager.StartTask(ctx, mcp.ProgressToken("step-tracker-test"), "session-st")
		tracker := NewStepTracker(task, 3) // 3 total steps

		_ = tracker.NextStep("Initialization")
		progress, total := task.GetProgress()
		assert.Equal(t, float64(1), progress)
		assert.Equal(t, float64(3), total)
		assert.Contains(t, task.GetMessage(), "Step 1/3: Initialization")
	})

	t.Run("ItemProcessor correctly reports item-based progress", func(t *testing.T) {
		t.Logf("  > Why it's important: Validates another common pattern: iterating over a list of items. This ensures accurate progress for batch operations.")
		task, _ := manager.StartTask(ctx, mcp.ProgressToken("item-processor-test"), "session-ip")
		processor := NewItemProcessor(task, 10, "files")

		_ = processor.ProcessItemWithName("file1.txt")
		progress, total := task.GetProgress()
		assert.Equal(t, float64(1), progress)
		assert.Equal(t, float64(10), total)
		assert.Contains(t, task.GetMessage(), "Processing files: file1.txt (1 of 10)")
	})
}
