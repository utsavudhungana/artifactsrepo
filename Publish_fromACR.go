//1. Download from ACR: The package is pulled from Azure Container Registry using ORAS.
//2. Unzip the Package: The compressed artifact is unzipped into individual files.
//3. Send PUT Requests: Each file is processed and sent to the Synapse workspace via its respective API.


package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Download the compressed artifact from ACR
func pullFromACR(acrURL, repository, tag, username, password string) ([]byte, error) {
	ref := fmt.Sprintf("%s/%s:%s", acrURL, repository, tag)
	remoteRepo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository reference: %v", err)
	}

	remoteRepo.Client = &auth.Client{
		Username: username,
		Password: password,
	}

	// Pull the artifact using ORAS
	desc, artifact, err := remoteRepo.FetchBytes(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to pull artifact from ACR: %v", err)
	}

	fmt.Printf("Pulled artifact with digest %s from ACR\n", desc.Digest)
	return artifact, nil
}

// Unzip the artifact
func unzipArtifact(artifact []byte) (map[string][]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}

	files := make(map[string][]byte)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in zip: %v", err)
		}

		content, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to read file from zip: %v", err)
		}
		files[f.Name] = content
		rc.Close()
	}
	return files, nil
}

// Send PUT request to Synapse for each artifact
func sendPutRequest(url, bearerToken string, bodyContent []byte) (int, string, error) {
	req, err := http.NewRequest(http.MethodPut, url, ioutil.NopCloser(bytes.NewReader(bodyContent)))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create PUT request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send PUT request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response body: %v", err)
	}

	return resp.StatusCode, string(respBody), nil
}

func main() {
	acrURL := flag.String("acr_url", "", "The URL of the Azure Container Registry.")
	repository := flag.String("repository", "", "The repository name in ACR.")
	tag := flag.String("tag", "", "The tag for the artifact in ACR.")
	username := flag.String("username", "", "The username for ACR.")
	password := flag.String("password", "", "The password for ACR.")
	targetWorkspaceName := flag.String("target_workspace_name", "", "The name of the Synapse workspace.")
	flag.Parse()

	if *acrURL == "" || *repository == "" || *tag == "" || *username == "" || *password == "" || *targetWorkspaceName == "" {
		fmt.Println("All parameters are required.")
		os.Exit(1)
	}

	// Fetch Bearer token
	accessToken, err := azCLI("account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv")
	if err != nil {
		fmt.Printf("Failed to retrieve access token: %v\n", err)
		os.Exit(1)
	}

	baseAPIURL := fmt.Sprintf("https://%s.dev.azuresynapse.net", *targetWorkspaceName)

	// Pull compressed artifact from ACR
	artifact, err := pullFromACR(*acrURL, *repository, *tag, *username, *password)
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
		apiURL := constructAPIURL(baseAPIURL, fileName, determineArtifactTypeFromFile(fileName))
		statusCode, responseText, err := sendPutRequest(apiURL, accessToken, content)
		if err != nil {
			fmt.Printf("Failed to process file %s: %v\n", fileName, err)
			continue
		}

		fmt.Printf("Successfully processed %s with status code %d\n", fileName, statusCode)
		fmt.Println(responseText)
	}
}
