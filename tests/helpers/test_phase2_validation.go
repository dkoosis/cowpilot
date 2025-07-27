//go:build ignore

package main

import (
	"fmt"
	"github.com/vcto/cowpilot/internal/debug"
	"github.com/vcto/cowpilot/internal/validator"
	"os"
)

func main() {
	fmt.Println("=== Testing Phase 2: Protocol Validation ===")

	// Enable debug with validation
	os.Setenv("MCP_DEBUG", "true")
	os.Setenv("MCP_DEBUG_STORAGE", "memory")
	os.Setenv("MCP_DEBUG_LEVEL", "DEBUG")

	storage, config, err := debug.StartDebugSystem()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer storage.Close()

	// Create validated interceptor
	interceptor := debug.NewValidatedMessageInterceptor(storage, config)

	fmt.Printf("✅ Validation system initialized\n")
	fmt.Printf("Session ID: %s\n", interceptor.GetSessionID())

	// Test 1: Valid MCP request
	fmt.Println("\n1. Testing valid tools/call request:")
	report1 := interceptor.LogRequestWithValidation("tools/call", map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "hello world",
		},
	})
	if report1 != nil {
		fmt.Printf("Score: %.1f, Valid: %v, Issues: %d\n",
			report1.Score, report1.IsValid, len(report1.Results))
	}

	// Test 2: Invalid request (missing required fields)
	fmt.Println("\n2. Testing invalid tools/call request:")
	report2 := interceptor.LogRequestWithValidation("tools/call", map[string]interface{}{
		"arguments": map[string]interface{}{
			"message": "hello world",
		},
	})
	if report2 != nil {
		fmt.Printf("Score: %.1f, Valid: %v, Issues: %d\n",
			report2.Score, report2.IsValid, len(report2.Results))
		for _, result := range report2.Results {
			fmt.Printf("  %s: %s\n", result.Level.String(), result.Message)
		}
	}

	// Test 3: Security threat detection
	fmt.Println("\n3. Testing security validation:")
	report3 := interceptor.LogRequestWithValidation("tools/call", map[string]interface{}{
		"name": "sql_query",
		"arguments": map[string]interface{}{
			"query": "SELECT * FROM users WHERE id = 1 OR 1=1; DROP TABLE users;--",
		},
	})
	if report3 != nil {
		fmt.Printf("Score: %.1f, Valid: %v, Issues: %d\n",
			report3.Score, report3.IsValid, len(report3.Results))
		for _, result := range report3.Results {
			if result.Level == validator.LevelCritical {
				fmt.Printf("  🚨 %s: %s\n", result.Level.String(), result.Message)
			}
		}
	}

	// Test 4: Prompts validation (critical feature)
	fmt.Println("\n4. Testing prompts/get validation:")
	report4 := interceptor.LogRequestWithValidation("prompts/get", map[string]interface{}{
		"name": "code_review",
		"arguments": map[string]interface{}{
			"language": "go",
			"code":     "func main() { fmt.Println(\"hello\") }",
		},
	})
	if report4 != nil {
		fmt.Printf("Score: %.1f, Valid: %v, Issues: %d\n",
			report4.Score, report4.IsValid, len(report4.Results))
	}

	// Get validation stats
	if vs, ok := storage.(debug.ValidationStorage); ok {
		stats, _ := vs.GetValidationStats()
		fmt.Printf("\n✅ Validation stats: %+v\n", stats)
	}

	fmt.Println("\n✅ Phase 2 Protocol Validation testing complete")
}
