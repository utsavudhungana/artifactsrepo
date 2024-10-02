package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Read .tar.gz file from local folder
func readTarGzFileFromLocal(filePath string) ([]byte, error) {
	// Open the .tar.gz file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local .tar.gz file: %v", err)
	}
	defer file.Close()

	// Read the file content into a byte slice
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	fileSize := fileInfo.Size()
	data := make([]byte, fileSize)
	_, err = io.ReadFull(file, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read local .tar.gz file: %v", err)
	}

	return data, nil
}

// Unzip .tar.gz artifacts and process only .json files
func unzipArtifacts(data []byte) (map[string][]byte, error) {
	artifactMap := make(map[string][]byte)

	// Create a GZIP reader
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create GZIP reader: %v", err)
	}
	defer gzipReader.Close()

	// Create a TAR reader from the GZIP stream
	tarReader := tar.NewReader(gzipReader)

	// Iterate through the files in the TAR archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// Reached the end of the archive
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %v", err)
		}

		// If the file type is not a regular file, skip it
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Process only .json files
		if filepath.Ext(header.Name) == ".json" {
			// Read the content of the file
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tarReader); err != nil {
				return nil, fmt.Errorf("failed to read file %s from tar.gz: %v", header.Name, err)
			}

			// Save the file content to the map
			artifactMap[header.Name] = buf.Bytes()

			// Print the name of the extracted .json file
			fmt.Printf("Extracted file: %s\n", header.Name)
		}
	}

	return artifactMap, nil
}

// Replace strings in .json files based on provided key-value pairs
func replaceStringsInJSONFiles(artifactMap map[string][]byte, replacements map[string]string) {
	for fileName, content := range artifactMap {
		// Convert content to string and replace
		contentStr := string(content)
		for oldStr, newStr := range replacements {
			contentStr = strings.ReplaceAll(contentStr, oldStr, newStr)
		}
		// Update the artifact map with modified content
		artifactMap[fileName] = []byte(contentStr)
	}
}

// Load replacements from a YAML configuration file
func loadReplacementsFromYAML(yamlFilePath string) (map[string]string, error) {
	// Read the YAML file
	data, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %v", err)
	}

	// Define a structure to hold the parsed YAML data
	var config struct {
		Replacements map[string]string `yaml:"replacements"`
	}

	// Parse the YAML file into the config structure
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML file: %v", err)
	}

	return config.Replacements, nil
}

// Main function to process the artifacts
func main() {
	// Variables
	tarGzFilePath := "path/to/your/artifacts.tar.gz" // Replace with the path to your local tar.gz file
	yamlFilePath := "path/to/your/config.yaml"       // Replace with the path to your YAML file

	// Read the .tar.gz file
	data, err := readTarGzFileFromLocal(tarGzFilePath)
	if err != nil {
		fmt.Printf("Error reading tar.gz file: %v\n", err)
		return
	}

	// Unzip the artifacts
	artifactMap, err := unzipArtifacts(data)
	if err != nil {
		fmt.Printf("Error unzipping artifacts: %v\n", err)
		return
	}

	// Load key-value pairs for string replacements from YAML
	replacements, err := loadReplacementsFromYAML(yamlFilePath)
	if err != nil {
		fmt.Printf("Error loading replacements from YAML: %v\n", err)
		return
	}

	// Replace strings in .json files
	replaceStringsInJSONFiles(artifactMap, replacements)

	// Print modified artifacts for verification
	for fileName, content := range artifactMap {
		fmt.Printf("Modified content of %s:\n%s\n", fileName, string(content))
	}
}
