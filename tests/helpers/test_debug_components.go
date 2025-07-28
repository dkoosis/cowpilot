//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vcto/mcp-adapters/internal/debug"
)

func main() {
	fmt.Println("=== Testing Debug System Components ===")

	// Test storage creation
	fmt.Println("1. Testing ConversationStorage...")
	storage, err := debug.NewConversationStorage("./test_debug.db")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer os.Remove("./test_debug.db")

	// Test logging a message
	fmt.Println("2. Testing message logging...")
	err = storage.LogMessage("test-session", "inbound", "initialize",
		map[string]interface{}{"test": "data"}, nil, nil, 10)
	if err != nil {
		log.Fatalf("Failed to log message: %v", err)
	}

	// Test getting stats
	fmt.Println("3. Testing statistics...")
	stats, err := storage.GetStats()
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}

	fmt.Printf("✓ Storage test passed. Stats: %+v\n", stats)

	// Test interceptor creation
	fmt.Println("4. Testing MessageInterceptor...")
	interceptor := debug.NewMessageInterceptor(storage)
	if interceptor == nil {
		log.Fatal("Failed to create interceptor")
	}
	fmt.Printf("✓ Interceptor created with session ID: %s\n", interceptor.GetSessionID())

	// Test debug config
	fmt.Println("5. Testing DebugConfig...")
	config := debug.LoadDebugConfig()
	fmt.Printf("✓ Debug config loaded: Enabled=%v, Level=%s\n", config.Enabled, config.Level)

	fmt.Println("\n✅ All debug system components tested successfully!")
	fmt.Println("\nTo run the full system:")
	fmt.Println("1. make run-debug-proxy")
	fmt.Println("2. Test with MCP inspector or client")
}
