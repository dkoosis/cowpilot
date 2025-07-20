package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"cowpilot",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add hello tool
	tool := mcp.NewTool("hello",
		mcp.WithDescription("Says hello to the world"),
	)

	// Add tool handler
	s.AddTool(tool, helloHandler)

	// Check if we're running on Fly.io or locally
	if os.Getenv("FLY_APP_NAME") != "" {
		// Run SSE server for Fly.io
		runSSEServer(s)
	} else {
		// Run stdio server for local development
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Hello, World!"), nil
}

func runSSEServer(mcpServer *server.MCPServer) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// SSE endpoint using mark3labs SSE support
		log.Printf("SSE connection from %s", r.RemoteAddr)
		if err := server.ServeSSE(w, r, mcpServer); err != nil {
			log.Printf("SSE error: %v", err)
		}
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting SSE server on port %s", port)
	log.Printf("SSE endpoint: http://localhost:%s/sse", port)
	
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	srv.Close()
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
