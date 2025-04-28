package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanUploadsDirectory cleans the uploads directory of temporary files
func (a *App) CleanUploadsDirectory() {
	// Get the executable file path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		return
	}

	// Build the uploads directory path
	execDir := filepath.Dir(execPath)
	uploadsDir := filepath.Join(execDir, "uploads")
	fmt.Printf("Uploads directory: %s\n", uploadsDir)

	// Ensure the uploads directory exists
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		fmt.Printf("Failed to create uploads directory: %v\n", err)
		return
	}

	// Clean files in the uploads directory
	a.CleanDirectory(uploadsDir)
}

// CleanDirectory removes all files and subdirectories in the specified directory
func (a *App) CleanDirectory(dirPath string) error {
	// Get all entries in the directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Iterate through and remove each entry
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		// If it's a directory, clean it recursively
		if entry.IsDir() {
			if err := a.CleanDirectory(entryPath); err != nil {
				fmt.Printf("Failed to clean subdirectory %s: %v\n", entryPath, err)
				continue
			}
			// After cleaning the subdirectory, try to remove it
			if err := os.Remove(entryPath); err != nil {
				fmt.Printf("Failed to remove subdirectory %s: %v\n", entryPath, err)
			}
		} else {
			// If it's a file, delete it directly
			if err := os.Remove(entryPath); err != nil {
				fmt.Printf("Failed to delete file %s: %v\n", entryPath, err)
			}
		}
	}

	fmt.Printf("Directory cleaned: %s\n", dirPath)
	return nil
}
