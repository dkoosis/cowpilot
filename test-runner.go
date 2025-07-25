package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("🧪 Running tests with verbose output...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	cmd := exec.Command("go", "test", "-v", "./internal/mcp/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "/Users/vcto/Projects/cowpilot"

	if err := cmd.Run(); err != nil {
		fmt.Printf("Test failed with error: %v\n", err)
	}
}
