package rtm

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// SetupHandler handles RTM credential setup
type SetupHandler struct {
	store     CredentialStore
	validator func(apiKey, secret string) error
}

// NewSetupHandler creates RTM setup handler
func NewSetupHandler() *SetupHandler {
	// Use default credential store path
	storePath := os.Getenv("RTM_CREDENTIAL_DB_PATH")
	if storePath == "" {
		storePath = "/tmp/rtm_credentials.db" // Default for development
	}

	store, err := NewCredentialStore(storePath)
	if err != nil {
		// Return handler without store - will show error to user
		return &SetupHandler{}
	}

	return &SetupHandler{
		store:     store,
		validator: defaultRTMValidator,
	}
}

// defaultRTMValidator is the default RTM credential validator
func defaultRTMValidator(apiKey, secret string) error {
	client := NewClient(apiKey, secret)
	_, err := client.GetFrob()
	if err != nil {
		return fmt.Errorf("RTM API test failed: %w", err)
	}
	return nil
}

// HandleSetup shows credential input form or processes submission
func (h *SetupHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.showSetupForm(w, r)
		return
	}

	if r.Method == "POST" {
		h.processSetup(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *SetupHandler) showSetupForm(w http.ResponseWriter, _ *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>RTM Setup</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; }
        h1 { color: #333; }
        .form-group { margin: 20px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], input[type="password"] { 
            width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; 
            box-sizing: border-box;
        }
        button { 
            background: #007bff; color: white; border: none; padding: 12px 24px; 
            border-radius: 4px; cursor: pointer; font-size: 16px; 
        }
        button:hover { background: #0056b3; }
        button:disabled { background: #6c757d; cursor: not-allowed; }
        .info { background: #e9ecef; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .error { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 15px; border-radius: 4px; margin: 20px 0; }
    </style>
    <script>
        function validateForm() {
            const apiKey = document.getElementById('api_key').value.trim();
            const secret = document.getElementById('secret').value.trim();
            const button = document.getElementById('submitBtn');
            
            if (apiKey.length > 0 && secret.length > 0) {
                button.disabled = false;
            } else {
                button.disabled = true;
            }
        }
        
        function showValidating() {
            document.getElementById('submitBtn').disabled = true;
            document.getElementById('submitBtn').innerHTML = 'Validating...';
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>Remember The Milk Setup</h1>
        
        <div class="info">
            <h3>Before you start:</h3>
            <p>You need RTM API credentials. If you don't have them:</p>
            <ol>
                <li>Visit <a href="https://www.rememberthemilk.com/services/api/" target="_blank">RTM API page</a></li>
                <li>Request API key and secret via email</li>
                <li>Wait for email with your credentials</li>
            </ol>
        </div>
        
        <form method="POST" onsubmit="showValidating()">
            <div class="form-group">
                <label for="api_key">API Key:</label>
                <input type="text" id="api_key" name="api_key" 
                       placeholder="Enter your RTM API key" 
                       oninput="validateForm()" required>
            </div>
            
            <div class="form-group">
                <label for="secret">API Secret:</label>
                <input type="password" id="secret" name="secret" 
                       placeholder="Enter your RTM API secret" 
                       oninput="validateForm()" required>
            </div>
            
            <button type="submit" id="submitBtn" disabled>Validate & Save Credentials</button>
        </form>
        
        <div class="info">
            <p><strong>Security:</strong> Your credentials are encrypted and stored securely. 
            Only you can access your RTM data.</p>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		http.Error(w, "Failed to render form", http.StatusInternalServerError)
	}
}

func (h *SetupHandler) processSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.showError(w, "Invalid form data")
		return
	}

	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	secret := strings.TrimSpace(r.FormValue("secret"))

	// Validate required fields
	if apiKey == "" || secret == "" {
		h.showError(w, "API key and secret are required")
		return
	}

	// Basic format validation
	if len(apiKey) < 10 || len(secret) < 10 {
		h.showError(w, "API key and secret appear to be too short")
		return
	}

	// Validate credentials with RTM API
	if err := h.validateRTMCredentials(apiKey, secret); err != nil {
		h.showError(w, fmt.Sprintf("Invalid RTM credentials: %v", err))
		return
	}

	// Store encrypted credentials
	if h.store == nil {
		h.showError(w, "Credential storage unavailable")
		return
	}

	// Use client IP as user ID for now (TODO: proper user sessions)
	userID := r.RemoteAddr
	if userID == "" {
		userID = "default_user"
	}

	if err := h.store.Store(userID, apiKey, secret); err != nil {
		h.showError(w, fmt.Sprintf("Failed to save credentials: %v", err))
		return
	}

	h.showSuccess(w, "Credentials validated and saved successfully!")
}

// validateRTMCredentials tests credentials with RTM API
func (h *SetupHandler) validateRTMCredentials(apiKey, secret string) error {
	client := NewClient(apiKey, secret)

	// Test credentials by getting a frob
	_, err := client.GetFrob()
	if err != nil {
		return fmt.Errorf("RTM API test failed: %w", err)
	}

	return nil
}

func (h *SetupHandler) showError(w http.ResponseWriter, message string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>RTM Setup - Error</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; }
        .error { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .button { display: inline-block; background: #007bff; color: white; text-decoration: none; padding: 10px 20px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Setup Error</h1>
        <div class="error">%s</div>
        <a href="/rtm/setup" class="button">Try Again</a>
    </div>
</body>
</html>`, message)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	if _, err := fmt.Fprint(w, html); err != nil {
		http.Error(w, "Failed to render error", http.StatusInternalServerError)
	}
}

func (h *SetupHandler) showSuccess(w http.ResponseWriter, message string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>RTM Setup - Success</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .button { display: inline-block; background: #28a745; color: white; text-decoration: none; padding: 10px 20px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Setup Complete</h1>
        <div class="success">%s</div>
        <a href="/oauth/authorize" class="button">Continue to Authorization</a>
    </div>
</body>
</html>`, message)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		http.Error(w, "Failed to render success", http.StatusInternalServerError)
	}
}
