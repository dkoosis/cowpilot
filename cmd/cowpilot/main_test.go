package main

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHelloHandler(t *testing.T) {
	ctx := context.Background()
	
	// Create request according to mark3labs API
	req := mcp.CallToolRequest{
		Params: mcp.CallToolRequestParams{
			Name:      "hello",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := helloHandler(ctx, req)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	
	// mark3labs uses Content array
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
}
