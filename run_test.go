package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("go", "test", "-v", "./internal/rtm", "-run", "TestFullOAuthFlow")
	cmd.Dir = "/Users/vcto/Projects/cowpilot"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Test failed: %v\n", err)
		os.Exit(1)
	}
}
