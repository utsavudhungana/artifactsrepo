//1. Download from ACR: The package is pulled from Azure Container Registry using ORAS.
//2. Unzip the Package: The compressed artifact is unzipped into individual files.
//3. Send PUT Requests: Each file is processed and sent to the Synapse workspace via its respective API.


package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Define your variable values here (replace with actual values or fetch from environment)
var (
	acrURL             = "your-acr-url.azurecr.io"
	repository         = "your-repository-name"
	tag                = "your-tag"
	username           = "your-username"
	password           = "your-password"
	targetWorkspaceName = "your-synapse-workspace"
)

// Pull the compressed artifact from ACR using ORAS
func pullFromACR() ([]byte, error) {
	ref := fmt.Sprintf("%s/%s:%s", acrURL, repository, tag)
	remoteRepo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository reference: %v", err)
	}

	remoteRepo.Client = &auth.Client{
		Username: username,
		Password: password,
	}

	// Pull the artifact
	var artifact bytes.Buffer
	_, err = remoteRepo.FetchBytes(nil, &artifact)
	if err != nil {
		return nil, fmt.Errorf("failed to pull artifact from ACR: %v", err)
	}

	fmt.Println("Successfully pulled artifact from ACR")
	return artifact.Bytes(), nil
}

// Unzip the artifact into individual files
func unzipArtifact(artifact []byte) (map[string][]byte, error) {
	files := make(map[string][]byte)
	r := bytes.NewReader(artifact)
	zr, err := zip.NewReader(r, int64(len(artifact)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip archive: %v", err)
	}

	for _, file := range zr.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %v", file.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", file.Name, err)
		}

		files[file.Name] = content
	}

	return files, nil
}

// Send PUT request to Synapse workspace
func sendPutRequest(url, accessToken string, bodyContent []byte) (int, string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bodyContent))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create PUT request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send PUT request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response body: %v", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// Determine artifact type based on file path
func determineArtifactTypeFromFile(filePath string) string {
	switch {
	case filepath.Dir(filePath) == "notebook":
		return "notebook"
	case filepath.Dir(filePath) == "sqlscript":
		return "sqlscript"
	case filepath.Dir(filePath) == "kqlscript":
		return "kqlscript"
	case filepath.Dir(filePath) == "sparkJobDefinition":
		return "sparkJobDefinition"
	case filepath.Dir(filePath) == "dataset":
		return "dataset"
	case filepath.Dir(filePath) == "linkedService":
		return "linkedService"
	case filepath.Dir(filePath) == "integrationRuntime":
		return "integrationRuntime"
	case filepath.Dir(filePath) == "managedVirtualNetwork":
		return "managedVirtualNetwork"
	case filepath.Dir(filePath) == "pipeline":
		return "pipeline"
	default:
		return "unknown"
	}
}

// Construct API URL for publishing to Synapse
func constructAPIURL(baseURL, filePath, artifactType string) string {
	fileName := filepath.Base(filePath)
	return fmt.Sprintf("%s/%s/%s?api-version=2020-12-01", baseURL, artifactType, fileName)
}

func main() {
	// Fetch Bearer token
	accessToken, err := azCLI("account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv")
	if err != nil {
		fmt.Printf("Failed to retrieve access token: %v\n", err)
		os.Exit(1)
	}

	baseAPIURL := fmt.Sprintf("https://%s.dev.azuresynapse.net", targetWorkspaceName)

	// Pull compressed artifact from ACR
	artifact, err := pullFromACR()
	if err != nil {
		fmt.Printf("Failed to pull artifact from ACR: %v\n", err)
		os.Exit(1)
	}

	// Unzip the artifact
	files, err := unzipArtifact(artifact)
	if err != nil {
		fmt.Printf("Failed to unzip artifact: %v\n", err)
		os.Exit(1)
	}

	// Process and publish each artifact
	for fileName, content := range files {
		artifactType := determineArtifactTypeFromFile(fileName)
		if artifactType == "unknown" {
			fmt.Printf("Skipping unknown artifact type for file: %s\n", fileName)
			continue
		}

		apiURL := constructAPIURL(baseAPIURL, fileName, artifactType)
		statusCode, responseText, err := sendPutRequest(apiURL, accessToken, content)
		if err != nil {
			fmt.Printf("Failed to process file %s: %v\n", fileName, err)
			continue
		}

		fmt.Printf("Successfully processed %s with status code %d\n", fileName, statusCode)
		fmt.Println(responseText)
	}
}
