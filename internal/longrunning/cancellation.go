package longrunning

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
)

// CancellationHandler processes cancellation requests from clients
type CancellationHandler struct {
	manager *Manager
}

// NewCancellationHandler creates a handler for cancellation notifications
func NewCancellationHandler(manager *Manager) *CancellationHandler {
	return &CancellationHandler{
		manager: manager,
	}
}

// Handle processes a cancellation notification
func (h *CancellationHandler) Handle(notification mcp.Notification) error {
	// Parse cancellation params
	paramsMap, ok := notification.Params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid cancellation params type: %T", notification.Params)
	}

	// Extract request ID and reason
	requestID, ok := paramsMap["requestId"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid requestId in cancellation params")
	}

	reason, _ := paramsMap["reason"].(string)
	if reason == "" {
		reason = "Cancelled by client"
	}

	log.Printf("Received cancellation for request %s: %s", requestID, reason)

	// Find task by request ID
	// Note: In a real implementation, we need to map request IDs to progress tokens
	// For now, we'll treat the request ID as the progress token
	progressToken := mcp.ProgressToken(requestID)

	task := h.manager.GetTask(progressToken)
	if task == nil {
		log.Printf("No task found for cancellation request: %s", requestID)
		return nil // Not an error - task might have already completed
	}

	// Cancel the task
	task.Cancel(reason)

	return nil
}

// ExtractProgressToken extracts the progress token from request metadata
func ExtractProgressToken(meta interface{}) mcp.ProgressToken {
	if meta == nil {
		return nil
	}

	// Handle different meta types
	switch m := meta.(type) {
	case map[string]interface{}:
		if token, exists := m["progressToken"]; exists {
			return token
		}
	case *map[string]interface{}:
		if m != nil {
			if token, exists := (*m)["progressToken"]; exists {
				return token
			}
		}
	}

	return nil
}

// WithProgress wraps a context with progress tracking if a token is provided
func WithProgress(ctx context.Context, req mcp.CallToolRequest, manager *Manager, sessionID string) (context.Context, *Task, bool) {
	// Extract progress token from meta
	var progressToken mcp.ProgressToken

	// Check if Meta exists and has progressToken
	if req.Params.Meta != nil {
		if metaMap, ok := req.Params.Meta.(map[string]interface{}); ok {
			if token, exists := metaMap["progressToken"]; exists {
				progressToken = token
			}
		}
	}

	// No progress token - run synchronously
	if progressToken == nil {
		return ctx, nil, false
	}

	// Create tracked task
	task, taskCtx := manager.StartTask(ctx, progressToken, sessionID)

	return taskCtx, task, true
}

// RunWithProgress executes a function with optional progress tracking
func RunWithProgress(ctx context.Context, req mcp.CallToolRequest, manager *Manager, sessionID string,
	fn func(context.Context, *Task) (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {

	taskCtx, task, hasProgress := WithProgress(ctx, req, manager, sessionID)

	if !hasProgress {
		// No progress tracking - run directly
		return fn(ctx, nil)
	}

	// Run with progress tracking
	defer task.Complete()

	// Execute function
	result, err := fn(taskCtx, task)

	// Handle errors
	if err != nil {
		task.CompleteWithError(err)
		return nil, err
	}

	return result, nil
}

// CheckCancellation checks if the context has been cancelled and returns appropriate error
func CheckCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("operation cancelled")
		}
		return ctx.Err()
	default:
		return nil
	}
}

// PeriodicallyCheckCancellation returns a function that checks for cancellation at intervals
func PeriodicallyCheckCancellation(ctx context.Context, interval int) func() error {
	count := 0
	return func() error {
		count++
		if count%interval == 0 {
			return CheckCancellation(ctx)
		}
		return nil
	}
}
