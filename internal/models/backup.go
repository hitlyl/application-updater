package models

// BackupResult represents the result of a device backup operation
type BackupResult struct {
	IP         string `json:"ip"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	BackupPath string `json:"backupPath"`
}

// BackupSettings stores persistent settings for device backup
type BackupSettings struct {
	StorageFolder string `json:"storageFolder"`
	RegionName    string `json:"regionName"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

// RestoreResult represents the result of a device database restoration operation
type RestoreResult struct {
	IP           string `json:"ip"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath"`
}
