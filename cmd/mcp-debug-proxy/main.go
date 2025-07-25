package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vcto/cowpilot/internal/debug"
)

const (
	appName    = "mcp-debug-proxy"
	appVersion = "1.0.0"
)

// ProxyConfig holds configuration for the debug proxy
type ProxyConfig struct {
	Port         int
	TargetBinary string
	TargetArgs   []string
	TargetPort   int
	DebugConfig  *debug.DebugConfig
}

func main() {
	config := parseFlags()

	log.Printf("%s v%s starting...", appName, appVersion)
	log.Printf("Target binary: %s", config.TargetBinary)
	log.Printf("Target port: %d", config.TargetPort)
	log.Printf("Proxy port: %d", config.Port)

	// Initialize debug system with runtime configuration
	storage, debugConfig, err := debug.StartDebugSystem()
	if err != nil {
		log.Fatalf("Failed to start debug system: %v", err)
	}
	defer func() {
		if storage != nil {
			storage.Close()
		}
	}()

	// Start target MCP server
	targetCmd, err := startTargetServer(config)
	if err != nil {
		log.Fatalf("Failed to start target server: %v", err)
	}
	defer func() {
		log.Println("Stopping target server...")
		if targetCmd != nil && targetCmd.Process != nil {
			targetCmd.Process.Kill()
		}
	}()

	// Wait for target server to be ready
	if !waitForServer(config.TargetPort, 30*time.Second) {
		log.Fatalf("Target server did not start within timeout")
	}

	// Create proxy server with runtime debug config
	proxy := createProxy(config, storage, debugConfig)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: proxy,
	}

	// Start proxy server
	go func() {
		log.Printf("Debug proxy listening on port %d", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down proxy server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Proxy server stopped")
}

// parseFlags parses command line flags and environment variables
func parseFlags() *ProxyConfig {
	var (
		port         = flag.Int("port", getEnvInt("MCP_PROXY_PORT", 8080), "Proxy server port")
		targetBinary = flag.String("target", getEnvDefault("MCP_TARGET_BINARY", "./bin/cowpilot"), "Target MCP server binary")
		targetPort   = flag.Int("target-port", getEnvInt("MCP_TARGET_PORT", 8081), "Target MCP server port")
		help         = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%s v%s - MCP Debug Proxy

USAGE:
    %s [OPTIONS]

OPTIONS:
`, appName, appVersion, appName)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
ENVIRONMENT VARIABLES:
    MCP_DEBUG=true                  Enable debug system
    MCP_DEBUG_STORAGE=memory|file   Storage type
    MCP_DEBUG_PATH=./debug.db       File storage path
    MCP_DEBUG_MAX_MB=100            Storage size limit
    MCP_DEBUG_LEVEL=INFO            Debug level
    MCP_PROXY_PORT=8080             Proxy server port
    MCP_TARGET_BINARY=./bin/cowpilot Target binary path
    MCP_TARGET_PORT=8081            Target server port

EXAMPLES:
    # Basic usage
    %s --target ./bin/cowpilot --port 8080

    # With debug enabled
    MCP_DEBUG=true MCP_DEBUG_STORAGE=file %s
`, appName, appName)
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	targetArgs := flag.Args()

	return &ProxyConfig{
		Port:         *port,
		TargetBinary: *targetBinary,
		TargetArgs:   targetArgs,
		TargetPort:   *targetPort,
	}
}

// startTargetServer starts the target MCP server
func startTargetServer(config *ProxyConfig) (*exec.Cmd, error) {
	// Check if target binary exists
	if _, err := os.Stat(config.TargetBinary); os.IsNotExist(err) {
		return nil, fmt.Errorf("target binary not found: %s", config.TargetBinary)
	}

	// Build command
	args := config.TargetArgs
	cmd := exec.Command(config.TargetBinary, args...)

	// Set environment variables for target server
	env := os.Environ()
	env = append(env, fmt.Sprintf("PORT=%d", config.TargetPort))
	env = append(env, "FLY_APP_NAME=debug-proxy") // Force HTTP mode
	cmd.Env = env

	// Set up stdout/stderr capture
	cmd.Stdout = &prefixWriter{prefix: "[TARGET] ", writer: os.Stdout}
	cmd.Stderr = &prefixWriter{prefix: "[TARGET] ", writer: os.Stderr}

	log.Printf("Starting target server: %s %v", config.TargetBinary, args)
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start target server: %w", err)
	}

	log.Printf("Target server started with PID %d", cmd.Process.Pid)
	return cmd, nil
}

// waitForServer waits for a server to be ready on the specified port
func waitForServer(port int, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if isServerReady(port) {
			log.Printf("Target server is ready on port %d", port)
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// isServerReady checks if a server is responding on the specified port
func isServerReady(port int) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// createProxy creates the HTTP proxy with debug middleware
func createProxy(config *ProxyConfig, storage *debug.ConversationStorage) http.Handler {
	// Create target URL
	targetURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", config.TargetPort),
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize proxy director for debugging
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Add debug headers
		req.Header.Set("X-Debug-Proxy", "true")
		req.Header.Set("X-Debug-Session", "proxy-session")
	}

	// Add error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Proxy Error", http.StatusBadGateway)
	}

	// Wrap with debug middleware
	debugMiddleware := debug.DebugMiddleware(storage)
	handler := debugMiddleware(proxy)

	// Add health check endpoint for the proxy itself
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/debug/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","proxy":"running","target":"http://localhost:%d"}`, config.TargetPort)
	})

	// Add debug stats endpoint
	if storage != nil {
		mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
			stats, err := storage.GetStats()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "%+v", stats)
		})

		mux.HandleFunc("/debug/sessions", func(w http.ResponseWriter, r *http.Request) {
			sessions, err := storage.GetRecentSessions(20)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "%+v", sessions)
		})
	}

	return mux
}

// prefixWriter prefixes each line with a given string
type prefixWriter struct {
	prefix string
	writer *os.File
}

func (pw *prefixWriter) Write(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}
		pw.writer.WriteString(pw.prefix + line + "\n")
	}
	return len(data), nil
}

// Helper functions
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
