package backup

import (
	"application-updater/internal/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

// RestoreDevicesDB restores databases for multiple devices
func (s *Service) RestoreDevicesDB(username, password, storageDir, areaDir string, selectIps []string) ([]models.RestoreResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var results []models.RestoreResult
	for _, ip := range selectIps {

		result, err := s.RestoreDeviceDB(context.Background(), ip, username, password, filepath.Join(storageDir, areaDir, ip))
		if err != nil {
			results = append(results, models.RestoreResult{
				Success: false,
				Message: err.Error(),
				IP:      ip,
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// RestoreDeviceDB restores database for a single device
func (s *Service) RestoreDeviceDB(ctx context.Context, ip string, username, password string, backupDir string) (*models.RestoreResult, error) {
	// Validate backup point exists
	_, err := os.Stat(backupDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("backup directory does not exist: %s", backupDir)
	}

	dbFilePath := filepath.Join(backupDir, "application-web.db")

	// Check if the database file exists
	_, err = os.Stat(dbFilePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("database file not found in backup point: %s", dbFilePath)
	}

	// Setup SSH connection to the device
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	// Connect to the device
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer client.Close()

	// 1. Stop the application service
	fmt.Printf("Stopping application service on %s...\n", ip)
	err = executeSSHCommand(client, "systemctl stop application-web")
	if err != nil {
		return nil, fmt.Errorf("failed to stop application service: %w", err)
	}

	// Wait a moment for the service to fully stop
	time.Sleep(2 * time.Second)

	// 2. Backup existing database on the device
	remoteDbPath := "/var/lib/application-web/db/application-web.db"
	backupCmd := fmt.Sprintf("cp %s %s.bak.%d",
		escapeShellArg(remoteDbPath),
		escapeShellArg(remoteDbPath),
		time.Now().Unix())

	err = executeSSHCommand(client, backupCmd)
	if err != nil {
		// Try to restart the service before returning
		_ = executeSSHCommand(client, "systemctl start application-web")
		return nil, fmt.Errorf("failed to backup existing database on device: %w", err)
	}

	// 3. Copy the backup database to the device
	err = scpFileToRemote(client, dbFilePath, remoteDbPath)
	if err != nil {
		// Try to restore from backup and restart the service
		restoreCmd := fmt.Sprintf("if [ -f %s.bak.* ]; then cp %s.bak.* %s; fi",
			escapeShellArg(remoteDbPath),
			escapeShellArg(remoteDbPath),
			escapeShellArg(remoteDbPath))
		_ = executeSSHCommand(client, restoreCmd)
		_ = executeSSHCommand(client, "systemctl start application-web")
		return nil, fmt.Errorf("failed to copy database to device: %w", err)
	}

	// 4. Restart the application service
	fmt.Printf("Restarting application service on %s...\n", ip)
	err = executeSSHCommand(client, "systemctl start application-web")
	if err != nil {
		return nil, fmt.Errorf("failed to restart application service: %w", err)
	}

	fmt.Printf("DEBUG: Restored device database successfully for %s using backup point %s\n",
		ip, backupDir)

	return &models.RestoreResult{
		Success:    true,
		Message:    "Database restored successfully",
		IP:         ip,
		BackupPath: backupDir,
	}, nil
}
