// Package rtm provides a client and handlers for integrating with Remember The Milk API.
// It includes OAuth authentication adapters, task management operations, and batch processing capabilities.
package rtm

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// RTMError represents an RTM API error
type RTMError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *RTMError) Error() string {
	return fmt.Sprintf("RTM API error %d: %s", e.Code, e.Msg)
}

// Client handles RTM API communication
type Client struct {
	// APIKey is the RTM API key for the application
	APIKey string
	// Secret is the shared secret for signing API requests
	Secret string
	// AuthToken is the user's authentication token (obtained via OAuth)
	AuthToken string
	// BaseURL is the RTM API endpoint (default: https://api.rememberthemilk.com/services/rest/)
	BaseURL string
	// client is the HTTP client used for API requests
	client *http.Client
}

// NewClient creates a new RTM API client with the specified API key and secret.
// The client uses these credentials to sign API requests but requires an auth token
// (obtained via OAuth flow) before making authenticated API calls.
// Returns a configured client with a 10-second HTTP timeout.
func NewClient(apiKey, secret string) *Client {
	return &Client{
		APIKey:  apiKey,
		Secret:  secret,
		BaseURL: "https://api.rememberthemilk.com/services/rest/",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AuthURL generates the RTM authentication URL for the OAuth flow.
// The perms parameter specifies the permission level: "read", "write", or "delete".
// Returns a URL that users should visit to authorize the application.
func (c *Client) AuthURL(perms string) string {
	params := map[string]string{
		"api_key": c.APIKey,
		"perms":   perms, // read, write, or delete
	}
	sig := c.sign(params)

	u, _ := url.Parse("https://www.rememberthemilk.com/services/auth/")
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("api_sig", sig)
	u.RawQuery = q.Encode()

	return u.String()
}

// GetFrob gets an authentication frob from RTM.
// A frob is a temporary token used in RTM's authentication flow that must be
// authorized by the user before it can be exchanged for an auth token.
// Returns the frob string or an error if the API call fails.
func (c *Client) GetFrob() (string, error) {
	resp, err := c.Call("rtm.auth.getFrob", nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Rsp struct {
			Stat string `json:"stat"`
			Frob string `json:"frob"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	if result.Rsp.Stat != "ok" {
		return "", fmt.Errorf("RTM API error")
	}

	return result.Rsp.Frob, nil
}

// GetToken exchanges an authorized frob for a permanent auth token.
// This method updates the client's AuthToken field upon success.
// Returns an error if the frob is invalid, expired, or not yet authorized.
func (c *Client) GetToken(frob string) error {
	params := map[string]string{"frob": frob}
	resp, err := c.Call("rtm.auth.getToken", params)
	if err != nil {
		return err
	}

	var result struct {
		Rsp struct {
			Stat string `json:"stat"`
			Auth struct {
				Token string `json:"token"`
				User  struct {
					ID       string `json:"id"`
					Username string `json:"username"`
					Fullname string `json:"fullname"`
				} `json:"user"`
			} `json:"auth"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if result.Rsp.Stat != "ok" {
		return fmt.Errorf("RTM API error")
	}

	c.AuthToken = result.Rsp.Auth.Token
	return nil
}

// Call makes an authenticated API call to the RTM API.
// Parameters:
//   - method: The RTM API method name (e.g., "rtm.tasks.getList")
//   - params: Optional parameters for the API call (can be nil)
//
// Returns the raw JSON response or an RTMError if the API returns an error.
// Automatically signs the request and includes the auth token if set.
func (c *Client) Call(method string, params map[string]string) ([]byte, error) {
	if params == nil {
		params = make(map[string]string)
	}

	params["method"] = method
	params["api_key"] = c.APIKey
	params["format"] = "json"

	if c.AuthToken != "" {
		params["auth_token"] = c.AuthToken
	}

	params["api_sig"] = c.sign(params)

	// Build URL
	u, _ := url.Parse(c.BaseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// Make request
	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log but don't fail on close errors
			fmt.Printf("Warning: failed to close RTM response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Check for API errors
	var errorCheck struct {
		Rsp struct {
			Stat string `json:"stat"`
			Err  struct {
				Code string `json:"code"`
				Msg  string `json:"msg"`
			} `json:"err"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(body, &errorCheck); err == nil {
		if errorCheck.Rsp.Stat == "fail" {
			code := 0
			if errorCheck.Rsp.Err.Code != "" {
				fmt.Sscanf(errorCheck.Rsp.Err.Code, "%d", &code)
			}
			return nil, &RTMError{
				Code: code,
				Msg:  errorCheck.Rsp.Err.Msg,
			}
		}
	}

	return body, nil
}

// sign generates API signature
func (c *Client) sign(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build string to sign
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+params[k])
	}

	toSign := c.Secret + strings.Join(parts, "")

	// MD5 hash
	h := md5.New()
	h.Write([]byte(toSign))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Task represents an RTM task with its properties and metadata
type Task struct {
	// ID is the unique task identifier
	ID string `json:"id"`
	// Name is the task title/description
	Name string `json:"name"`
	// Due is the due date/time in RTM format (empty if no due date)
	Due string `json:"due"`
	// Priority is the task priority ("1"=high, "2"=medium, "3"=low, "N"=none)
	Priority string `json:"priority"`
	// Completed is the completion timestamp (empty if not completed)
	Completed string `json:"completed"`
	// Deleted is the deletion timestamp (empty if not deleted)
	Deleted string `json:"deleted"`
	// Modified is when the task was last modified
	Modified time.Time `json:"modified"`
	// Added is when the task was created
	Added time.Time `json:"added"`
	// ListID is the ID of the list containing this task
	ListID string `json:"list_id"`
	// SeriesID is the task series ID (for recurring tasks)
	SeriesID string `json:"series_id"`
	// URL is the web URL for viewing this task
	URL string `json:"url"`
}

// List represents an RTM list (a container for tasks)
type List struct {
	// ID is the unique list identifier
	ID string `json:"id"`
	// Name is the list name
	Name string `json:"name"`
	// Deleted indicates if the list is deleted ("1" if deleted)
	Deleted string `json:"deleted"`
	// Locked indicates if the list is locked ("1" if locked)
	Locked string `json:"locked"`
	// Archived indicates if the list is archived ("1" if archived)
	Archived string `json:"archived"`
	// Position is the sort position of the list
	Position string `json:"position"`
	// Smart indicates if this is a smart list ("1" if smart)
	Smart string `json:"smart"`
}

// GetLists retrieves all lists
func (c *Client) GetLists() ([]List, error) {
	resp, err := c.Call("rtm.lists.getList", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rsp struct {
			Stat  string `json:"stat"`
			Lists struct {
				List []List `json:"list"`
			} `json:"lists"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing lists: %w", err)
	}

	return result.Rsp.Lists.List, nil
}

// GetTasks retrieves tasks with optional filter
func (c *Client) GetTasks(filter, listID string) ([]Task, error) {
	params := make(map[string]string)
	if filter != "" {
		params["filter"] = filter
	}
	if listID != "" {
		params["list_id"] = listID
	}

	resp, err := c.Call("rtm.tasks.getList", params)
	if err != nil {
		return nil, err
	}

	// RTM's task response structure is complex
	var result struct {
		Rsp struct {
			Stat  string `json:"stat"`
			Tasks struct {
				List []struct {
					ID         string `json:"id"`
					Taskseries []struct {
						ID       string `json:"id"`
						Created  string `json:"created"`
						Modified string `json:"modified"`
						Name     string `json:"name"`
						Source   string `json:"source"`
						URL      string `json:"url"`
						Task     []struct {
							ID        string `json:"id"`
							Due       string `json:"due"`
							Added     string `json:"added"`
							Completed string `json:"completed"`
							Deleted   string `json:"deleted"`
							Priority  string `json:"priority"`
						} `json:"task"`
					} `json:"taskseries"`
				} `json:"list"`
			} `json:"tasks"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing tasks: %w", err)
	}

	// Flatten the nested structure
	var tasks []Task
	for _, list := range result.Rsp.Tasks.List {
		for _, series := range list.Taskseries {
			for _, task := range series.Task {
				if task.Deleted == "" && task.Completed == "" {
					t := Task{
						ID:       task.ID,
						Name:     series.Name,
						Due:      task.Due,
						Priority: task.Priority,
						ListID:   list.ID,
						SeriesID: series.ID,
						URL:      series.URL,
					}
					tasks = append(tasks, t)
				}
			}
		}
	}

	return tasks, nil
}

// AddTask creates a new task
func (c *Client) AddTask(name string, listID string) (*Task, error) {
	// First get timeline
	timeline, err := c.getTimeline()
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"timeline": timeline,
		"name":     name,
	}
	if listID != "" {
		params["list_id"] = listID
	}

	resp, err := c.Call("rtm.tasks.add", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rsp struct {
			Stat string `json:"stat"`
			List struct {
				ID         string `json:"id"`
				Taskseries []struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Created string `json:"created"`
					URL     string `json:"url"`
					Task    []struct {
						ID         string `json:"id"`
						Due        string `json:"due"`
						HasDueTime string `json:"has_due_time"`
						Added      string `json:"added"`
						Completed  string `json:"completed"`
						Deleted    string `json:"deleted"`
						Priority   string `json:"priority"`
					} `json:"task"`
				} `json:"taskseries"`
			} `json:"list"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing add task response: %w", err)
	}

	if len(result.Rsp.List.Taskseries) == 0 {
		return nil, fmt.Errorf("no taskseries returned from RTM")
	}

	taskseries := result.Rsp.List.Taskseries[0]
	if len(taskseries.Task) == 0 {
		return nil, fmt.Errorf("no task returned in taskseries from RTM")
	}

	task := taskseries.Task[0]
	return &Task{
		ID:        task.ID,
		Name:      taskseries.Name,
		ListID:    result.Rsp.List.ID,
		SeriesID:  taskseries.ID,
		Priority:  task.Priority,
		Due:       task.Due,
		Completed: task.Completed,
		Deleted:   task.Deleted,
		URL:       taskseries.URL,
	}, nil
}

// CompleteTask marks a task as complete
func (c *Client) CompleteTask(listID, seriesID, taskID string) error {
	timeline, err := c.getTimeline()
	if err != nil {
		return err
	}

	params := map[string]string{
		"timeline":      timeline,
		"list_id":       listID,
		"taskseries_id": seriesID,
		"task_id":       taskID,
	}

	_, err = c.Call("rtm.tasks.complete", params)
	return err
}

// getTimeline gets a timeline for making changes
func (c *Client) getTimeline() (string, error) {
	resp, err := c.Call("rtm.timelines.create", nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Rsp struct {
			Stat     string `json:"stat"`
			Timeline string `json:"timeline"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parsing timeline: %w", err)
	}

	return result.Rsp.Timeline, nil
}

// UpdateTask updates task properties
func (c *Client) UpdateTask(listID, seriesID, taskID string, updates map[string]string) error {
	timeline, err := c.getTimeline()
	if err != nil {
		return err
	}

	for field, value := range updates {
		params := map[string]string{
			"timeline":      timeline,
			"list_id":       listID,
			"taskseries_id": seriesID,
			"task_id":       taskID,
		}

		var method string
		switch field {
		case "name":
			method = "rtm.tasks.setName"
			params["name"] = value
		case "due":
			method = "rtm.tasks.setDueDate"
			params["due"] = value
		case "priority":
			method = "rtm.tasks.setPriority"
			params["priority"] = value
		case "estimate":
			method = "rtm.tasks.setEstimate"
			params["estimate"] = value
		case "tags":
			method = "rtm.tasks.setTags"
			params["tags"] = value
		case "list":
			method = "rtm.tasks.moveTo"
			params["to_list_id"] = value
		default:
			return fmt.Errorf("unsupported field: %s", field)
		}

		_, err := c.Call(method, params)
		if err != nil {
			return fmt.Errorf("updating %s: %w", field, err)
		}
	}

	return nil
}

// CreateList creates a new list
func (c *Client) CreateList(name string) (*List, error) {
	timeline, err := c.getTimeline()
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"timeline": timeline,
		"name":     name,
	}

	resp, err := c.Call("rtm.lists.add", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Rsp struct {
			Stat string `json:"stat"`
			List List   `json:"list"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing create list response: %w", err)
	}

	return &result.Rsp.List, nil
}

// RenameList renames a list
func (c *Client) RenameList(listID, newName string) error {
	timeline, err := c.getTimeline()
	if err != nil {
		return err
	}

	params := map[string]string{
		"timeline": timeline,
		"list_id":  listID,
		"name":     newName,
	}

	_, err = c.Call("rtm.lists.setName", params)
	return err
}

// ArchiveList archives or unarchives a list
func (c *Client) ArchiveList(listID string, archive bool) error {
	timeline, err := c.getTimeline()
	if err != nil {
		return err
	}

	params := map[string]string{
		"timeline": timeline,
		"list_id":  listID,
	}

	method := "rtm.lists.unarchive"
	if archive {
		method = "rtm.lists.archive"
	}

	_, err = c.Call(method, params)
	return err
}
