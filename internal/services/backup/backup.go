package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"application-updater/internal/services/device"

	"golang.org/x/crypto/ssh"
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
	DeviceSN        string `json:"deviceSN,omitempty"`
}

// BackupSettings represents the settings for backup operations
type BackupSettings struct {
	BackupPath     string `json:"backupPath"`
	BackupFreq     string `json:"backupFreq"`
	BackupEnabled  bool   `json:"backupEnabled"`
	LastBackupTime int64  `json:"lastBackupTime"`
}

// BackupDevices backs up all device configurations and databases
func (s *Service) BackupDevices(backupSettings *BackupSettings, username string, password string, selectIps []string) ([]BackupResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Default credentials if not provided
	if username == "" {
		username = "admin" // Default username
	}
	if password == "" {
		password = "admin" // Default password
	}

	var results []BackupResult
	for _, ip := range selectIps {

		result, err := s.backupSingleDevice(context.Background(), ip, backupSettings.BackupPath, username, password)
		if err != nil {
			results = append(results, BackupResult{
				Success:  false,
				Message:  err.Error(),
				DeviceIP: ip,
			})
			continue
		}
		results = append(results, *result)
	}

	// Update last backup time
	backupSettings.LastBackupTime = time.Now().Unix()
	if err := s.SaveBackupSettings(backupSettings); err != nil {
		fmt.Printf("Warning: Failed to update backup settings after backup: %v\n", err)
	}

	return results, nil
}

// backupSingleDevice backs up a single device configuration and database
func (s *Service) backupSingleDevice(ctx context.Context, ip string, backupPath string, username string, password string) (*BackupResult, error) {
	if backupPath == "" {
		return nil, fmt.Errorf("backup path is empty")
	}

	// Ensure backup directory exists
	err := os.MkdirAll(backupPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Setup device-specific backup directory
	deviceDir := filepath.Join(backupPath, ip) // Using IP as device identifier
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

	// Try to obtain authentication token - just to verify credentials
	_, err = s.deviceService.LoginToDevice(ip, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login to device: %w", err)
	}

	// Setup SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For development only! In production use proper host key verification
		Timeout:         30 * time.Second,
	}

	// Connect to the device
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer client.Close()

	// 1. First stop the application service
	fmt.Printf("Stopping application service on %s...\n", ip)
	err = executeSSHCommand(client, "systemctl stop application-web")
	if err != nil {
		return nil, fmt.Errorf("failed to stop application service: %w", err)
	}

	// Wait a moment for the service to fully stop
	time.Sleep(2 * time.Second)

	// 2. Copy the database file to a temporary location
	dbFilePath := "/var/lib/application-web/db/application-web.db"
	localDbPath := filepath.Join(backupDir, "application-web.db")

	// Create an SCP session to copy the file
	fmt.Printf("Downloading database from %s...\n", ip)
	err = scpFileFromRemote(client, dbFilePath, localDbPath)
	if err != nil {
		// Try to restart the service before returning error
		_ = executeSSHCommand(client, "systemctl start application-web")
		return nil, fmt.Errorf("failed to download database file: %w", err)
	}

	// 3. Restart the application service
	fmt.Printf("Restarting application service on %s...\n", ip)
	err = executeSSHCommand(client, "systemctl start application-web")
	if err != nil {
		return nil, fmt.Errorf("failed to restart application service: %w", err)
	}

	// 4. Create a metadata file with information about the backup
	metadataPath := filepath.Join(backupDir, "metadata.json")
	metadata := map[string]interface{}{
		"timestamp":   timestamp,
		"device_ip":   ip,
		"backup_type": "database",
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write backup metadata: %w", err)
	}

	return &BackupResult{
		Success:         true,
		Message:         "Backup completed successfully",
		BackupTimestamp: timestamp,
		DeviceIP:        ip,
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
