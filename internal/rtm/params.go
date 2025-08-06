package rtm

import "encoding/json"

// Parameter structs for RTM tool handlers
// These structs define the expected parameters for each tool,
// providing type safety and preparing for future SDK migration.

// AuthURLParams for rtm_auth_url tool
type AuthURLParams struct {
	Permissions string `json:"permissions"`
}

// SearchParams for rtm_search tool
type SearchParams struct {
	Query            string  `json:"query"`
	IncludeCompleted string  `json:"include_completed,omitempty"`
	Page             float64 `json:"page,omitempty"`
	PageSize         float64 `json:"page_size,omitempty"`
	UseCache         string  `json:"use_cache,omitempty"`
}

// QuickAddParams for rtm_quick_add tool
type QuickAddParams struct {
	Task      string `json:"task"`
	ParseOnly string `json:"parse_only,omitempty"`
}

// CompleteParams for rtm_complete tool
type CompleteParams struct {
	TaskID   string `json:"task_id"`
	SeriesID string `json:"series_id"`
	ListID   string `json:"list_id"`
}

// UpdateTaskParams for rtm_update tool
type UpdateTaskParams struct {
	TaskID   string `json:"task_id"`
	SeriesID string `json:"series_id"`
	ListID   string `json:"list_id"`
	Name     string `json:"name,omitempty"`
	Due      string `json:"due,omitempty"`
	Priority string `json:"priority,omitempty"`
	Estimate string `json:"estimate,omitempty"`
	Tags     string `json:"tags,omitempty"`
	ListName string `json:"list_name,omitempty"`
}

// ManageListParams for rtm_manage_list tool
type ManageListParams struct {
	Action  string `json:"action"`
	Name    string `json:"name,omitempty"`
	NewName string `json:"new_name,omitempty"`
	ListID  string `json:"list_id,omitempty"`
}

// Helper function to parse params from generic map
func parseParams[T any](args interface{}) (*T, error) {
	// Convert map[string]any to JSON then to struct
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	var params T
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return &params, nil
}
