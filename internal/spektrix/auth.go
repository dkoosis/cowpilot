package spektrix

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// getAuthorizationHeader generates Spektrix API Authorization header
// Ported from SpektrixAuth.js getAuthorizationHeader function
func getAuthorizationHeader(method, url, date, body, apiUser, apiKey string) (string, error) {
	// Build string to sign: METHOD\nURL\nDATE\n[MD5_BODY]
	stringToSign := strings.ToUpper(method) + "\n" + url + "\n" + date

	// Add MD5 hash of body if present (required even for empty bodies)
	if body != "" {
		bodyHash := md5.Sum([]byte(body))
		encodedBodyHash := base64.StdEncoding.EncodeToString(bodyHash[:])
		stringToSign += "\n" + encodedBodyHash
	}

	// Decode API key from base64
	decodedKeyBytes, err := base64.StdEncoding.DecodeString(apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode API key: %w", err)
	}

	// Convert bytes to string (matching JavaScript implementation)
	keyAsString := string(decodedKeyBytes)

	// Generate HMAC signature using custom implementation
	signatureBytes, err := hmacSHA1(stringToSign, keyAsString)
	if err != nil {
		return "", fmt.Errorf("failed to generate HMAC signature: %w", err)
	}

	// Encode signature to base64
	encodedSignature := base64.StdEncoding.EncodeToString(signatureBytes)

	// Return formatted authorization header
	return fmt.Sprintf("SpektrixAPI3 %s:%s", apiUser, encodedSignature), nil
}

// getDateHeader generates properly formatted date header
func getDateHeader() string {
	return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

// validateCredentials checks if all required Spektrix credentials are present
func validateCredentials(clientName, apiUser, apiKey string) error {
	if clientName == "" {
		return fmt.Errorf("SPEKTRIX_CLIENT_NAME is required")
	}
	if apiUser == "" {
		return fmt.Errorf("SPEKTRIX_API_USER is required")
	}
	if apiKey == "" {
		return fmt.Errorf("SPEKTRIX_API_KEY is required")
	}
	return nil
}

// getSpektrixAPIBaseURL constructs the base URL for Spektrix API
func getSpektrixAPIBaseURL(clientName string) string {
	return fmt.Sprintf("https://system.spektrix.com/%s/api/v3", clientName)
}
