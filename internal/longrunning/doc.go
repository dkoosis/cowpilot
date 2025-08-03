// Package longrunning provides infrastructure for long-running tasks in MCP servers.
//
// This package implements the MCP protocol's progress notification and cancellation
// features, allowing tools to report progress and handle cancellations gracefully.
//
// Basic usage:
//
//	// Create task manager
//	taskManager := longrunning.NewManager(mcpServer)
//
//	// In your tool handler:
//	func handleTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	    return longrunning.RunWithProgress(ctx, req, taskManager, sessionID,
//	        func(ctx context.Context, task *Task) (*mcp.CallToolResult, error) {
//	            if task == nil {
//	                // No progress tracking - run synchronously
//	                return runToolSync(ctx)
//	            }
//
//	            // With progress tracking
//	            items := getItems()
//	            processor := NewItemProcessor(task, len(items), "items")
//
//	            for _, item := range items {
//	                if err := CheckCancellation(ctx); err != nil {
//	                    return nil, err
//	                }
//	                processItem(item)
//	                processor.ProcessItem()
//	            }
//
//	            return result, nil
//	        })
//	}
//
// The package handles:
// - Task lifecycle management
// - Progress reporting with rate limiting
// - Cancellation propagation
// - Session-based task cleanup
// - Graceful degradation when no progress token provided
package longrunning
