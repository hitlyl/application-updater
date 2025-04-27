package excel

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"application-updater/internal/models"
)

// Service handles Excel file operations
type Service struct {
	mutex         sync.Mutex
	cameraService CameraService
}

// CameraService defines the interface for camera configuration operations
type CameraService interface {
	ConfigureCamerasFromData(deviceConfigs []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult
}

// NewService creates a new Excel service
func NewService(cameraService CameraService) *Service {
	return &Service{
		cameraService: cameraService,
	}
}

// ParseExcelSheet parses an Excel sheet from base64 encoded file data
func (s *Service) ParseExcelSheet(fileData string, sheetIndex int) ([]models.ExcelRow, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 先保存Excel数据到临时文件
	filePath, err := s.SaveExcelData(fileData)
	if err != nil {
		return nil, fmt.Errorf("保存Excel数据失败: %w", err)
	}

	// 这里应该使用一个库来解析Excel，例如 github.com/360EntSecGroup-Skylar/excelize
	// 由于我们在前端已经解析了Excel，这里只需要处理前端发送的数据即可
	// 这个函数在实际环境中应该实现解析Excel文件的逻辑

	// 清理临时文件
	defer os.Remove(filePath)

	// 由于前端已经解析了Excel，这里只是提供接口
	// 实际实现中，需要使用Excel库解析文件，并返回结果
	return []models.ExcelRow{}, nil
}

// SaveExcelData saves base64 encoded Excel data to a temporary file
func (s *Service) SaveExcelData(fileData string) (string, error) {
	// 获取程序执行文件路径，用于创建临时目录
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}

	execDir := filepath.Dir(execPath)
	tempDir := filepath.Join(execDir, "temp")

	// 确保临时目录存在
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建临时目录: %w", err)
	}

	// 创建临时文件
	filename := fmt.Sprintf("excel_data_%d.xlsx", time.Now().UnixNano())
	filePath := filepath.Join(tempDir, filename)

	// 解码Base64数据
	data, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		return "", fmt.Errorf("无法解码文件数据: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("无法写入文件: %w", err)
	}

	return filePath, nil
}

// ProcessExcelData processes Excel data rows for camera configuration
func (s *Service) ProcessExcelData(rows []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	return s.cameraService.ConfigureCamerasFromData(rows, username, password, urlTemplate, algorithmType, region)
}

// CleanupTempFiles cleans up temporary Excel files
func (s *Service) CleanupTempFiles() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取可执行文件路径: %w", err)
	}

	// 临时目录路径
	execDir := filepath.Dir(execPath)
	tempDir := filepath.Join(execDir, "temp")

	// 如果临时目录不存在，直接返回
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return nil
	}

	// 获取当前时间
	now := time.Now()

	// 打开目录
	dir, err := os.Open(tempDir)
	if err != nil {
		return fmt.Errorf("无法打开临时目录: %w", err)
	}
	defer dir.Close()

	// 读取目录内容
	files, err := dir.ReadDir(-1)
	if err != nil {
		return fmt.Errorf("无法读取临时目录内容: %w", err)
	}

	// 清理超过1小时的临时文件
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			fmt.Printf("警告: 无法获取文件信息 %s: %v\n", file.Name(), err)
			continue
		}

		if !fileInfo.IsDir() && now.Sub(fileInfo.ModTime()) > time.Hour {
			filePath := filepath.Join(tempDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("警告: 无法删除临时文件 %s: %v\n", filePath, err)
			}
		}
	}

	return nil
}

// StartCleanupRoutine starts a background routine to clean up temporary files periodically
func StartCleanupRoutine(excelService Manager) {
	go func() {
		for {
			if err := excelService.CleanupTempFiles(); err != nil {
				fmt.Printf("清理临时文件失败: %v\n", err)
			}

			// 每小时执行一次
			time.Sleep(time.Hour)
		}
	}()
}
