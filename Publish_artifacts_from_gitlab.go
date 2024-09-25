package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"
)

// Map folder names to artifact types
func getArtifactTypeFromFolder(folder string) (string, error) {
	switch folder {
	case "managedVirtualNetwork":
		return "managedVirtualNetwork", nil
	case "integrationRuntime":
		return "integrationRuntime", nil
	case "linkedService":
		return "linkedService", nil
	case "dataset":
		return "dataset", nil
	case "notebook":
		return "notebook", nil
	case "sqlscript":
		return "sqlscript", nil
	case "kqlscript":
		return "kqlscript", nil
	case "sparkJobDefinition":
		return "sparkJobDefinition", nil
	case "pipeline":
		return "pipeline", nil
	default:
		return "", fmt.Errorf("unsupported folder: %s", folder)
	}
}

func getFilePathsFromGitLabDirectory(repoID int, ref, gitlabURL, privateToken string) ([]string, error) {
	git, err := gitlab.NewClient(privateToken, gitlab.WithBaseURL(gitlabURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %v", err)
	}

	project, _, err := git.Projects.GetProject(repoID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %v", err)
	}

	opts := &gitlab.ListTreeOptions{
		Ref: &ref,
	}

	files, _, err := git.Repositories.ListTree(project.ID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory tree: %v", err)
	}

	var filePaths []string
	for _, file := range files {
		if file.Type == "blob" {
			filePaths = append(filePaths, file.Path)
		}
	}

	return filePaths, nil
}

func getFileContentFromGitLab(repoID int, filePath, ref, gitlabURL, privateToken string) (string, error) {
	git, err := gitlab.NewClient(privateToken, gitlab.WithBaseURL(gitlabURL))
	if err != nil {
		return "", fmt.Errorf("failed to create GitLab client: %v", err)
	}

	file, _, err := git.RepositoryFiles.GetFile(repoID, filePath, &gitlab.GetFileOptions{Ref: gitlab.String(ref)})
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}

	decoded, err := file.DecodeBase64()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %v", err)
	}

	return string(decoded), nil
}

func sendPutRequest(url, bearerToken string, bodyContent []byte) (int, string, error) {
	req, err := http.NewRequest(http.MethodPut, url, ioutil.NopCloser(strings.NewReader(string(bodyContent))))
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

func constructAPIURL(baseURL, filePath, artifactType string) string {
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return fmt.Sprintf("%s/%s/%s?api-version=2020-12-01", baseURL, artifactType, fileNameWithoutExt)
}

func azCLI(args string) (string, error) {
	cmd := exec.Command("az", strings.Split(args, " ")...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute az CLI command: %v", err)
	}
	return string(out), nil
}

func main() {
	repoID := flag.Int("repo_id", 0, "The ID of the GitLab repository.")
	ref := flag.String("ref", "", "The branch name or commit hash.")
	targetWorkspaceName := flag.String("target_workspace_name", "", "The name of the Synapse workspace.")
	flag.Parse()

	if *repoID == 0 || *ref == "" || *targetWorkspaceName == "" {
		fmt.Println("All parameters are required.")
		os.Exit(1)
	}

	gitlabURL := "https://gitlab.com"
	privateToken := os.Getenv("GITLAB_PRIVATE_TOKEN")
	if privateToken == "" {
		fmt.Println("GITLAB_PRIVATE_TOKEN environment variable must be set.")
		os.Exit(1)
	}

	// Fetch Bearer token
	accessToken, err := azCLI("account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv")
	if err != nil {
		fmt.Printf("Failed to retrieve access token: %v\n", err)
		os.Exit(1)
	}

	baseAPIURL := fmt.Sprintf("https://%s.dev.azuresynapse.net", *targetWorkspaceName)

	// Get list of files in the repository
	filePaths, err := getFilePathsFromGitLabDirectory(*repoID, *ref, gitlabURL, privateToken)
	if err != nil {
		fmt.Printf("Failed to retrieve file paths from repository: %v\n", err)
		os.Exit(1)
	}

	for _, filePath := range filePaths {
		content, err := getFileContentFromGitLab(*repoID, filePath, *ref, gitlabURL, privateToken)
		if err != nil {
			fmt.Printf("Failed to retrieve content for file %s: %v\n", filePath, err)
			continue
		}

		// Determine the artifact type based on the directory structure
		dir := filepath.Dir(filePath)
		folderName := filepath.Base(dir)

		artifactType, err := getArtifactTypeFromFolder(folderName)
		if err != nil {
			fmt.Printf("Failed to determine artifact type for file %s: %v\n", filePath, err)
			continue
		}

		apiURL := constructAPIURL(baseAPIURL, filePath, artifactType)
		statusCode, responseText, err := sendPutRequest(apiURL, accessToken, []byte(content))
		if err != nil {
			fmt.Printf("Failed to process file %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("Successfully processed %s with status code %d\n", filePath, statusCode)
		fmt.Println(responseText)
	}
}
