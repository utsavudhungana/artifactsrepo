//1. Download from ACR: The package is pulled from Azure Container Registry using ORAS.
//2. Unzip the Package: The compressed artifact is unzipped into individual files.
//3. Send PUT Requests: Each file is processed and sent to the Synapse workspace via its respective API.

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse"
	"github.com/deislabs/oras/pkg/oras"
)

func downloadFromACR(acrUrl, repository, tag string) ([]byte, error) {
	// Example using ORAS to download from ACR
	content := []byte{}
	err := oras.Pull(acrUrl, repository+":"+tag, nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to download from ACR: %v", err)
	}
	return content, nil
}

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
	}
	return artifactMap, nil
}

func publishArtifactToSynapse(client *armsynapse.WorkspaceClient, artifactType, artifactName string, content []byte, credential azcore.TokenCredential, resourceGroupName, targetWorkspaceName string) error {
	ctx := context.Background()

	switch artifactType {
	case "notebook":
		_, err := client.CreateOrUpdateNotebook(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update notebook: %v", err)
		}
	case "sqlscript":
		_, err := client.CreateOrUpdateSQLScript(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update SQL script: %v", err)
		}
	case "pipeline":
		_, err := client.CreateOrUpdatePipeline(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update pipeline: %v", err)
		}
	case "linkedService":
		_, err := client.CreateOrUpdateLinkedService(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update linked service: %v", err)
		}
	case "dataset":
		_, err := client.CreateOrUpdateDataset(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update dataset: %v", err)
		}
	case "sparkJobDefinition":
		_, err := client.CreateOrUpdateSparkJobDefinition(ctx, resourceGroupName, targetWorkspaceName, artifactName, content, nil)
		if err != nil {
			return fmt.Errorf("failed to create/update spark job definition: %v", err)
		}
	default:
		return fmt.Errorf("unknown artifact type: %s", artifactType)
	}

	return nil
}

// Create a client using DefaultAzureCredential
func createWorkspaceClient(subscriptionID string) (*armsynapse.WorkspaceClient, error) {
	// Get default credentials using DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get default Azure credentials: %v", err)
	}

	// Create Synapse Workspace client
	client, err := armsynapse.NewWorkspaceClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Synapse workspace client: %v", err)
	}

	return client, nil
}

func processArtifactsFromACR(acrUrl, repository, tag string, client *armsynapse.WorkspaceClient, credential azcore.TokenCredential, resourceGroupName, targetWorkspaceName string) error {
	// Download the zip package from ACR
	data, err := downloadFromACR(acrUrl, repository, tag)
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
		"integrationRuntime",
		"linkedService",
		"dataset",
		"notebook",
		"sqlscript",
		"sparkJobDefinition",
		"pipeline",
	}

	// Iterate over the ordered artifact types and publish
	for _, artifactType := range artifactTypesOrder {
		for fileName, content := range artifactMap {
			artifactName := filepath.Base(fileName)
			folderHierarchy := filepath.Dir(fileName) // Use the directory structure to maintain hierarchy

			// Check if the current file matches the current artifact type based on the folder hierarchy
			if strings.Contains(folderHierarchy, artifactType) {
				err := publishArtifactToSynapse(client, artifactType, artifactName, folderHierarchy, content, credential, resourceGroupName, targetWorkspaceName)
				if err != nil {
					return fmt.Errorf("failed to publish artifact %s: %v", artifactName, err)
				}
				fmt.Printf("Successfully published artifact: %s (type: %s)\n", artifactName, artifactType)
			}
		}
	}

	return nil
}
