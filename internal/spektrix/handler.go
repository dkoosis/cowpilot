package spektrix

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Handler manages Spektrix MCP operations
type Handler struct {
	client *Client
}

// NewHandler creates new Spektrix handler
func NewHandler() *Handler {
	client := NewClient()
	if client == nil {
		return nil
	}
	
	return &Handler{
		client: client,
	}
}

// IsAuthenticated checks if credentials are available
func (h *Handler) IsAuthenticated() bool {
	return h.client != nil
}

// GetClient returns the Spektrix API client
func (h *Handler) GetClient() *Client {
	return h.client
}

// SetupTools registers Spektrix tools with MCP server
func (h *Handler) SetupTools(s *server.MCPServer) {
	h.setupSearchCustomers(s)
	h.setupCreateCustomer(s)
	h.setupAddAddress(s)
	h.setupUpdateTags(s)
	h.setupGetTags(s)
}

func (h *Handler) setupSearchCustomers(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("spektrix_search_customers",
		mcp.WithDescription("Search for customers by email address"),
		mcp.WithString("email", mcp.Required(), mcp.Description("Customer email to search for")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}
		email, ok := args["email"].(string)
		if !ok || email == "" {
			return mcp.NewToolResultError("email parameter is required"), nil
		}

		customers, err := h.client.SearchCustomers(email)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
		}

		result := map[string]interface{}{
			"customers": customers,
			"count":     len(customers),
		}

		resultBytes, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultBytes),
				},
			},
		}, nil
	})
}

func (h *Handler) setupCreateCustomer(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("spektrix_create_customer",
		mcp.WithDescription("Create a new customer (step 1 of 2-step process)"),
		mcp.WithString("firstName", mcp.Required(), mcp.Description("Customer first name")),
		mcp.WithString("lastName", mcp.Required(), mcp.Description("Customer last name")),
		mcp.WithString("email", mcp.Required(), mcp.Description("Customer email address")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}
		
		firstName, _ := args["firstName"].(string)
		lastName, _ := args["lastName"].(string)
		email, _ := args["email"].(string)
		
		if firstName == "" || lastName == "" || email == "" {
			return mcp.NewToolResultError("firstName, lastName, and email are required"), nil
		}

		customerReq := CreateCustomerRequest{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		}

		customer, err := h.client.CreateCustomer(customerReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Customer creation failed: %v", err)), nil
		}

		result := map[string]interface{}{
			"customer": customer,
			"note":     "Customer created. Use spektrix_add_address to add address.",
		}

		resultBytes, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultBytes),
				},
			},
		}, nil
	})
}

func (h *Handler) setupAddAddress(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("spektrix_add_address",
		mcp.WithDescription("Add address to existing customer (step 2 of 2-step process)"),
		mcp.WithString("customerId", mcp.Required(), mcp.Description("Customer ID")),
		mcp.WithString("country", mcp.Required(), mcp.Description("Country code (e.g., 'US')")),
		mcp.WithString("postcode", mcp.Required(), mcp.Description("Postal/zip code")),
		mcp.WithString("line1", mcp.Description("Address line 1")),
		mcp.WithString("line2", mcp.Description("Address line 2")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State/province")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}
		
		customerID, _ := args["customerId"].(string)
		country, _ := args["country"].(string)
		postcode, _ := args["postcode"].(string)
		
		if customerID == "" || country == "" || postcode == "" {
			return mcp.NewToolResultError("customerId, country, and postcode are required"), nil
		}

		address := Address{
			Country:  country,
			Postcode: postcode,
			Line1:    getString(args, "line1"),
			Line2:    getString(args, "line2"),
			City:     getString(args, "city"),
			State:    getString(args, "state"),
		}

		err := h.client.AddCustomerAddress(customerID, address)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Address creation failed: %v", err)), nil
		}

		result := map[string]interface{}{
			"success":     true,
			"customerId":  customerID,
			"address":     address,
		}

		resultBytes, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultBytes),
				},
			},
		}, nil
	})
}

func (h *Handler) setupUpdateTags(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("spektrix_update_tags",
		mcp.WithDescription("Update customer tags (replaces all existing tags)"),
		mcp.WithString("customerId", mcp.Required(), mcp.Description("Customer ID")),
		mcp.WithString("tagIds", mcp.Required(), mcp.Description("Comma-separated tag IDs")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}
		
		customerID, _ := args["customerId"].(string)
		tagIdsStr, _ := args["tagIds"].(string)
		
		if customerID == "" {
			return mcp.NewToolResultError("customerId is required"), nil
		}

		var tagIDs []string
		if tagIdsStr != "" {
			tagIDs = splitAndTrim(tagIdsStr, ",")
		}

		err := h.client.UpdateCustomerTags(customerID, tagIDs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Tag update failed: %v", err)), nil
		}

		result := map[string]interface{}{
			"success":    true,
			"customerId": customerID,
			"tagIds":     tagIDs,
		}

		resultBytes, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultBytes),
				},
			},
		}, nil
	})
}

func (h *Handler) setupGetTags(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("spektrix_get_tags",
		mcp.WithDescription("Get all available tags in Spektrix system"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tags, err := h.client.GetTags()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get tags: %v", err)), nil
		}

		result := map[string]interface{}{
			"tags":  tags,
			"count": len(tags),
		}

		resultBytes, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultBytes),
				},
			},
		}, nil
	})
}

// Helper functions
func getString(args map[string]interface{}, key string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return ""
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	
	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
