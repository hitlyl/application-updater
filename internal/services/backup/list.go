package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BackupPoint represents a single backup point for a device
type BackupPoint struct {
	DeviceSN  string    `json:"deviceSN"`
	Timestamp string    `json:"timestamp"`
	DateTime  time.Time `json:"dateTime"`
	Path      string    `json:"path"`
}

// ListBackupPoints lists all available backup points for all devices
func (s *Service) ListBackupPoints() (map[string][]BackupPoint, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings, err := s.GetBackupSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup settings: %w", err)
	}

	if settings.BackupPath == "" {
		return nil, fmt.Errorf("backup path is not configured")
	}

	// Check if backup directory exists
	_, err = os.Stat(settings.BackupPath)
	if os.IsNotExist(err) {
		fmt.Printf("DEBUG: Backup directory does not exist, no backups available\n")
		return make(map[string][]BackupPoint), nil
	}

	// Map of device SN to its backup points
	backupPoints := make(map[string][]BackupPoint)

	// List device directories in backup path
	deviceDirs, err := os.ReadDir(settings.BackupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, deviceDir := range deviceDirs {
		if !deviceDir.IsDir() {
			continue
		}

		deviceID := deviceDir.Name() // This is the device IP
		devicePath := filepath.Join(settings.BackupPath, deviceID)

		// List backup timestamp directories for this device
		timestampDirs, err := os.ReadDir(devicePath)
		if err != nil {
			fmt.Printf("WARN: Failed to read device backup directory for %s: %v\n",
				deviceID, err)
			continue
		}

		var points []BackupPoint
		for _, timestampDir := range timestampDirs {
			if !timestampDir.IsDir() {
				continue
			}

			timestamp := timestampDir.Name()
			backupPath := filepath.Join(devicePath, timestamp)

			// Parse timestamp into a time.Time
			dateTime, err := time.Parse("20060102150405", timestamp)
			if err != nil {
				fmt.Printf("WARN: Failed to parse backup timestamp for device %s, timestamp %s: %v\n",
					deviceID, timestamp, err)
				continue
			}

			points = append(points, BackupPoint{
				DeviceSN:  deviceID, // Store the device IP here
				Timestamp: timestamp,
				DateTime:  dateTime,
				Path:      backupPath,
			})
		}

		// Sort backup points by timestamp (newest first)
		sort.Slice(points, func(i, j int) bool {
			return points[i].DateTime.After(points[j].DateTime)
		})

		if len(points) > 0 {
			backupPoints[deviceID] = points
		}
	}

	return backupPoints, nil
}

// GetDeviceBackupPoints lists available backup points for a specific device
func (s *Service) GetDeviceBackupPoints(deviceIP string) ([]BackupPoint, error) {
	allPoints, err := s.ListBackupPoints()
	if err != nil {
		return nil, err
	}

	points, exists := allPoints[deviceIP]
	if !exists {
		return []BackupPoint{}, nil
	}

	return points, nil
}

// DeleteBackupPoint deletes a specific backup point
func (s *Service) DeleteBackupPoint(deviceIP, timestamp string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings, err := s.GetBackupSettings()
	if err != nil {
		return fmt.Errorf("failed to get backup settings: %w", err)
	}

	backupPath := filepath.Join(settings.BackupPath, deviceIP, timestamp)
	_, err = os.Stat(backupPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("backup point does not exist: %s", backupPath)
	}

	err = os.RemoveAll(backupPath)
	if err != nil {
		return fmt.Errorf("failed to delete backup point: %w", err)
	}

	fmt.Printf("DEBUG: Deleted backup point for device %s, timestamp %s\n",
		deviceIP, timestamp)

	return nil
}
