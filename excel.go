package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ParseExcelSheet 解析Excel文件的指定工作表
func (a *App) ParseExcelSheet(fileData string, sheetIndex int) ([]ExcelRow, error) {
	// 先保存Excel数据到临时文件
	filePath, err := a.SaveExcelData(fileData)
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
	return []ExcelRow{}, nil
}

// ProcessExcelData 处理前端发送的Excel数据
func (a *App) ProcessExcelData(rows []ExcelRow, username, password, urlTemplate string, algorithmType int) []CameraConfigResult {
	return a.ConfigureCamerasFromData(rows, username, password, urlTemplate, algorithmType)
}

// CleanupTempFiles 清理临时文件
func (a *App) CleanupTempFiles() error {
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
	files, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("无法读取临时目录内容: %w", err)
	}

	// 清理超过1小时的临时文件
	for _, file := range files {
		if !file.IsDir() && now.Sub(file.ModTime()) > time.Hour {
			filePath := filepath.Join(tempDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("警告: 无法删除临时文件 %s: %v\n", filePath, err)
			}
		}
	}

	return nil
}

// 在应用启动时清理临时文件
func init() {
	// 在应用启动时注册一个定时器，每小时清理一次临时文件
	go func() {
		for {
			// 创建一个新的App实例用于清理临时文件
			app := &App{}
			if err := app.CleanupTempFiles(); err != nil {
				fmt.Printf("清理临时文件失败: %v\n", err)
			}

			// 每小时执行一次
			time.Sleep(time.Hour)
		}
	}()
}
