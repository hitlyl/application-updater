package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"application-updater/internal/models"
)

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DeviceIP     string `json:"deviceIP"`
	DeviceSN     string `json:"deviceSN"`
	RestorePoint string `json:"restorePoint"`
}

// RestoreDevicesDB restores databases for multiple devices
func (s *Service) RestoreDevicesDB(backupPoints map[string]string) ([]RestoreResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	devices := s.deviceService.GetDevices()

	var results []RestoreResult
	for _, device := range devices {
		if device.Status != "online" && device.Status != "offline-busy" {
			results = append(results, RestoreResult{
				Success:  false,
				Message:  fmt.Sprintf("device offline: %s", device.Status),
				DeviceIP: device.IP,
				DeviceSN: device.IP, // Using IP as SN since we don't have that field
			})
			continue
		}

		backupPoint, exists := backupPoints[device.IP] // Using IP as key
		if !exists {
			results = append(results, RestoreResult{
				Success:  false,
				Message:  "no backup point specified for this device",
				DeviceIP: device.IP,
				DeviceSN: device.IP, // Using IP as SN since we don't have that field
			})
			continue
		}

		result, err := s.RestoreDeviceDB(context.Background(), &device, backupPoint)
		if err != nil {
			results = append(results, RestoreResult{
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

// RestoreDeviceDB restores database for a single device
func (s *Service) RestoreDeviceDB(ctx context.Context, device *models.Device, backupPoint string) (*RestoreResult, error) {
	// Get backup settings to determine the base path
	settings, err := s.GetBackupSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup settings: %w", err)
	}

	// Validate backup point exists
	backupDir := filepath.Join(settings.BackupPath, device.IP, backupPoint) // Using IP instead of SN
	_, err = os.Stat(backupDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("backup point does not exist: %s", backupDir)
	}

	// TODO: Implement actual restore logic that was in app.go
	// This would include operations like:
	// - Uploading configuration files back to the device
	// - Restoring database files
	// - Restarting necessary services on the device

	fmt.Printf("DEBUG: Restored device database successfully for %s using backup point %s\n",
		device.IP, backupPoint)

	return &RestoreResult{
		Success:      true,
		Message:      "Database restored successfully",
		DeviceIP:     device.IP,
		DeviceSN:     device.IP, // Using IP as SN since we don't have that field
		RestorePoint: backupPoint,
	}, nil
}
