package main

import (
	"fmt"
	"github.com/vcto/cowpilot/internal/debug"
	"os"
)

func main() {
	fmt.Println("=== Testing Runtime Debug Configuration ===")

	// Test 1: Default (disabled)
	fmt.Println("\n1. Default configuration (debug disabled):")
	storage1, config1, err1 := debug.StartDebugSystem()
	if err1 != nil {
		fmt.Printf("Error: %v\n", err1)
	} else {
		fmt.Printf("Enabled: %v, Storage Type: %s, Is Storage Enabled: %v\n",
			config1.Enabled, config1.StorageType, storage1.IsEnabled())
		storage1.Close()
	}

	// Test 2: Enable debug with memory storage
	fmt.Println("\n2. Debug enabled with memory storage:")
	os.Setenv("MCP_DEBUG", "true")
	os.Setenv("MCP_DEBUG_STORAGE", "memory")
	os.Setenv("MCP_DEBUG_MAX_MB", "50")

	storage2, config2, err2 := debug.StartDebugSystem()
	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)
	} else {
		fmt.Printf("Enabled: %v, Storage Type: %s, Max MB: %d, Is Storage Enabled: %v\n",
			config2.Enabled, config2.StorageType, config2.MaxMemoryMB, storage2.IsEnabled())

		// Test logging a message
		err := storage2.LogMessage("test-session", "inbound", "test_method",
			map[string]string{"param": "value"},
			map[string]string{"result": "success"},
			nil, 42)
		if err != nil {
			fmt.Printf("Log error: %v\n", err)
		} else {
			fmt.Println("✅ Message logged successfully")
		}

		// Get stats
		stats, _ := storage2.GetStats()
		fmt.Printf("Stats: %+v\n", stats)

		storage2.Close()
	}

	// Test 3: File storage
	fmt.Println("\n3. Debug enabled with file storage:")
	os.Setenv("MCP_DEBUG_STORAGE", "file")
	os.Setenv("MCP_DEBUG_PATH", "./test_debug.db")

	storage3, config3, err3 := debug.StartDebugSystem()
	if err3 != nil {
		fmt.Printf("Error: %v\n", err3)
	} else {
		fmt.Printf("Enabled: %v, Storage Type: %s, Path: %s\n",
			config3.Enabled, config3.StorageType, config3.StoragePath)
		storage3.Close()

		// Cleanup test file
		os.Remove("./test_debug.db")
	}

	fmt.Println("\n✅ Runtime configuration tests completed")
}
