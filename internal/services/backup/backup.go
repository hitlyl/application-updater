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

// BackupDevices backs up all device configurations and databases
func (s *Service) BackupDevices(backupSettings *models.BackupSettings, username string, password string, selectIps []string) ([]models.BackupResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Default credentials if not provided
	if username == "" {
		username = "admin" // Default username
	}
	if password == "" {
		password = "admin" // Default password
	}

	var results []models.BackupResult
	for _, ip := range selectIps {

		result, err := s.backupSingleDevice(context.Background(), ip, filepath.Join(backupSettings.BackupPath, backupSettings.AreaPath, ip), username, password)
		if err != nil {
			results = append(results, models.BackupResult{
				Success: false,
				Message: err.Error(),
				IP:      ip,
			})
			continue
		}
		results = append(results, *result)
	}
	backupSettings.Username = username
	backupSettings.Password = password

	if err := s.SaveBackupSettings(backupSettings); err != nil {
		fmt.Printf("Warning: Failed to update backup settings after backup: %v\n", err)
	}

	return results, nil
}

// backupSingleDevice backs up a single device configuration and database
func (s *Service) backupSingleDevice(ctx context.Context, ip string, backupPath string, username string, password string) (*models.BackupResult, error) {
	if backupPath == "" {
		return nil, fmt.Errorf("backup path is empty")
	}

	// Ensure backup directory exists
	err := os.MkdirAll(backupPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
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
	localDbPath := filepath.Join(backupPath, "application-web.db")

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
	fmt.Printf("Backup completed successfully for %s\n", ip)

	return &models.BackupResult{
		Success: true,
		Message: "Backup completed successfully, please check the backup directory "+backupPath,
		IP:      ip,
	}, nil
}

// SaveBackupSettings saves backup settings to a file
func (s *Service) SaveBackupSettings(settings *models.BackupSettings) error {

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
func (s *Service) GetBackupSettings() (*models.BackupSettings, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	configDir := "configs"
	settingsPath := filepath.Join(configDir, "backup_settings.json")

	_, err := os.Stat(settingsPath)
	if os.IsNotExist(err) {
		// Default settings if file doesn't exist
		defaultSettings := &models.BackupSettings{
			BackupPath: filepath.Join("backups"),
			AreaPath:   "area1",
			Username:   "root",
			Password:   "ematech",
		}
		fmt.Printf("DEBUG: Using default backup settings, file %s not found\n", settingsPath)
		return defaultSettings, nil
	}

	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup settings: %w", err)
	}

	var settings models.BackupSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup settings: %w", err)
	}

	fmt.Printf("DEBUG: Loaded backup settings from %s\n", settingsPath)
	return &settings, nil
}
