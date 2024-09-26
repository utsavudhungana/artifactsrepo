//Set followinng Env variables
//export AZURE_CLIENT_ID="your-client-id"
//export AZURE_CLIENT_SECRET="your-client-secret"
//export AZURE_TENANT_ID="your-tenant-id"

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	// Get environment variables for credentials
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	// Validate that required env variables are set
	if clientID == "" || clientSecret == "" || tenantID == "" {
		log.Fatal("Environment variables AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, or AZURE_TENANT_ID are not set")
	}

	// Define OAuth 2.0 token endpoint and scope
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	scope := "https://dev.azuresynapse.net/.default"
	grantType := "client_credentials"

	// Set the data you want to send in the request body
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", scope)
	data.Set("grant_type", grantType)

	// Create the request
	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body using the io package
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to get token: %v", string(body))
	}

	// Parse the response body
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("Error parsing JSON response: %v", err)
	}

	// Print the token
	if token, ok := result["access_token"].(string); ok {
		fmt.Printf("Access token: %s\n", token)
	} else {
		log.Fatal("No access token found in response")
	}
}
