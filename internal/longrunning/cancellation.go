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
	// FIX: Access the AdditionalFields map for cancellation-specific parameters.
	additionalFields := notification.Params.AdditionalFields
	if additionalFields == nil {
		return fmt.Errorf("invalid cancellation notification: AdditionalFields is nil")
	}

	rawRequestID, ok1 := additionalFields["requestId"]
	requestID, ok2 := rawRequestID.(string)

	if !ok1 || !ok2 {
		return fmt.Errorf("invalid cancellation notification: requestId is missing or not a string")
	}

	var reason string
	if rawReason, ok := additionalFields["reason"]; ok {
		reason, _ = rawReason.(string) // It's okay if reason is missing.
	}
	if reason == "" {
		reason = "Cancelled by client"
	}

	log.Printf("Received cancellation for request %s: %s", requestID, reason)

	progressToken := mcp.ProgressToken(requestID)
	task := h.manager.GetTask(progressToken)
	if task == nil {
		log.Printf("No task found for cancellation request: %s", requestID)
		return nil
	}

	task.Cancel(reason)
	return nil
}

// ExtractProgressToken extracts the progress token from request metadata
func ExtractProgressToken(meta *mcp.Meta) mcp.ProgressToken {
	if meta == nil {
		return nil
	}
	return meta.ProgressToken
}

// WithProgress wraps a context with progress tracking if a token is provided
func WithProgress(ctx context.Context, req mcp.CallToolRequest, manager *Manager, sessionID string) (context.Context, *Task, bool) {
	var progressToken mcp.ProgressToken

	// FIX: The Meta field is now a pointer to a struct. Check for nil before accessing.
	if req.Params.Meta != nil {
		progressToken = req.Params.Meta.ProgressToken
	}

	if progressToken == nil {
		return ctx, nil, false
	}

	task, taskCtx := manager.StartTask(ctx, progressToken, sessionID)
	return taskCtx, task, true
}

// RunWithProgress executes a function with optional progress tracking
func RunWithProgress(ctx context.Context, req mcp.CallToolRequest, manager *Manager, sessionID string,
	fn func(context.Context, *Task) (*mcp.CallToolResult, error)) (*mcp.CallToolResult, error) {

	taskCtx, task, hasProgress := WithProgress(ctx, req, manager, sessionID)

	if !hasProgress {
		return fn(ctx, nil)
	}

	defer task.Complete()

	result, err := fn(taskCtx, task)
	if err != nil {
		task.CompleteWithError(err)
		return nil, err
	}
	return result, err
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
