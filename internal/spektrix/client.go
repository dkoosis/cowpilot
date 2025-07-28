package spektrix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client handles Spektrix API requests with HMAC authentication
type Client struct {
	ClientName string
	APIUser    string
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Spektrix API client
func NewClient() *Client {
	clientName := os.Getenv("SPEKTRIX_CLIENT_NAME")
	apiUser := os.Getenv("SPEKTRIX_API_USER")
	apiKey := os.Getenv("SPEKTRIX_API_KEY")

	if err := validateCredentials(clientName, apiUser, apiKey); err != nil {
		return nil
	}

	return &Client{
		ClientName: clientName,
		APIUser:    apiUser,
		APIKey:     apiKey,
		BaseURL:    getSpektrixAPIBaseURL(clientName),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// makeRequest performs authenticated API request with HMAC signature
func (c *Client) makeRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	url := c.BaseURL + endpoint
	date := getDateHeader()

	var bodyBytes []byte
	var bodyString string

	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		bodyString = string(bodyBytes)
	}

	// Generate authorization header
	authHeader, err := getAuthorizationHeader(method, url, date, bodyString, c.APIUser, c.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth header: %w", err)
	}

	// Create request
	var req *http.Request
	if bodyBytes != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}

// handleResponse processes API response and returns parsed data or error
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	defer func() {
		_ = resp.Body.Close() // Ignore close error
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// SearchCustomers searches for customers by email
func (c *Client) SearchCustomers(email string) ([]Customer, error) {
	endpoint := fmt.Sprintf("/customers?email=%s", email)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 404 is normal - customer doesn't exist
	if resp.StatusCode == 404 {
		return []Customer{}, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if len(body) == 0 {
		return []Customer{}, nil
	}

	// Spektrix returns single customer object (not array) for email search
	var customer Customer
	if err := json.Unmarshal(body, &customer); err != nil {
		return nil, fmt.Errorf("failed to parse customer: %w", err)
	}

	return []Customer{customer}, nil
}

// GetCustomer retrieves customer by ID
func (c *Client) GetCustomer(customerID string) (*Customer, error) {
	endpoint := fmt.Sprintf("/customers/%s", customerID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var customer Customer
	if err := c.handleResponse(resp, &customer); err != nil {
		return nil, err
	}

	return &customer, nil
}

// CreateCustomer creates a new customer (step 1 of 2-step process)
func (c *Client) CreateCustomer(customer CreateCustomerRequest) (*Customer, error) {
	resp, err := c.makeRequest("POST", "/customers", customer)
	if err != nil {
		return nil, err
	}

	var result Customer
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FindOrCreateCustomer implements upsert pattern
func (c *Client) FindOrCreateCustomer(email, firstName, lastName string) (*Customer, error) {
	customers, err := c.SearchCustomers(email)
	if err != nil {
		return nil, err
	}

	// Return existing customer if found
	if len(customers) > 0 {
		return &customers[0], nil
	}

	// Create new customer
	req := CreateCustomerRequest{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	return c.CreateCustomer(req)
}

// AddCustomerAddress adds address to existing customer (step 2 of 2-step process)
func (c *Client) AddCustomerAddress(customerID string, address Address) error {
	endpoint := fmt.Sprintf("/customers/%s/addresses", customerID)

	resp, err := c.makeRequest("POST", endpoint, []Address{address})
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}

// GetTags retrieves all available tags
func (c *Client) GetTags() ([]Tag, error) {
	resp, err := c.makeRequest("GET", "/tags", nil)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	if err := c.handleResponse(resp, &tags); err != nil {
		return nil, err
	}

	return tags, nil
}

// UpdateCustomerTags updates customer tags (replaces all existing tags)
func (c *Client) UpdateCustomerTags(customerID string, tagIDs []string) error {
	endpoint := fmt.Sprintf("/customers/%s/tags", customerID)

	tags := make([]TagReference, len(tagIDs))
	for i, id := range tagIDs {
		tags[i] = TagReference{ID: id}
	}

	resp, err := c.makeRequest("PUT", endpoint, tags)
	if err != nil {
		return err
	}

	return c.handleResponse(resp, nil)
}
