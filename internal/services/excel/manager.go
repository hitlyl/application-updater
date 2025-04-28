package excel

import (
	"application-updater/internal/models"
)

// Manager defines the interface for Excel processing operations
type Manager interface {
	// ParseExcelSheet parses an Excel sheet from base64 encoded file data
	ParseExcelSheet(fileData string, sheetIndex int) ([]models.ExcelRow, error)

	// SaveExcelData saves base64 encoded Excel data to a temporary file
	SaveExcelData(fileData string) (string, error)

	// ProcessExcelData processes Excel data rows for camera configuration
	ProcessExcelData(rows []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult

	// CleanupTempFiles cleans up temporary Excel files
	CleanupTempFiles() error
}

// Ensure Service implements Manager
var _ Manager = (*Service)(nil)
