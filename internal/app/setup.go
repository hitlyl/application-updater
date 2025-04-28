package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"application-updater/internal/services/backup"
	"application-updater/internal/services/camera"
	"application-updater/internal/services/device"
	"application-updater/internal/services/excel"
	"application-updater/internal/services/time"
	"application-updater/internal/services/updater"
)

// Startup is the callback function when the application starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize config directory based on execution environment
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		return
	}

	execDir := filepath.Dir(execPath)
	a.configDir = filepath.Join(execDir, "config")
	fmt.Printf("Config directory: %s\n", a.configDir)

	// Ensure config directory exists
	if err := os.MkdirAll(a.configDir, 0755); err != nil {
		fmt.Printf("Failed to create config directory: %v\n", err)
		return
	}

	// Clean uploads directory of temporary files
	a.CleanUploadsDirectory()

	// Initialize services
	a.initServices()
}

// initServices initializes all service components
func (a *App) initServices() {
	// Initialize device service
	a.deviceService = device.NewService(a.configDir)

	// Load devices from storage
	err := a.deviceService.LoadDevices()
	if err != nil {
		fmt.Printf("Warning: Failed to load devices: %v\n", err)
	}

	// Initialize camera service
	a.cameraService = camera.NewService(a.client)

	// Create camera service adapter for excel service
	cameraAdapter := camera.NewCameraServiceAdapter(a.cameraService, a.deviceService.Auth)

	// Initialize excel service with camera adapter
	a.excelService = excel.NewService(cameraAdapter)

	// Initialize time service
	a.timeService = time.NewService()

	// Initialize backup and updater services with the device manager
	a.backupService = backup.NewService(a.deviceService.Manager)
	a.updaterService = updater.NewService(a.deviceService.Manager)

	fmt.Println("Services initialized successfully")
}
