// 1. Fetch Files from GitLab: The files are fetched from GitLab based on the repository and branch/commit reference.
// 2. Compress Artifacts: All files are zipped into a single artifact.
// 3. Push to ACR using ORAS: The ORAS Go library is used to upload the compressed artifact to the Azure Container Registry.



package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	gitlab "github.com/xanzy/go-gitlab"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
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
func pushToACR(acrURL, repository, tag, username, password string, artifact []byte) error {
	ref := fmt.Sprintf("%s/%s:%s", acrURL, repository, tag)
	remoteRepo, err := remote.NewRepository(ref)
	if err != nil {
		return fmt.Errorf("failed to create repository reference: %v", err)
	}

	remoteRepo.Client = &auth.Client{
		Username: username,
		Password: password,
	}

	// Push the artifact using ORAS (oras-go)
	desc, err := remoteRepo.PushBytes(nil, "application/vnd.zip", artifact)
	if err != nil {
		return fmt.Errorf("failed to push artifact to ACR: %v", err)
	}

	fmt.Printf("Pushed artifact with digest %s to ACR\n", desc.Digest)
	return nil
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

func main() {
	repoID := flag.Int("repo_id", 0, "The ID of the GitLab repository.")
	ref := flag.String("ref", "", "The branch name or commit hash.")
	acrURL := flag.String("acr_url", "", "The URL of the Azure Container Registry.")
	repository := flag.String("repository", "", "The repository name in ACR.")
	tag := flag.String("tag", "", "The tag for the artifact in ACR.")
	username := flag.String("username", "", "The username for ACR.")
	password := flag.String("password", "", "The password for ACR.")
	flag.Parse()

	if *repoID == 0 || *ref == "" || *acrURL == "" || *repository == "" || *tag == "" || *username == "" || *password == "" {
		fmt.Println("All parameters are required.")
		os.Exit(1)
	}

	gitlabURL := "https://gitlab.com"
	privateToken := os.Getenv("GITLAB_PRIVATE_TOKEN")
	if privateToken == "" {
		fmt.Println("GITLAB_PRIVATE_TOKEN environment variable must be set.")
		os.Exit(1)
	}

	// Get list of files from GitLab
	filePaths, err := getFilePathsFromGitLabDirectory(*repoID, *ref, gitlabURL, privateToken)
	if err != nil {
		fmt.Printf("Failed to retrieve file paths from repository: %v\n", err)
		os.Exit(1)
	}

	// Compress files into a ZIP
	artifact, err := compressFiles(filePaths)
	if err != nil {
		fmt.Printf("Failed to compress files: %v\n", err)
		os.Exit(1)
	}

	// Push the compressed artifact to ACR using ORAS
	err = pushToACR(*acrURL, *repository, *tag, *username, *password, artifact)
	if err != nil {
		fmt.Printf("Failed to push to ACR: %v\n", err)
		os.Exit(1)
	}
}
