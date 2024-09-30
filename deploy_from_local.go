package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse"
)

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

func processArtifactsFromLocal(zipFilePath string, client *armsynapse.WorkspaceClient, credential azcore.TokenCredential, resourceGroupName, targetWorkspaceName string) error {
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
				err := publishArtifactToSynapse(client, artifactType, artifactName, content, credential, resourceGroupName, targetWorkspaceName)
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
	// Variables
	subscriptionID := "your-subscription-id"     // Replace with your Azure Subscription ID
	resourceGroupName := "your-resource-group"   // Replace with your Resource Group Name
	targetWorkspaceName := "your-workspace-name" // Replace with your Synapse Workspace Name
	zipFilePath := "path/to/your/zipfile.zip"    // Replace with the path to your local zip file

	// Create Synapse Workspace client
	client, err := createWorkspaceClient(subscriptionID)
	if err != nil {
		fmt.Printf("Error creating Synapse client: %v\n", err)
		return
	}

	// Get default credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("Error obtaining default Azure credentials: %v\n", err)
		return
	}

	// Process the artifacts from the local zip file
	err = processArtifactsFromLocal(zipFilePath, client, cred, resourceGroupName, targetWorkspaceName)
	if err != nil {
		fmt.Printf("Error processing artifacts: %v\n", err)
		return
	}

	fmt.Println("Artifacts published successfully.")
}
