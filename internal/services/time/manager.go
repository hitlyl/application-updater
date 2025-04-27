package time

import (
	"application-updater/internal/models"
)

// Manager defines the interface for time synchronization operations
type Manager interface {
	// SyncDeviceTime synchronizes time across multiple devices
	SyncDeviceTime(username, password string, deviceIPs []string) []models.TimeSyncResult
}

// Ensure Service implements Manager
var _ Manager = (*Service)(nil)
