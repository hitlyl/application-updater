package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"application-updater/internal/models"
	"application-updater/internal/services/backup"
	"application-updater/internal/services/camera"
	"application-updater/internal/services/device"
	"application-updater/internal/services/excel"
	"application-updater/internal/services/time"
	"application-updater/internal/utils"
)

// CameraAdapter adapts camera.Service to excel.CameraService interface
type cameraAdapter struct {
	cameraService *camera.Service
}

// ConfigureCamerasFromData implements the excel.CameraService interface
func (a *cameraAdapter) ConfigureCamerasFromData(rows []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	// Create a token getter function for the camera service
	getTokenFunc := func(ip, user, pass string) (string, error) {
		return device.NewAuth(&http.Client{}).LoginToDevice(ip, user, pass)
	}

	// Call the camera service's method with the token getter and region
	return a.cameraService.Config.ConfigureCamerasFromData(rows, getTokenFunc, username, password, urlTemplate, algorithmType, region)
}

// App struct represents the main application
type App struct {
	ctx           context.Context
	configDir     string
	client        *http.Client
	mutex         sync.RWMutex
	deviceService *device.Service
	cameraService *camera.Service
	excelService  *excel.Service
	timeService   *time.Service
	backupService *backup.Service
}

// NewApp creates a new App instance
func NewApp() *App {
	// Create optimized HTTP client
	client := &http.Client{
		Transport: utils.CreateOptimizedTransport(),
		Timeout:   0, // No timeout, let each request control its own timeout
	}

	// Get config directory
	configDir := utils.GetConfigDir()

	// Initialize device service first since other services depend on it
	deviceService := device.NewService(configDir)

	// Initialize other services
	cameraService := camera.NewService(client)

	// Set device manager in camera service
	cameraService.SetDeviceService(deviceService)

	timeService := time.NewService()
	backupService := backup.NewService(deviceService)

	// Create adapter to bridge camera service to excel service
	cameraAdapterInstance := &cameraAdapter{cameraService: cameraService}

	// Excel service needs camera adapter for ConfigureCamerasFromData
	excelService := excel.NewService(cameraAdapterInstance)

	return &App{
		client:        client,
		configDir:     configDir,
		deviceService: deviceService,
		cameraService: cameraService,
		excelService:  excelService,
		timeService:   timeService,
		backupService: backupService,
	}
}

func (a *App) SelectFolder() (string, error) {
	return utils.SelectFolder(a.ctx)
}

// DomReady is called when the DOM is fully loaded
func (a *App) DomReady(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("DOM is ready")

	// Load devices from storage
	if a.deviceService != nil {
		err := a.deviceService.LoadDevices()
		if err != nil {
			fmt.Printf("Warning: Failed to load devices from storage: %v\n", err)
		} else {
			fmt.Println("Successfully loaded devices from storage")
		}
	}
}
func (a *App) ClearDevices() error {
	return a.deviceService.ClearDevices()
}

// Shutdown is called when the application is shutting down
func (a *App) Shutdown(ctx context.Context) {
	fmt.Println("Application is shutting down")

	// 关闭设备服务资源
	if a.deviceService != nil {
		if err := a.deviceService.Close(); err != nil {
			fmt.Printf("关闭设备服务时出错: %v\n", err)
		}
	}
}

// BeforeClose is called when the user attempts to close the application
func (a *App) BeforeClose(ctx context.Context) bool {
	fmt.Println("User is attempting to close the application")
	return false // Allow the application to close
}

// GetDevices returns all devices
func (a *App) GetDevices() []models.Device {
	return a.deviceService.GetDevices()
}

// GetAllDevices returns all devices without filtering
func (a *App) GetAllDevices() []models.Device {
	return a.deviceService.GetAllDevices()
}

// SetRegionFilter sets the region filter for devices
func (a *App) SetRegionFilter(region string) []models.Device {
	a.deviceService.SetRegionFilter(region)
	return a.deviceService.GetDevices()
}

// GetCurrentRegion returns the current region filter
func (a *App) GetCurrentRegion() string {
	return a.deviceService.GetCurrentRegion()
}

// RefreshDevices refreshes device status
func (a *App) RefreshDevices() []models.Device {
	return a.deviceService.RefreshDevices()
}

// AddDevice adds a new device
func (a *App) AddDevice(ip string, region string) (models.Device, error) {
	// 调用设备服务的TestAndAddDevice方法，完成测试和添加
	device, err := a.deviceService.TestAndAddDevice(ip, region)
	if err != nil {
		return models.Device{}, fmt.Errorf("设备测试或添加失败: %w", err)
	}
	return device, nil
}

// RemoveDevice removes a device by ID
func (a *App) RemoveDevice(deviceID string) error {
	return a.deviceService.RemoveDevice(deviceID)
}

// LoginToDevice tests login credentials for a device
func (a *App) LoginToDevice(ip, username, password string) (bool, string) {
	token, err := a.deviceService.LoginToDevice(ip, username, password)
	if err != nil {
		return false, err.Error()
	}
	return true, token
}

// ScanIPRange scans an IP range for devices
func (a *App) ScanIPRange(startIP, endIP string) []models.Device {
	devices := a.deviceService.ScanIPRange(a.ctx, startIP, endIP)
	return devices
}

// ConfigureCamera configures a camera on a device
func (a *App) ConfigureCamera(ip, username, password, cameraName, cameraURL string, algorithmType int) (bool, string) {
	// 先登录获取token
	token, err := a.deviceService.LoginToDevice(ip, username, password)
	if err != nil {
		return false, fmt.Sprintf("登录失败: %v", err)
	}

	// 判断是新增还是修改
	existingCamera := false
	cameras, err := a.cameraService.GetCameraTasksWithToken(ip, token)
	if err == nil {
		for _, camera := range cameras {
			if camera.DeviceName == cameraName {
				existingCamera = true
				break
			}
		}
	}

	// 配置摄像头
	return a.cameraService.ConfigureCamera(ip, token, cameraName, cameraURL, algorithmType, existingCamera)
}

// GetCameraConfig gets camera configuration from a device
func (a *App) GetCameraConfig(ip, username, password, taskID string) (models.Camera, error) {
	// 先登录获取token
	token, err := a.deviceService.LoginToDevice(ip, username, password)
	if err != nil {
		return models.Camera{}, err
	}

	// 获取摄像头配置
	config, err := a.cameraService.GetCameraConfig(ip, token, taskID)
	if err != nil {
		return models.Camera{}, err
	}

	// 返回简化的摄像头信息
	return models.Camera{
		TaskID:     taskID,
		DeviceName: config.Device.Name,
		URL:        config.Device.URL,
	}, nil
}

// GetCameraTasks gets all camera tasks from a device
func (a *App) GetCameraTasks(ip, username, password string) ([]models.Camera, error) {
	// 创建用于获取token的函数
	getTokenFunc := func(deviceIP, user, pass string) (string, error) {
		return a.deviceService.LoginToDevice(deviceIP, user, pass)
	}

	// 获取摄像头任务列表
	return a.cameraService.GetCameraTasks(ip, username, password, getTokenFunc)
}

// SetCameraIndex sets the index of a camera
func (a *App) SetCameraIndex(ip, username, password, taskID string, index int) (bool, string) {
	// 先登录获取token
	token, err := a.deviceService.LoginToDevice(ip, username, password)
	if err != nil {
		return false, fmt.Sprintf("登录失败: %v", err)
	}

	// 获取摄像头配置
	config, err := a.cameraService.GetCameraConfig(ip, token, taskID)
	if err != nil {
		return false, fmt.Sprintf("获取配置失败: %v", err)
	}

	// 设置摄像头索引
	return a.cameraService.SetCameraIndex(ip, token, taskID, config, index)
}

// SyncDeviceTime synchronizes the time of devices with the current machine's time
func (a *App) SyncDeviceTime(username, password string, deviceIPs []string) []models.TimeSyncResult {
	return a.timeService.SyncDeviceTime(username, password, deviceIPs)
}

// ParseExcelSheet parses an Excel sheet from base64 encoded file data
func (a *App) ParseExcelSheet(fileData string, sheetIndex int) ([]models.ExcelRow, error) {
	return a.excelService.ParseExcelSheet(fileData, sheetIndex)
}

// SaveExcelData saves base64 encoded Excel data to a temporary file
func (a *App) SaveExcelData(fileData string) (string, error) {
	return a.excelService.SaveExcelData(fileData)
}

// ProcessExcelData processes Excel data rows for camera configuration
func (a *App) ProcessExcelData(rows []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	return a.excelService.ProcessExcelData(rows, username, password, urlTemplate, algorithmType, region)
}

// BackupDevices backs up the configuration and database of all devices
func (a *App) BackupDevices(username, password string, storageDir, areaDir string, selectIps []string) []models.BackupResult {
	// 从存储中获取备份设置
	settings, err := a.backupService.GetBackupSettings()
	if err != nil {
		fmt.Printf("Warning: Failed to get backup settings: %v, using defaults\n", err)
		settings = &models.BackupSettings{
			BackupPath: "backups",
			AreaPath:   "area1",
			Username:   "root",
			Password:   "ematech",
		}
	}
	settings.BackupPath = storageDir
	settings.AreaPath = areaDir
	a.backupService.SaveBackupSettings(settings)

	// 执行备份
	results, err := a.backupService.BackupDevices(settings, username, password, selectIps)
	if err != nil {
		fmt.Printf("Error performing backup: %v\n", err)
		return []models.BackupResult{}
	}

	// 转换结果类型
	modelResults := make([]models.BackupResult, len(results))
	for i, result := range results {
		modelResults[i] = models.BackupResult{
			IP:      result.IP,
			Success: result.Success,
			Message: result.Message,
		}
	}

	return modelResults
}

// RestoreDevicesDB restores device databases from backup
func (a *App) RestoreDevicesDB(username, password, storageDir, areaDir string, selectIps []string) []models.RestoreResult {
	results, err := a.backupService.RestoreDevicesDB(username, password, storageDir, areaDir, selectIps)
	if err != nil {
		fmt.Printf("Error performing restore: %v\n", err)
		return []models.RestoreResult{}
	}
	return results
}

// GetBackupSettings gets the current backup settings
func (a *App) GetBackupSettings() models.BackupSettings {
	// 从备份服务获取设置
	settings, err := a.backupService.GetBackupSettings()
	if err != nil {
		fmt.Printf("Warning: Failed to get backup settings: %v, using defaults\n", err)
		return models.BackupSettings{
			BackupPath: "backups",
			AreaPath:   "",
			Username:   "",
			Password:   "",
		}
	}
	return *settings
}

// SaveBackupSettings saves backup settings
func (a *App) SaveBackupSettings(settings models.BackupSettings) error {
	// 调用服务保存设置
	return a.backupService.SaveBackupSettings(&settings)
}

// GetRegions returns all device regions
func (a *App) GetRegions() []string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.deviceService == nil {
		return []string{}
	}
	return a.deviceService.GetAllRegions()

}

// UpdateDevicesFile uploads update files to devices with build time less than the selected build time
func (a *App) UpdateDevicesFile(deviceIds []string, fileName string, fileBinary []byte, md5FileName string, md5FileBinary []byte, username string, password string) ([]models.UpdateResult, error) {
	results, err := a.deviceService.UpdateDevicesFile(deviceIds, fileName, fileBinary, md5FileName, md5FileBinary, username, password)
	if err != nil {
		return nil, err
	}

	// Convert device.UpdateResult to models.UpdateResult
	modelResults := make([]models.UpdateResult, len(results))
	for i, result := range results {
		modelResults[i] = models.UpdateResult{
			IP:      result.IP,
			Success: result.Success,
			Message: result.Message,
		}
	}

	return modelResults, nil
}

// SetDevicesRegion sets the region for multiple devices
func (a *App) SetDevicesRegion(deviceIDs []string, region string) error {
	return a.deviceService.SetDevicesRegion(deviceIDs, region)
}

<<<<<<< HEAD
// syncSingleDeviceTime synchronizes the time of a single device
func (a *App) syncSingleDeviceTime(deviceIP, username, password, dateTimeString string, currentTime time.Time, workerID int) TimeSyncResult {
	result := TimeSyncResult{
		IP:        deviceIP,
		Success:   false,
		Message:   "",
		Timestamp: currentTime.Format("2006-01-02 15:04:05"),
	}

	// 创建SSH客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 在生产环境中应使用更安全的方法
		Timeout:         10 * time.Second,
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:22", deviceIP)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		fmt.Printf("ERROR: [Worker-%d] 连接设备 %s 失败: %v\n", workerID, deviceIP, err)
		result.Message = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer client.Close()

	fmt.Printf("DEBUG: [Worker-%d] 已成功连接到设备 %s\n", workerID, deviceIP)

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("ERROR: [Worker-%d] 创建SSH会话失败: %v\n", workerID, deviceIP, err)
		result.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
		return result
	}
	defer session.Close()

	// 设置输出缓冲区
	var stdoutBuffer, stderrBuffer bytes.Buffer
	session.Stdout = &stdoutBuffer
	session.Stderr = &stderrBuffer

	// 执行date命令设置系统时间
	// 格式: date MMDDHHmmYYYY.ss
	// 例如: date 010112002023.00 设置时间为 2023年1月1日12:00:00
	dateCommand := fmt.Sprintf("date %s%s%s%s%s.%s && hwclock -w",
		dateTimeString[4:6],   // 月
		dateTimeString[6:8],   // 日
		dateTimeString[8:10],  // 时
		dateTimeString[10:12], // 分
		dateTimeString[0:4],   // 年
		dateTimeString[12:14], // 秒
	)

	fmt.Printf("DEBUG: [Worker-%d] 执行命令: %s\n", workerID, dateCommand)

	err = session.Run(dateCommand)
	if err != nil {
		errMsg := stderrBuffer.String()
		fmt.Printf("ERROR: [Worker-%d] 设置时间失败: %v, 错误输出: %s\n", workerID, deviceIP, err, errMsg)
		result.Message = fmt.Sprintf("设置时间失败: %v, %s", err, errMsg)
		return result
	}

	output := stdoutBuffer.String()
	fmt.Printf("DEBUG: [Worker-%d] 设备 %s 时间设置成功，输出: %s\n", workerID, deviceIP, output)

	// 验证时间设置成功
	result.Success = true
	result.Message = "时间同步成功"
	return result
=======
// SetDeviceRegion 设置单个设备的区域
func (a *App) SetDeviceRegion(deviceID string, region string) error {
	return a.deviceService.SetDeviceRegion(deviceID, region)
>>>>>>> dev
}
