package main

import (
	"context"
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
		// Run HTTP server for Fly.io
		runHTTPServer(s)
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

func runHTTPServer(mcpServer *server.MCPServer) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	
	// Create SSE server and get its handler
	sseServer := server.NewSSEServer(mcpServer)
	mux.Handle("/", sseServer)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting HTTP server on port %s", port)
	
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
