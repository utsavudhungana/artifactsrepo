package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Obtain access token from Azure AD
func getAccessToken(tenantID, clientID, clientSecret string) (string, error) {
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
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body using the io package
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %v", string(body))
	}

	// Parse the response body
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %v", err)
	}

	// Extract and return the token
	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}

	return "", fmt.Errorf("no access token found in response")
}

// Read zip file from local folder
func readZipFileFromLocal(filePath string) ([]byte, error) {
	// Open the zip file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local zip file: %v", err)
	}
	defer file.Close()

	// Read the file content
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read local zip file: %v", err)
	}

	return buf.Bytes(), nil
}

// Unzip artifacts from zip file
func unzipArtifacts(data []byte) (map[string][]byte, error) {
	artifactMap := make(map[string][]byte)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %v", err)
	}

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s in zip: %v", f.Name, err)
		}
		defer rc.Close()

		var buf bytes.Buffer
		_, err = io.Copy(&buf, rc)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s in zip: %v", f.Name, err)
		}

		artifactMap[f.Name] = buf.Bytes()

		fmt.Printf("Extracted file: %s\n", f.Name)
	}
	return artifactMap, nil
}

// Publish artifact to Synapse using REST API
func publishArtifactToSynapseREST(artifactType, artifactName string, content []byte, token, workspaceName string) error {
	// Synapse REST API endpoint
	var apiURL string
	switch artifactType {
	case "notebook":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/notebooks/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	case "sqlscript":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/sqlScripts/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	case "pipeline":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/pipelines/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	case "linkedService":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/linkedServices/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	case "dataset":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/datasets/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	case "sparkJobDefinition":
		apiURL = fmt.Sprintf("https://%s.dev.azuresynapse.net/sparkJobDefinitions/%s?api-version=2019-06-01-preview", workspaceName, artifactName)
	default:
		return fmt.Errorf("unsupported artifact type: %s", artifactType)
	}

	// Create the HTTP request
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(content))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to publish artifact: %s, response: %s", artifactName, string(body))
	}

	return nil
}

// Process and publish artifacts from local zip file using REST API
func processArtifactsFromLocalREST(zipFilePath, tenantID, clientID, clientSecret, workspaceName string) error {
	// Obtain access token
	token, err := getAccessToken(tenantID, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("failed to obtain access token: %v", err)
	}

	// Read the zip package from local folder
	data, err := readZipFileFromLocal(zipFilePath)
	if err != nil {
		return err
	}

	// Unzip the artifacts
	artifactMap, err := unzipArtifacts(data)
	if err != nil {
		return err
	}

	// Process in the correct order of artifact types
	artifactTypesOrder := []string{
		"notebook",
		"sqlscript",
		"sparkJobDefinition",
		"pipeline",
	}

	// Iterate over the ordered artifact types and publish
	for _, artifactType := range artifactTypesOrder {
		for fileName, content := range artifactMap {
			artifactName := filepath.Base(fileName)
			folderHierarchy := filepath.Dir(fileName)

			// Check if the current file matches the current artifact type based on the folder hierarchy
			if strings.Contains(folderHierarchy, artifactType) && strings.HasSuffix(fileName, ".json") {
				err := publishArtifactToSynapseREST(artifactType, artifactName, content, token, workspaceName)
				if err != nil {
					return fmt.Errorf("failed to publish artifact %s: %v", artifactName, err)
				}
				fmt.Printf("Successfully published artifact: %s (type: %s)\n", artifactName, artifactType)
			}
		}
	}

	return nil
}

func main() {
	// Azure credentials and Synapse workspace details
	tenantID := os.Getenv("AZURE_TENANT_ID") // Use environment variables to store secrets securely
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	workspaceName := "synawsp-dev-2"              // Replace with your Synapse workspace name
	zipFilePath := "C:/Users/livea/artifacts.zip" // Replace with the path to your local zip file

	// Process the artifacts from the local zip file
	err := processArtifactsFromLocalREST(zipFilePath, tenantID, clientID, clientSecret, workspaceName)
	if err != nil {
		fmt.Printf("Error processing artifacts: %v\n", err)
		return
	}

	fmt.Println("Artifacts published successfully.")
}
