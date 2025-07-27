package rtm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Handler manages RTM integration
type Handler struct {
	client *Client
}

// NewHandler creates RTM handler with credentials from env
func NewHandler() *Handler {
	apiKey := os.Getenv("RTM_API_KEY")
	secret := os.Getenv("RTM_API_SECRET")

	if apiKey == "" || secret == "" {
		return nil // RTM tools won't be registered
	}

	return &Handler{
		client: NewClient(apiKey, secret),
	}
}

// SetAuthToken sets the RTM auth token
func (h *Handler) SetAuthToken(token string) {
	h.client.AuthToken = token
}

// SetupTools registers RTM-related tools
func (h *Handler) SetupTools(s *server.MCPServer) {
	// Check auth token from env (for testing)
	if token := os.Getenv("RTM_AUTH_TOKEN"); token != "" {
		h.client.AuthToken = token
	}

	// rtm_auth_url - Get authentication URL
	s.AddTool(mcp.NewTool("rtm_auth_url",
		mcp.WithDescription("Generate RTM authentication URL"),
		mcp.WithString("permissions", mcp.Required(), mcp.Description("Permissions level: read, write, or delete")),
	), h.handleAuthURL)

	// rtm_lists - Get all RTM lists
	s.AddTool(mcp.NewTool("rtm_lists",
		mcp.WithDescription("Get all Remember The Milk lists"),
	), h.handleGetLists)

	// rtm_tasks - Get tasks with optional filter
	s.AddTool(mcp.NewTool("rtm_tasks",
		mcp.WithDescription("Get Remember The Milk tasks"),
		mcp.WithString("filter", mcp.Description("RTM Smart Add filter (e.g., 'due:today', 'list:Inbox')")),
		mcp.WithString("list_id", mcp.Description("Filter by specific list ID")),
	), h.handleGetTasks)

	// rtm_add_task - Add new task
	s.AddTool(mcp.NewTool("rtm_add_task",
		mcp.WithDescription("Add a new task to Remember The Milk"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Task name (supports Smart Add syntax)")),
		mcp.WithString("list_id", mcp.Description("List ID to add task to (default: Inbox)")),
	), h.handleAddTask)

	// rtm_complete_task - Mark task as complete
	s.AddTool(mcp.NewTool("rtm_complete_task",
		mcp.WithDescription("Mark a Remember The Milk task as complete"),
		mcp.WithString("list_id", mcp.Required(), mcp.Description("List ID containing the task")),
		mcp.WithString("series_id", mcp.Required(), mcp.Description("Task series ID")),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to complete")),
	), h.handleCompleteTask)
}

func (h *Handler) handleAuthURL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}
	perms, ok := args["permissions"].(string)
	if !ok {
		perms = "read"
	}

	url := h.client.AuthURL(perms)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Please visit this URL to authenticate:\n%s\n\nAfter authentication, the RTM_AUTH_TOKEN environment variable needs to be set.", url),
			},
		},
	}, nil
}

func (h *Handler) handleGetLists(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	lists, err := h.client.GetLists()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get lists: %v", err)), nil
	}

	// Format as JSON
	data, err := json.MarshalIndent(lists, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format lists"), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (h *Handler) handleGetTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	filter, _ := args["filter"].(string)
	listID, _ := args["list_id"].(string)

	tasks, err := h.client.GetTasks(filter, listID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get tasks: %v", err)), nil
	}

	// Format as JSON
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format tasks"), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (h *Handler) handleAddTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Task name is required"), nil
	}

	listID, _ := args["list_id"].(string)

	task, err := h.client.AddTask(name, listID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add task: %v", err)), nil
	}

	// Format response
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format task"), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Task added successfully:\n%s", data),
			},
		},
	}, nil
}

func (h *Handler) handleCompleteTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	listID, ok1 := args["list_id"].(string)
	seriesID, ok2 := args["series_id"].(string)
	taskID, ok3 := args["task_id"].(string)

	if !ok1 || !ok2 || !ok3 {
		return mcp.NewToolResultError("list_id, series_id, and task_id are required"), nil
	}

	err := h.client.CompleteTask(listID, seriesID, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to complete task: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Task marked as complete",
			},
		},
	}, nil
}
