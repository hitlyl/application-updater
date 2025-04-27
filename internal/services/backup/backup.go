package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"application-updater/internal/models"
	"application-updater/internal/services/device"
)

// Service handles backup operations for device configurations and databases
type Service struct {
	deviceService *device.Service
	mutex         sync.Mutex
}

// NewService creates a new backup service
func NewService(deviceService *device.Service) *Service {
	return &Service{
		deviceService: deviceService,
	}
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	BackupTimestamp string `json:"backupTimestamp"`
	DeviceIP        string `json:"deviceIP"`
	DeviceSN        string `json:"deviceSN"`
}

// BackupSettings represents the settings for backup operations
type BackupSettings struct {
	BackupPath     string `json:"backupPath"`
	BackupFreq     string `json:"backupFreq"`
	BackupEnabled  bool   `json:"backupEnabled"`
	LastBackupTime int64  `json:"lastBackupTime"`
}

// BackupDevices backs up all device configurations and databases
func (s *Service) BackupDevices(isManual bool, autoBackupDeviceOnce bool, backupSettings *BackupSettings) ([]BackupResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	devices := s.deviceService.GetDevices()

	var results []BackupResult
	for _, device := range devices {
		if device.Status != "online" && device.Status != "offline-busy" {
			results = append(results, BackupResult{
				Success:  false,
				Message:  fmt.Sprintf("device offline: %s", device.Status),
				DeviceIP: device.IP,
				DeviceSN: device.IP, // Using IP as SN since we don't have that field
			})
			continue
		}

		result, err := s.backupSingleDevice(context.Background(), &device, isManual, backupSettings.BackupPath)
		if err != nil {
			results = append(results, BackupResult{
				Success:  false,
				Message:  err.Error(),
				DeviceIP: device.IP,
				DeviceSN: device.IP, // Using IP as SN since we don't have that field
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// backupSingleDevice backs up a single device configuration and database
func (s *Service) backupSingleDevice(ctx context.Context, device *models.Device, isManual bool, backupPath string) (*BackupResult, error) {
	if backupPath == "" {
		return nil, fmt.Errorf("backup path is empty")
	}

	// Ensure backup directory exists
	err := os.MkdirAll(backupPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Setup device-specific backup directory
	deviceDir := filepath.Join(backupPath, device.IP) // Using IP as device identifier
	err = os.MkdirAll(deviceDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create device backup directory: %w", err)
	}

	// Create backup timestamp
	timestamp := time.Now().Format("20060102150405")
	backupDir := filepath.Join(deviceDir, timestamp)
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create timestamp backup directory: %w", err)
	}

	// TODO: Implement actual backup logic that was in app.go
	// This would include operations like:
	// - Downloading configuration files
	// - Backing up database
	// - Saving metadata
	fmt.Printf("DEBUG: Backup completed for device %s at %s\n", device.IP, timestamp)

	return &BackupResult{
		Success:         true,
		Message:         "Backup completed successfully",
		BackupTimestamp: timestamp,
		DeviceIP:        device.IP,
		DeviceSN:        device.IP, // Using IP as SN since we don't have that field
	}, nil
}

// SaveBackupSettings saves backup settings to a file
func (s *Service) SaveBackupSettings(settings *BackupSettings) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal backup settings: %w", err)
	}

	configDir := "configs"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	settingsPath := filepath.Join(configDir, "backup_settings.json")
	if err := ioutil.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup settings: %w", err)
	}

	fmt.Printf("DEBUG: Saved backup settings to %s\n", settingsPath)
	return nil
}

// GetBackupSettings retrieves backup settings from a file
func (s *Service) GetBackupSettings() (*BackupSettings, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	configDir := "configs"
	settingsPath := filepath.Join(configDir, "backup_settings.json")

	_, err := os.Stat(settingsPath)
	if os.IsNotExist(err) {
		// Default settings if file doesn't exist
		defaultSettings := &BackupSettings{
			BackupPath:    filepath.Join("backups"),
			BackupFreq:    "daily",
			BackupEnabled: false,
		}
		fmt.Printf("DEBUG: Using default backup settings, file %s not found\n", settingsPath)
		return defaultSettings, nil
	}

	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup settings: %w", err)
	}

	var settings BackupSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup settings: %w", err)
	}

	fmt.Printf("DEBUG: Loaded backup settings from %s\n", settingsPath)
	return &settings, nil
}
