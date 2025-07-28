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

// Client handles RTM API communication
type Client struct {
	APIKey    string
	Secret    string
	AuthToken string
	BaseURL   string
	client    *http.Client
}

// NewClient creates RTM client
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

// AuthURL generates RTM authentication URL
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

// GetFrob gets authentication frob
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

// GetToken exchanges frob for auth token
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

// Call makes authenticated API call
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
			return nil, fmt.Errorf("RTM API error %s: %s",
				errorCheck.Rsp.Err.Code,
				errorCheck.Rsp.Err.Msg)
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

// Task represents an RTM task
type Task struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Due       string    `json:"due"`
	Priority  string    `json:"priority"`
	Completed string    `json:"completed"`
	Deleted   string    `json:"deleted"`
	Modified  time.Time `json:"modified"`
	Added     time.Time `json:"added"`
	ListID    string    `json:"list_id"`
	SeriesID  string    `json:"series_id"`
	URL       string    `json:"url"`
}

// List represents an RTM list
type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Deleted  string `json:"deleted"`
	Locked   string `json:"locked"`
	Archived string `json:"archived"`
	Position string `json:"position"`
	Smart    string `json:"smart"`
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
				Taskseries struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Task struct {
						ID       string `json:"id"`
						Due      string `json:"due"`
						Priority string `json:"priority"`
					} `json:"task"`
				} `json:"taskseries"`
			} `json:"list"`
		} `json:"rsp"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing add task response: %w", err)
	}

	return &Task{
		ID:       result.Rsp.List.Taskseries.Task.ID,
		Name:     result.Rsp.List.Taskseries.Name,
		ListID:   result.Rsp.List.ID,
		SeriesID: result.Rsp.List.Taskseries.ID,
		Priority: result.Rsp.List.Taskseries.Task.Priority,
		Due:      result.Rsp.List.Taskseries.Task.Due,
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
