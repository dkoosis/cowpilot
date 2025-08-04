// File: internal/rtm/batch_handlers.go

package rtm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/mcp-adapters/internal/longrunning"
)

// SetupBatchTools adds RTM batch operation tools with progress support
func (h *Handler) SetupBatchTools(s *server.MCPServer, taskManager *longrunning.Manager) {
	// Need to store task manager reference for handlers
	handlerWithManager := &batchHandler{
		Handler:     h,
		taskManager: taskManager,
	}

	// Batch update due dates
	s.AddTool(mcp.NewTool("set_rtm_tasks_due_date",
		mcp.WithDescription("Batch update due dates for multiple tasks by position. Returns job ID for async processing."),
		mcp.WithString("positions", mcp.Required(), mcp.Description("Comma-separated numbers from search (1,3,7,11,19)")),
		mcp.WithString("due_date", mcp.Required(), mcp.Description("Natural language date (Wed, tomorrow, next Monday)")),
	), handlerWithManager.createBatchHandler(handlerWithManager.handleBatchSetDueDate))

	// Batch update priority
	s.AddTool(mcp.NewTool("set_rtm_tasks_priority",
		mcp.WithDescription("Batch update priority for tasks by position. Returns job ID for async processing."),
		mcp.WithString("positions", mcp.Required(), mcp.Description("Task position numbers")),
		mcp.WithString("priority", mcp.Required(), mcp.Description("1 (high), 2 (med), 3 (low), N (none)")),
	), handlerWithManager.createBatchHandler(handlerWithManager.handleBatchSetPriority))

	// Batch add tags
	s.AddTool(mcp.NewTool("add_rtm_tags_to_tasks",
		mcp.WithDescription("Add tags to multiple tasks. Returns job ID for async processing."),
		mcp.WithString("positions", mcp.Required(), mcp.Description("Task position numbers")),
		mcp.WithString("tags", mcp.Required(), mcp.Description("Comma-separated tags to add")),
	), handlerWithManager.createBatchHandler(handlerWithManager.handleBatchAddTags))

	// Batch complete tasks
	s.AddTool(mcp.NewTool("complete_rtm_tasks_batch",
		mcp.WithDescription("Mark multiple tasks complete by position. Returns job ID for async processing."),
		mcp.WithString("positions", mcp.Required(), mcp.Description("Task position numbers to complete")),
	), handlerWithManager.createBatchHandler(handlerWithManager.handleBatchComplete))

	// Check job status
	s.AddTool(mcp.NewTool("check_rtm_job_status",
		mcp.WithDescription("Check status of async batch operation. Shows progress and any failures."),
		mcp.WithString("job_id", mcp.Required(), mcp.Description("Job ID returned from batch operation")),
	), handlerWithManager.createJobStatusHandler())
}

// batchHandler wraps Handler with task manager
type batchHandler struct {
	*Handler
	taskManager *longrunning.Manager
}

// BatchOperation represents a batch operation function
type BatchOperation func(ctx context.Context, task *longrunning.Task, positions []int, args map[string]any) error

// createBatchHandler creates a handler that runs operations with progress tracking
func (h *batchHandler) createBatchHandler(operation BatchOperation) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		// Parse positions
		positionsStr, ok := args["positions"].(string)
		if !ok || positionsStr == "" {
			return mcp.NewToolResultError("positions parameter is required"), nil
		}

		positions, err := parsePositions(positionsStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid positions: %v", err)), nil
		}

		// Get session ID (would come from connection context in real implementation)
		sessionID := "default-session" // TODO: Get from connection context

		// Run with progress tracking
		result, err := longrunning.RunWithProgress(ctx, request, h.taskManager, sessionID,
			func(ctx context.Context, task *longrunning.Task) (*mcp.CallToolResult, error) {
				// If no progress tracking, run synchronously
				if task == nil {
					return h.runBatchSynchronously(ctx, positions, args, operation)
				}

				// Run asynchronously with progress
				jobID := task.ID()

				// Start operation in background
				go func() {
					defer task.Complete()
					if err := operation(ctx, task, positions, args); err != nil {
						task.CompleteWithError(err)
					}
				}()

				// Return job ID immediately
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Batch operation started\nJob ID: %s\nUse check_rtm_job_status to monitor progress", jobID),
						},
					},
				}, nil
			})

		return result, err
	}
}

// runBatchSynchronously runs the operation without progress tracking
func (h *batchHandler) runBatchSynchronously(ctx context.Context, positions []int, args map[string]any, operation BatchOperation) (*mcp.CallToolResult, error) {
	// For synchronous operation, we pass nil task since there's no progress tracking
	err := operation(ctx, nil, positions, args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Batch operation failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Batch operation completed successfully for %d tasks", len(positions)),
			},
		},
	}, nil
}

// Batch operation implementations

func (h *batchHandler) handleBatchSetDueDate(ctx context.Context, task *longrunning.Task, positions []int, args map[string]any) error {
	dueDate, _ := args["due_date"].(string)
	if dueDate == "" {
		return fmt.Errorf("due_date is required")
	}

	// Get cached tasks
	tasks, err := h.getCachedTasksByPositions(positions)
	if err != nil {
		return err
	}

	// Create processor only if we have progress tracking
	var processor *longrunning.ItemProcessor
	if task != nil {
		processor = longrunning.NewItemProcessor(task, len(tasks), "tasks")
	}

	for _, t := range tasks {
		// Check cancellation
		if err := longrunning.CheckCancellation(ctx); err != nil {
			return err
		}

		// Update task
		updates := map[string]string{"due": dueDate}
		err := h.client.UpdateTask(t.ListID, t.SeriesID, t.ID, updates)
		if err != nil {
			// Log error but continue
			// Log error but continue
			if task != nil {
				progress, _ := task.GetProgress()
				_ = task.UpdateProgress(progress, fmt.Sprintf("Failed to update task %s: %v", t.Name, err))
			}
		}

		// Report progress
		if processor != nil {
			_ = processor.ProcessItemWithName(t.Name)
		}

		// Rate limit (RTM API restriction)
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (h *batchHandler) handleBatchSetPriority(ctx context.Context, task *longrunning.Task, positions []int, args map[string]any) error {
	priority, _ := args["priority"].(string)
	if priority == "" {
		return fmt.Errorf("priority is required")
	}

	tasks, err := h.getCachedTasksByPositions(positions)
	if err != nil {
		return err
	}

	var processor *longrunning.ItemProcessor
	if task != nil {
		processor = longrunning.NewItemProcessor(task, len(tasks), "tasks")
	}

	for _, t := range tasks {
		if err := longrunning.CheckCancellation(ctx); err != nil {
			return err
		}

		updates := map[string]string{"priority": priority}
		err := h.client.UpdateTask(t.ListID, t.SeriesID, t.ID, updates)
		if err != nil && task != nil {
			progress, _ := task.GetProgress()
			_ = task.UpdateProgress(progress, fmt.Sprintf("Failed: %v", err))
		}

		if processor != nil {
			_ = processor.ProcessItemWithName(t.Name)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (h *batchHandler) handleBatchAddTags(ctx context.Context, task *longrunning.Task, positions []int, args map[string]any) error {
	tags, _ := args["tags"].(string)
	if tags == "" {
		return fmt.Errorf("tags are required")
	}

	tasks, err := h.getCachedTasksByPositions(positions)
	if err != nil {
		return err
	}

	var processor *longrunning.ItemProcessor
	if task != nil {
		processor = longrunning.NewItemProcessor(task, len(tasks), "tasks")
	}

	for _, t := range tasks {
		if err := longrunning.CheckCancellation(ctx); err != nil {
			return err
		}

		// Get existing tags and add new ones
		existingTags := "" // TODO: Get from task
		allTags := existingTags
		if allTags != "" {
			allTags += ","
		}
		allTags += tags

		updates := map[string]string{"tags": allTags}
		err := h.client.UpdateTask(t.ListID, t.SeriesID, t.ID, updates)
		if err != nil && task != nil {
			progress, _ := task.GetProgress()
			_ = task.UpdateProgress(progress, fmt.Sprintf("Failed: %v", err))
		}

		if processor != nil {
			_ = processor.ProcessItemWithName(t.Name)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (h *batchHandler) handleBatchComplete(ctx context.Context, task *longrunning.Task, positions []int, args map[string]any) error {
	tasks, err := h.getCachedTasksByPositions(positions)
	if err != nil {
		return err
	}

	var processor *longrunning.ItemProcessor
	if task != nil {
		processor = longrunning.NewItemProcessor(task, len(tasks), "tasks")
	}

	for _, t := range tasks {
		if err := longrunning.CheckCancellation(ctx); err != nil {
			return err
		}

		err := h.client.CompleteTask(t.ListID, t.SeriesID, t.ID)
		if err != nil && task != nil {
			progress, _ := task.GetProgress()
			_ = task.UpdateProgress(progress, fmt.Sprintf("Failed: %v", err))
		}

		if processor != nil {
			_ = processor.ProcessItemWithName(t.Name)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

// createJobStatusHandler creates a handler to check job status
func (h *batchHandler) createJobStatusHandler() func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}

		jobID, ok := args["job_id"].(string)
		if !ok || jobID == "" {
			return mcp.NewToolResultError("job_id is required"), nil
		}

		// Get task by ID
		task := h.taskManager.GetTask(mcp.ProgressToken(jobID))
		if task == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Job not found. It may have completed or expired.",
					},
				},
			}, nil
		}

		// Get task status
		progress, total := task.GetProgress()
		message := task.GetMessage()
		duration := task.Duration()

		var status string
		if task.IsComplete() {
			if task.GetError() != nil {
				status = fmt.Sprintf("Failed: %v", task.GetError())
			} else {
				status = "Completed"
			}
		} else if task.IsCancelled() {
			status = "Cancelled"
		} else {
			status = "In Progress"
		}

		// Format response
		response := fmt.Sprintf("Job Status: %s\n", status)
		percentage := 0.0
		if total > 0 {
			percentage = (progress / total) * 100
		}
		response += fmt.Sprintf("Progress: %.0f/%.0f (%.1f%%)\n", progress, total, percentage)
		response += fmt.Sprintf("Duration: %s\n", duration.Round(time.Second))
		if message != "" {
			response += fmt.Sprintf("Last Update: %s\n", message)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: response,
				},
			},
		}, nil
	}
}

// Helper functions

func parsePositions(positionsStr string) ([]int, error) {
	parts := strings.Split(positionsStr, ",")
	positions := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		var pos int
		if _, err := fmt.Sscanf(part, "%d", &pos); err != nil {
			return nil, fmt.Errorf("invalid position: %s", part)
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// getCachedTasksByPositions retrieves tasks from cache
func (h *batchHandler) getCachedTasksByPositions(_ []int) ([]Task, error) {
	// This would retrieve tasks from a cache populated by search_rtm_tasks_smart
	// For now, return an error
	return nil, fmt.Errorf("task cache not implemented - use search_rtm_tasks_smart first")
}
