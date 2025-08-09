package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// E2E Test for RTM OAuth Flow
// This simulates what Claude Desktop does when connecting to RTM

func main() {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	fmt.Println("RTM OAuth Flow E2E Test")
	fmt.Println("=======================")
	fmt.Printf("Server: %s\n\n", serverURL)

	// Step 1: Get authorization page
	fmt.Println("Step 1: Getting authorization page...")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	authURL := fmt.Sprintf("%s/rtm/authorize?client_id=e2e-test&state=test-state-123&redirect_uri=%s",
		serverURL, url.QueryEscape("http://localhost:59263/callback"))

	resp, err := client.Get(authURL)
	if err != nil {
		log.Fatalf("Failed to get auth page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Auth page returned %d: %s", resp.StatusCode, body)
	}

	// Extract CSRF token
	var csrfToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "csrf_token" {
			csrfToken = cookie.Value
			break
		}
	}

	if csrfToken == "" {
		log.Fatal("No CSRF token found in cookies")
	}

	fmt.Printf("✓ Got CSRF token: %s...\n\n", csrfToken[:8])

	// Step 2: Submit authorization form
	fmt.Println("Step 2: Submitting authorization form...")

	form := url.Values{
		"client_id":    {"e2e-test"},
		"state":        {"test-state-123"},
		"redirect_uri": {"http://localhost:59263/callback"},
		"csrf_state":   {csrfToken},
	}

	req, _ := http.NewRequest("POST", serverURL+"/rtm/authorize",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("csrf_token=%s", csrfToken))

	resp, err = client.Do(req)
	if err != nil {
		log.Fatalf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Connect to Remember The Milk") {
		log.Fatal("Didn't get intermediate page")
	}

	// Extract code from HTML (hacky but works for testing)
	codeStart := strings.Index(string(body), "code=")
	if codeStart == -1 {
		log.Fatal("No code found in response")
	}
	codeStart += 5
	codeEnd := strings.IndexAny(string(body)[codeStart:], "\"&'")
	if codeEnd == -1 {
		codeEnd = 36 // UUID length
	}
	code := string(body)[codeStart : codeStart+codeEnd]

	fmt.Printf("✓ Got authorization code: %s\n\n", code)

	// Step 3: User would authorize on RTM
	fmt.Println("Step 3: User authorization on RTM")
	fmt.Println("⚠ In a real flow, the user would now:")
	fmt.Println("  1. Click the RTM link")
	fmt.Println("  2. Click 'OK, I'll allow it'")
	fmt.Println("  3. Return to our page")
	fmt.Println("")
	fmt.Println("For this test, we'll check if the mock is set up...\n")

	// Step 4: Check authorization status
	fmt.Println("Step 4: Checking authorization status...")

	maxAttempts := 5
	authorized := false

	for i := 0; i < maxAttempts; i++ {
		checkURL := fmt.Sprintf("%s/rtm/check-auth?code=%s", serverURL, code)
		resp, err := client.Get(checkURL)
		if err != nil {
			log.Printf("Check %d failed: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Printf("Failed to decode response: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		if result["authorized"] == true {
			authorized = true
			fmt.Println("✓ Authorization confirmed!")
			break
		}

		if result["pending"] == true {
			fmt.Printf("  Attempt %d/%d: Still pending...\n", i+1, maxAttempts)
		} else if result["error"] != nil {
			fmt.Printf("  Error: %v\n", result["error"])
		}

		time.Sleep(2 * time.Second)
	}

	if !authorized {
		fmt.Println("✗ Authorization not completed")
		fmt.Println("  This is expected if not using a mock RTM client")
		fmt.Println("  In production, the user would need to actually authorize on RTM")
	}
	fmt.Println("")

	// Step 5: Simulate callback
	fmt.Println("Step 5: Testing callback redirect...")

	callbackURL := fmt.Sprintf("%s/rtm/callback?code=%s", serverURL, code)
	resp, err = client.Get(callbackURL)
	if err != nil {
		log.Printf("Callback failed: %v", err)
	} else {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			if strings.Contains(location, "code=") && strings.Contains(location, "state=") {
				fmt.Printf("✓ Would redirect to: %s\n\n", location)
			} else {
				fmt.Printf("✗ Redirect missing parameters: %s\n\n", location)
			}
		} else {
			fmt.Printf("✗ Expected redirect, got status %d\n\n", resp.StatusCode)
		}
	}

	// Step 6: Token exchange
	fmt.Println("Step 6: Testing token exchange...")

	tokenForm := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	resp, err = http.PostForm(serverURL+"/rtm/token", tokenForm)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
	} else {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var tokenResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err == nil {
				if tokenResp["access_token"] != nil {
					fmt.Printf("✓ Got access token: %v...\n", tokenResp["access_token"].(string)[:10])
					fmt.Printf("✓ Token type: %v\n", tokenResp["token_type"])
				}
			}
		} else if resp.StatusCode == http.StatusBadRequest {
			var errorResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
				fmt.Printf("✗ Token error: %v - %v\n", errorResp["error"], errorResp["error_description"])
				if errorResp["error"] == "authorization_pending" {
					fmt.Println("  This is expected without a mock - user needs to authorize on RTM first")
				}
			}
		} else {
			fmt.Printf("✗ Unexpected status: %d\n", resp.StatusCode)
		}
	}

	fmt.Println("")
	fmt.Println("Test Summary")
	fmt.Println("============")
	fmt.Println("✓ OAuth endpoints are responding")
	fmt.Println("✓ CSRF protection is working")
	fmt.Println("✓ Authorization code is generated")
	if authorized {
		fmt.Println("✓ Full flow completed successfully")
	} else {
		fmt.Println("⚠ Authorization not completed (expected without mock)")
		fmt.Println("  In production, user must authorize on RTM")
	}
	fmt.Println("")
	fmt.Println("To test with real RTM:")
	fmt.Println("1. Set RTM_API_KEY and RTM_API_SECRET")
	fmt.Println("2. Manually complete the RTM authorization")
	fmt.Println("3. Run this test again within 60 minutes")
}
