package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Real-time log monitor for OAuth debugging
func main() {
	fmt.Println("RTM OAuth Real-Time Monitor")
	fmt.Println("===========================")
	fmt.Println("This will start the server with enhanced logging and monitor OAuth events")
	fmt.Println()

	// Check if server is already running
	checkCmd := exec.Command("pgrep", "-f", "cmd/rtm/main.go")
	if err := checkCmd.Run(); err == nil {
		fmt.Println("⚠️  Server is already running. Please stop it first:")
		fmt.Println("   pkill -f 'cmd/rtm/main.go'")
		return
	}

	// Start server with logging
	fmt.Println("Starting RTM server with OAuth logging...")
	cmd := exec.Command("go", "run", "cmd/rtm/main.go", "--disable-auth=false")
	cmd.Env = append(os.Environ(),
		"DEBUG=true",
		"PORT=8081",
		"SERVER_URL=http://localhost:8081",
	)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stderr pipe: %v\n", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}

	fmt.Println("Server started. Monitoring OAuth events...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("-" + strings.Repeat("-", 60))
	fmt.Println()

	// Monitor stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			timestamp := time.Now().Format("15:04:05")
			
			// Highlight OAuth-related lines
			if strings.Contains(line, "[OAuth") {
				fmt.Printf("\033[1;36m%s | %s\033[0m\n", timestamp, line)
			} else if strings.Contains(line, "ERROR") || strings.Contains(line, "error") {
				fmt.Printf("\033[1;31m%s | %s\033[0m\n", timestamp, line)
			} else if strings.Contains(line, "SUCCESS") || strings.Contains(line, "✓") {
				fmt.Printf("\033[1;32m%s | %s\033[0m\n", timestamp, line)
			} else if strings.Contains(strings.ToLower(line), "auth") || 
			          strings.Contains(strings.ToLower(line), "token") ||
			          strings.Contains(strings.ToLower(line), "callback") {
				fmt.Printf("\033[1;33m%s | %s\033[0m\n", timestamp, line)
			} else {
				fmt.Printf("%s | %s\n", timestamp, line)
			}
		}
	}()

	// Monitor stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			timestamp := time.Now().Format("15:04:05")
			fmt.Printf("\033[1;31m%s | STDERR: %s\033[0m\n", timestamp, line)
		}
	}()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Printf("\nServer exited with error: %v\n", err)
	} else {
		fmt.Println("\nServer stopped cleanly")
	}
}
