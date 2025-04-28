package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// EnsureDirExists ensures that a directory exists, creating it if necessary
func EnsureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// SetUploadFile sets the upload file path in a temporary location
func SetUploadFile(filename string, data []byte) (string, error) {
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, filename)

	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return filePath, nil
}

// GetUploadFile retrieves the file from the specified path
func GetUploadFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	return data, nil
}
