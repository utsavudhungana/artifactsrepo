// 1. Fetch Files from GitLab: The files are fetched from GitLab based on the repository and branch/commit reference.
// 2. Compress Artifacts: All files are zipped into a single artifact.
// 3. Push to ACR using ORAS: The ORAS Go library is used to upload the compressed artifact to the Azure Container Registry.


package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Define your variable values here (replace with actual values or fetch from environment)
var (
	acrURL     = "your-acr-url.azurecr.io"
	repository = "your-repository-name"
	tag        = "your-tag"
	username   = "your-username"
	password   = "your-password"
)

// Compress files into a ZIP archive
func compressFiles(filePaths []string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, filePath := range filePaths {
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
		}

		w, err := zipWriter.Create(filepath.Base(filePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create zip entry: %v", err)
		}

		_, err = w.Write(fileContent)
		if err != nil {
			return nil, fmt.Errorf("failed to write to zip: %v", err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip: %v", err)
	}

	return buf.Bytes(), nil
}

// Push the compressed artifact to Azure Container Registry using ORAS
func pushToACR(artifact []byte) error {
	ref := fmt.Sprintf("%s/%s:%s", acrURL, repository, tag)
	remoteRepo, err := remote.NewRepository(ref)
	if err != nil {
		return fmt.Errorf("failed to create repository reference: %v", err)
	}

	remoteRepo.Client = &auth.Client{
		Username: username,
		Password: password,
	}

	// Push the artifact using ORAS
	desc, err := remoteRepo.PushBytes(nil, "application/vnd.zip", artifact)
	if err != nil {
		return fmt.Errorf("failed to push artifact to ACR: %v", err)
	}

	fmt.Printf("Pushed artifact with digest %s to ACR\n", desc.Digest)
	return nil
}

func getFilePathsFromRootDirectory() ([]string, error) {
	// Fetch all JSON files from the current directory
	var filePaths []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".json" {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files in directory: %v", err)
	}

	return filePaths, nil
}

func main() {
	// Get list of JSON files in the root directory
	filePaths, err := getFilePathsFromRootDirectory()
	if err != nil {
		fmt.Printf("Failed to retrieve file paths from directory: %v\n", err)
		os.Exit(1)
	}

	// Compress files into a ZIP
	artifact, err := compressFiles(filePaths)
	if err != nil {
		fmt.Printf("Failed to compress files: %v\n", err)
		os.Exit(1)
	}

	// Push the compressed artifact to ACR using ORAS
	err = pushToACR(artifact)
	if err != nil {
		fmt.Printf("Failed to push to ACR: %v\n", err)
		os.Exit(1)
	}
}
