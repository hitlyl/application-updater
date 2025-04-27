package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateTimestampedDirectory creates a directory with the current timestamp
func CreateTimestampedDirectory(basePath string, prefix string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	dirName := fmt.Sprintf("%s_%s", prefix, timestamp)
	dirPath := filepath.Join(basePath, dirName)

	if err := EnsureDirExists(dirPath); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return dirPath, nil
}

// ZipDirectory creates a zip archive from a directory
func ZipDirectory(source, destination string) error {
	zipFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	baseDir := filepath.Base(source)

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header: %w", err)
		}

		// Calculate relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		if relPath == "." {
			return nil
		}

		// Update header for directories
		if info.IsDir() {
			header.Name = filepath.Join(baseDir, relPath) + "/"
			header.Method = zip.Store
			_, err = archive.CreateHeader(header)
			return err
		}

		// Update header for files
		header.Name = filepath.Join(baseDir, relPath)
		header.Method = zip.Deflate

		// Create file in archive
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create file in archive: %w", err)
		}

		// Copy file contents to archive
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}

// UnzipFile extracts a zip file to a destination directory
func UnzipFile(zipPath, dstDir string) error {
	// Open the zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Create destination directory if it doesn't exist
	if err := EnsureDirExists(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract each file
	for _, file := range reader.File {
		// Handle directory entries
		if file.FileInfo().IsDir() {
			dirPath := filepath.Join(dstDir, file.Name)
			if err := EnsureDirExists(dirPath); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Handle file entries
		if err := extractZipFile(file, dstDir); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(file *zip.File, destination string) error {
	// Sanitize file path to prevent zip slip vulnerability
	filePath := filepath.Join(destination, file.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	// Create directory for file if needed
	if err := EnsureDirExists(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("failed to create directory for file: %w", err)
	}

	// Create file
	dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dstFile.Close()

	// Open file in zip
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer srcFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to extract file: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// ListFilesInDirectory lists all files in a directory matching a pattern
func ListFilesInDirectory(dir string, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return matches, nil
}
