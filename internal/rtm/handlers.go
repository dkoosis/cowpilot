package rtm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Handler manages RTM integration for the MCP server.
// It wraps an RTM client and provides tool handlers for MCP operations.
type Handler struct {
	// client is the underlying RTM API client
	client *Client
	// searchCache holds the last search results for pagination
	searchCache *searchResultCache
}

// searchResultCache stores search results for pagination
type searchResultCache struct {
	query     string
	tasks     []Task
	timestamp time.Time
}

// Constants for pagination
const (
	defaultPageSize = 25
	maxPageSize     = 100
	cacheTTL        = 5 * time.Minute
)

// NewHandler creates an RTM handler with credentials from environment variables.
// Requires RTM_API_KEY and RTM_API_SECRET environment variables to be set.
// Returns nil if credentials are missing, allowing graceful degradation.
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

// SetAuthToken sets the RTM auth token on the underlying client.
// This is typically called after successful OAuth authentication.
func (h *Handler) SetAuthToken(token string) {
	h.client.AuthToken = token
}

// GetClient returns the underlying RTM client for direct API access.
// Useful for accessing RTM functionality not exposed through handler methods.
func (h *Handler) GetClient() *Client {
	return h.client
}

// SetupTools registers RTM-related tools with the MCP server.
// This includes tools for authentication, task management, list operations,
// and search functionality. If RTM_AUTH_TOKEN is set in the environment,
// it will be used for immediate authentication.
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

	// rtm_search - Enhanced task search with pagination
	s.AddTool(mcp.NewTool("rtm_search",
		mcp.WithDescription("Search tasks with RTM's search syntax. Results are paginated."),
		mcp.WithString("query", mcp.Required(), mcp.Description("RTM search: 'dueBefore:tomorrow AND tag:work', 'list:Shopping', 'priority:1'")),
		mcp.WithString("include_completed", mcp.Description("Include completed tasks in results (true/false)")),
		mcp.WithNumber("page", mcp.Description("Page number (1-based, default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Results per page (default: 25, max: 100)")),
		mcp.WithString("use_cache", mcp.Description("Use cached results if available (true/false, default: true)")),
	), h.handleSearch)

	// rtm_quick_add - Primary task creation tool using Smart Add
	s.AddTool(mcp.NewTool("rtm_quick_add",
		mcp.WithDescription("Add a task using RTM's Smart Add syntax. Supports natural language for due dates, priorities, lists, and tags."),
		mcp.WithString("task", mcp.Required(), mcp.Description("Task in Smart Add format: 'Buy milk tomorrow !2 #shopping ^Tuesday =30min @store'")),
		mcp.WithString("parse_only", mcp.Description("If true, only parse and return the interpretation without adding (true/false)")),
	), h.handleQuickAdd)

	// rtm_update - Update task properties
	s.AddTool(mcp.NewTool("rtm_update",
		mcp.WithDescription("Update task properties. Only specify fields to change."),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to update")),
		mcp.WithString("series_id", mcp.Required(), mcp.Description("Task series ID")),
		mcp.WithString("list_id", mcp.Required(), mcp.Description("List ID containing the task")),
		mcp.WithString("name", mcp.Description("New task name")),
		mcp.WithString("due", mcp.Description("Natural language date/time (e.g., 'tomorrow', '2pm Friday')")),
		mcp.WithString("priority", mcp.Description("Priority: 1 (high), 2 (medium), 3 (low), or N (none)")),
		mcp.WithString("estimate", mcp.Description("Time estimate (e.g., '30 min', '2 hours')")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
		mcp.WithString("list_name", mcp.Description("Move to different list by name")),
	), h.handleUpdateTask)

	// rtm_complete - Mark task(s) as complete
	s.AddTool(mcp.NewTool("rtm_complete",
		mcp.WithDescription("Mark one or more tasks as complete"),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID or comma-separated IDs")),
		mcp.WithString("series_id", mcp.Required(), mcp.Description("Task series ID or comma-separated IDs")),
		mcp.WithString("list_id", mcp.Required(), mcp.Description("List ID or comma-separated IDs")),
	), h.handleComplete)

	// rtm_manage_list - List management
	s.AddTool(mcp.NewTool("rtm_manage_list",
		mcp.WithDescription("Create, rename, or archive lists"),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: create, rename, archive, unarchive")),
		mcp.WithString("name", mcp.Description("List name (required for create/rename)")),
		mcp.WithString("new_name", mcp.Description("New name for rename action")),
		mcp.WithString("list_id", mcp.Description("List ID for archive/unarchive actions")),
	), h.handleManageList)
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

func (h *Handler) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("search query is required"), nil
	}

	// Parse pagination params
	page := 1
	if pageFloat, ok := args["page"].(float64); ok {
		page = int(pageFloat)
		if page < 1 {
			page = 1
		}
	}

	pageSize := defaultPageSize
	if pageSizeFloat, ok := args["page_size"].(float64); ok {
		pageSize = int(pageSizeFloat)
		if pageSize < 1 {
			pageSize = defaultPageSize
		}
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
	}

	useCache := true
	if useCacheStr, ok := args["use_cache"].(string); ok {
		useCache = useCacheStr != "false"
	}

	includeCompleted := false
	if includeCompletedStr, ok := args["include_completed"].(string); ok {
		includeCompleted = includeCompletedStr == "true"
	}
	if includeCompleted {
		query = "(" + query + ") OR (" + query + " AND completed:within \"1 week\")"
	}

	// Check cache validity
	var tasks []Task
	if useCache && h.searchCache != nil &&
		h.searchCache.query == query &&
		time.Since(h.searchCache.timestamp) < cacheTTL {
		// Use cached results
		tasks = h.searchCache.tasks
	} else {
		// Fetch new results
		var err error
		tasks, err = h.client.GetTasks(query, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search tasks: %v", err)), nil
		}
		// Update cache
		h.searchCache = &searchResultCache{
			query:     query,
			tasks:     tasks,
			timestamp: time.Now(),
		}
	}

	// Calculate pagination
	totalTasks := len(tasks)
	totalPages := (totalTasks + pageSize - 1) / pageSize
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > totalTasks {
		endIdx = totalTasks
	}

	var pagedTasks []Task
	if startIdx < totalTasks {
		pagedTasks = tasks[startIdx:endIdx]
	}

	// Enhanced result with pagination metadata
	result := map[string]interface{}{
		"query":       query,
		"total_found": totalTasks,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"has_more":    page < totalPages,
		"tasks":       pagedTasks,
		"search_time": time.Now().Format("2006-01-02 15:04:05"),
		"cache_used":  useCache && h.searchCache != nil && h.searchCache.query == query,
	}

	if totalTasks > pageSize {
		result["pagination_tip"] = fmt.Sprintf("Showing tasks %d-%d of %d. Use page parameter to navigate.", startIdx+1, endIdx, totalTasks)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format search results"), nil
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

func (h *Handler) handleQuickAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	taskText, ok := args["task"].(string)
	if !ok || taskText == "" {
		return mcp.NewToolResultError("Task text is required"), nil
	}

	parseOnly := false
	if parseOnlyStr, ok := args["parse_only"].(string); ok {
		parseOnly = parseOnlyStr == "true"
	}

	if parseOnly {
		// Return what would be parsed without adding
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Smart Add would interpret: '%s'\n\nNote: RTM's Smart Add parsing happens on the server. Examples:\n- 'Buy milk tomorrow !2' = high priority, due tomorrow\n- 'Meeting @office #work ^Monday 2pm' = tagged work, location office, Monday 2pm\n- 'Review report =1hour' = 1 hour time estimate", taskText),
				},
			},
		}, nil
	}

	// Use Smart Add - RTM's addTask API supports Smart Add syntax
	task, err := h.client.AddTask(taskText, "")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add task: %v", err)), nil
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format task"), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Task added using Smart Add:\n%s\n\nOriginal: %s", data, taskText),
			},
		},
	}, nil
}

func (h *Handler) handleComplete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	listIDs, ok1 := args["list_id"].(string)
	seriesIDs, ok2 := args["series_id"].(string)
	taskIDs, ok3 := args["task_id"].(string)

	if !ok1 || !ok2 || !ok3 {
		return mcp.NewToolResultError("list_id, series_id, and task_id are required"), nil
	}

	// Support comma-separated IDs for bulk operations
	listIDList := strings.Split(listIDs, ",")
	seriesIDList := strings.Split(seriesIDs, ",")
	taskIDList := strings.Split(taskIDs, ",")

	if len(listIDList) != len(seriesIDList) || len(seriesIDList) != len(taskIDList) {
		return mcp.NewToolResultError("list_id, series_id, and task_id must have same number of comma-separated values"), nil
	}

	var completed []string
	var failed []string

	for i := 0; i < len(taskIDList); i++ {
		err := h.client.CompleteTask(strings.TrimSpace(listIDList[i]), strings.TrimSpace(seriesIDList[i]), strings.TrimSpace(taskIDList[i]))
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", taskIDList[i], err))
		} else {
			completed = append(completed, taskIDList[i])
		}
	}

	result := fmt.Sprintf("Completed %d task(s)", len(completed))
	if len(failed) > 0 {
		result += fmt.Sprintf("\nFailed: %v", failed)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

func (h *Handler) handleUpdateTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	updates := make(map[string]string)
	var messages []string

	// Check each optional field
	if name, ok := args["name"].(string); ok && name != "" {
		updates["name"] = name
		messages = append(messages, "name updated")
	}

	if due, ok := args["due"].(string); ok && due != "" {
		updates["due"] = due
		messages = append(messages, "due date updated")
	}

	if priority, ok := args["priority"].(string); ok && priority != "" {
		updates["priority"] = priority
		messages = append(messages, "priority updated")
	}

	if estimate, ok := args["estimate"].(string); ok && estimate != "" {
		updates["estimate"] = estimate
		messages = append(messages, "time estimate updated")
	}

	if tags, ok := args["tags"].(string); ok && tags != "" {
		updates["tags"] = tags
		messages = append(messages, "tags updated")
	}

	if listName, ok := args["list_name"].(string); ok && listName != "" {
		updates["list"] = listName
		messages = append(messages, "moved to different list")
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No updates specified. Provide at least one field to update."), nil
	}

	// Apply updates using RTM API
	err := h.client.UpdateTask(listID, seriesID, taskID, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Task updated: %s", strings.Join(messages, ", ")),
			},
		},
	}, nil
}

func (h *Handler) handleManageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	if h.client.AuthToken == "" {
		return mcp.NewToolResultError("RTM authentication required. Use rtm_auth_url first."), nil
	}

	action, ok := args["action"].(string)
	if !ok || action == "" {
		return mcp.NewToolResultError("action is required"), nil
	}

	switch action {
	case "create":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required for create action"), nil
		}

		list, err := h.client.CreateList(name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create list: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("List '%s' created with ID: %s", name, list.ID),
				},
			},
		}, nil

	case "rename":
		listID, ok1 := args["list_id"].(string)
		newName, ok2 := args["new_name"].(string)
		if !ok1 || !ok2 || listID == "" || newName == "" {
			return mcp.NewToolResultError("list_id and new_name are required for rename action"), nil
		}

		err := h.client.RenameList(listID, newName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to rename list: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("List renamed to '%s'", newName),
				},
			},
		}, nil

	case "archive", "unarchive":
		listID, ok := args["list_id"].(string)
		if !ok || listID == "" {
			return mcp.NewToolResultError("list_id is required for archive/unarchive action"), nil
		}

		archive := action == "archive"
		err := h.client.ArchiveList(listID, archive)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to %s list: %v", action, err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("List %sd", action),
				},
			},
		}, nil

	default:
		return mcp.NewToolResultError("Invalid action. Use: create, rename, archive, or unarchive"), nil
	}
}
