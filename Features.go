package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func main() {
	// Example usage
	filePath := "path/to/your/tarfile.tar.gz"

	// Read the .tar.gz file into memory
	fileContent, err := readTarGzFileFromLocal(filePath)
	if err != nil {
		fmt.Printf("Error reading .tar.gz file: %v\n", err)
		return
	}

	// Extract files from the .tar.gz archive and process .json files only
	artifactMap, err := unzipArtifacts(fileContent)
	if err != nil {
		fmt.Printf("Error extracting artifacts: %v\n", err)
		return
	}

	// Process the extracted .json artifacts
	for name, content := range artifactMap {
		fmt.Printf("File: %s, Size: %d bytes\n", name, len(content))

		// Example: You can now handle the .json content here (e.g., publishing it or saving it)
		fmt.Printf("Content: %s\n", string(content)) // Print the content as a string
	}
}
