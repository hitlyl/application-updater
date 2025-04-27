package device

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"application-updater/internal/models"

	_ "github.com/mattn/go-sqlite3" // SQLite驱动
)

// Scanner 接口定义设备扫描功能
type Scanner interface {
	ScanIPRange(ctx context.Context, startIP, endIP string) []models.Device
	TestDevice(ip string) (*models.Device, error)
}

// Service 设备服务
type Service struct {
	Scanner         Scanner
	Auth            *Auth
	mutex           sync.RWMutex
	currentRegion   string
	filteredDevices []models.Device

	// 原Manager字段
	configDir string
	db        *sql.DB
}

// NewService 创建设备服务实例
func NewService(configDir string) *Service {
	client := &http.Client{}

	service := &Service{
		Scanner:         NewScanner(client),
		Auth:            NewAuth(client),
		filteredDevices: []models.Device{},
		configDir:       configDir,
	}

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("创建配置目录失败: %v\n", err)
	}

	// 初始化数据库
	if err := service.initDatabase(); err != nil {
		fmt.Printf("初始化数据库失败: %v\n", err)
	}

	// 加载设备列表
	if err := service.LoadDevices(); err != nil {
		fmt.Printf("加载设备列表失败: %v\n", err)
	}

	return service
}

// initDatabase 初始化SQLite数据库
func (s *Service) initDatabase() error {
	dbPath := filepath.Join(s.configDir, "devices.db")
	var err error

	// 打开数据库连接
	s.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建设备表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		ip TEXT NOT NULL,
		build_time TEXT,
		status TEXT,
		region TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices(ip);
	`

	_, err = s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("创建数据库表失败: %w", err)
	}

	return nil
}
func (s *Service) GetAllRegions() []string {
	rows, err := s.db.Query("SELECT DISTINCT region FROM devices")
	if err != nil {
		fmt.Printf("查询区域失败: %v\n", err)
		return []string{}
	}
	defer rows.Close()
	fmt.Printf("查询区域成功: %v\n", rows)
	regions := []string{}
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			continue
		}
		if region != "" {
			regions = append(regions, region)
		}
	}
	return regions
}

// GetDevices 获取所有设备
func (s *Service) GetDevices() []models.Device {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 如果有过滤区域，返回过滤后的设备列表
	if s.currentRegion != "" {
		return append([]models.Device{}, s.filteredDevices...)
	}

	// 否则返回所有设备
	return s.getAllDevicesFromDB()
}

// getAllDevicesFromDB 直接从数据库获取所有设备
func (s *Service) getAllDevicesFromDB() []models.Device {
	// 直接从数据库查询所有设备
	rows, err := s.db.Query("SELECT id, ip, build_time, status, region FROM devices")
	if err != nil {
		fmt.Printf("查询设备失败: %v\n", err)
		return []models.Device{}
	}
	defer rows.Close()

	devices := []models.Device{}
	for rows.Next() {
		var device models.Device
		err := rows.Scan(&device.ID, &device.IP, &device.BuildTime, &device.Status, &device.Region)
		if err != nil {
			fmt.Printf("扫描设备记录失败: %v\n", err)
			continue
		}
		devices = append(devices, device)
	}

	return devices
}

// GetAllDevices 获取所有设备，不考虑过滤
func (s *Service) GetAllDevices() []models.Device {
	return s.getAllDevicesFromDB()
}

// SetRegionFilter 设置区域过滤
func (s *Service) SetRegionFilter(region string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.currentRegion = region

	// 如果区域为空，清空过滤设备列表
	if region == "" {
		s.filteredDevices = []models.Device{}
		return
	}

	// 获取所有设备
	allDevices := s.getAllDevicesFromDB()

	// 过滤设备，包含指定区域和空区域的设备
	s.filteredDevices = make([]models.Device, 0, len(allDevices))
	for _, device := range allDevices {
		if device.Region == region || device.Region == "" {
			s.filteredDevices = append(s.filteredDevices, device)
		}
	}

	fmt.Printf("已过滤区域 %s 的设备(包含无区域设备)，共找到 %d 个设备\n", region, len(s.filteredDevices))
}

// GetCurrentRegion 获取当前过滤区域
func (s *Service) GetCurrentRegion() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentRegion
}

// RefreshDevices 刷新设备状态
func (s *Service) RefreshDevices() []models.Device {
	// 先解锁获取所有设备
	allDevices := s.GetDevices()

	var refreshedDevices []models.Device
	// 使用Scanner刷新所有设备状态
	if scanner, ok := s.Scanner.(*DeviceScanner); ok {
		refreshedDevices = scanner.RefreshDevices(allDevices)

		// 获取锁后更新数据库
		s.mutex.Lock()
		for _, device := range refreshedDevices {
			_, err := s.db.Exec("UPDATE devices SET status = ? WHERE id = ?", device.Status, device.ID)
			if err != nil {
				fmt.Printf("更新设备 %s 状态失败: %v\n", device.ID, err)
			}
		}

		// 如果有设置区域过滤，重新应用过滤
		if s.currentRegion != "" {
			s.filteredDevices = make([]models.Device, 0, len(refreshedDevices))
			for _, device := range refreshedDevices {
				if device.Region == s.currentRegion || device.Region == "" {
					s.filteredDevices = append(s.filteredDevices, device)
				}
			}
			filtered := append([]models.Device{}, s.filteredDevices...)
			s.mutex.Unlock()
			return filtered
		}
		s.mutex.Unlock()
		return refreshedDevices
	}

	return allDevices
}

// SaveDevices 保存设备列表，在使用数据库的情况下不再需要，但为了兼容性保留
func (s *Service) SaveDevices() error {
	// 数据库操作无需保存文件
	return nil
}

// LoadDevices 从数据库加载设备列表，兼容旧的JSON文件导入
func (s *Service) LoadDevices() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fmt.Println("开始加载设备列表...")

	// 检查数据库中是否有设备
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil {
		fmt.Printf("查询设备数量失败: %v\n", err)
		count = 0
	}

	// 检查JSON文件是否存在
	jsonPath := filepath.Join(s.configDir, "devices.json")
	jsonExists := false

	if _, err := os.Stat(jsonPath); err == nil {
		jsonExists = true
		fmt.Printf("检测到旧的devices.json文件: %s\n", jsonPath)
	} else {
		fmt.Printf("未检测到旧的devices.json文件: %v\n", err)
	}

	// 如果存在JSON文件且数据库中没有设备，则从JSON导入
	if jsonExists && count == 0 {
		fmt.Printf("数据库中没有设备，正在从JSON文件导入...\n")

		// 尝试读取JSON文件
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			fmt.Printf("读取旧的JSON文件失败: %v\n", err)
			return nil
		}

		// 如果文件为空，返回空列表
		if len(data) == 0 {
			fmt.Printf("JSON文件为空\n")
			return nil
		}

		// 临时设备列表，用于解析JSON
		var jsonDevices []models.Device

		// 解析JSON
		if err := json.Unmarshal(data, &jsonDevices); err != nil {
			fmt.Printf("解析JSON文件失败: %v\n", err)
			return nil
		}

		// 确保每个设备都有ID
		for i := range jsonDevices {
			if jsonDevices[i].ID == "" {
				jsonDevices[i].ID = models.GenerateDeviceID(jsonDevices[i].Region, jsonDevices[i].IP)
			}
		}

		fmt.Printf("从JSON解析了 %d 个设备\n", len(jsonDevices))

		// 准备插入语句
		stmt, err := s.db.Prepare("INSERT INTO devices (id, ip, build_time, status, region) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			fmt.Printf("准备插入语句失败: %v\n", err)
			return err
		}
		defer stmt.Close()

		// 开始事务
		tx, err := s.db.Begin()
		if err != nil {
			fmt.Printf("开始事务失败: %v\n", err)
			return err
		}

		// 将设备导入数据库
		for _, device := range jsonDevices {
			_, err = tx.Stmt(stmt).Exec(device.ID, device.IP, device.BuildTime, device.Status, device.Region)
			if err != nil {
				tx.Rollback()
				fmt.Printf("插入设备记录失败: %v\n", err)
				return err
			}
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			fmt.Printf("提交事务失败: %v\n", err)
			return err
		}

		// 备份旧的JSON文件
		backupPath := filepath.Join(s.configDir, fmt.Sprintf("devices.json.bak.%d", time.Now().Unix()))
		if err := os.Rename(jsonPath, backupPath); err != nil {
			fmt.Printf("警告: 备份旧的JSON文件失败: %v\n", err)
		} else {
			fmt.Printf("成功将旧的JSON文件备份为: %s\n", backupPath)
		}

		fmt.Printf("成功从JSON导入 %d 个设备到数据库\n", len(jsonDevices))
	} else if jsonExists {
		fmt.Printf("数据库中已有 %d 个设备，不需要从JSON导入\n", count)

		// 备份旧的JSON文件，即使不导入也应该备份
		backupPath := filepath.Join(s.configDir, fmt.Sprintf("devices.json.bak.%d", time.Now().Unix()))
		if err := os.Rename(jsonPath, backupPath); err != nil {
			fmt.Printf("警告: 备份旧的JSON文件失败: %v\n", err)
		} else {
			fmt.Printf("成功将旧的JSON文件备份为: %s\n", backupPath)
		}
	}

	return nil
}

// AddDevice 添加设备
func (s *Service) AddDevice(device models.Device) (models.Device, error) {
	// 确保设备ID已设置
	if device.ID == "" {
		device.ID = models.GenerateDeviceID(device.Region, device.IP)
	}

	// 添加设备到数据库
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查设备是否已存在
	var existingID string
	err := s.db.QueryRow("SELECT id FROM devices WHERE id = ?", device.ID).Scan(&existingID)

	if err == nil {
		// 设备已存在，更新记录
		_, err = s.db.Exec(
			"UPDATE devices SET ip = ?, build_time = ?, status = ?, region = ? WHERE id = ?",
			device.IP, device.BuildTime, device.Status, device.Region, device.ID)
		if err != nil {
			return models.Device{}, fmt.Errorf("更新设备失败: %w", err)
		}
	} else {
		// 设备不存在，插入新记录
		_, err = s.db.Exec(
			"INSERT INTO devices (id, ip, build_time, status, region) VALUES (?, ?, ?, ?, ?)",
			device.ID, device.IP, device.BuildTime, device.Status, device.Region)
		if err != nil {
			return models.Device{}, fmt.Errorf("添加设备失败: %w", err)
		}
	}

	// 如果有区域过滤且设备属于该区域或无区域，更新过滤后的设备列表
	if s.currentRegion != "" && (device.Region == s.currentRegion || device.Region == "") {
		// 检查设备是否已存在于过滤列表中
		found := false
		for i, d := range s.filteredDevices {
			if d.ID == device.ID {
				s.filteredDevices[i] = device
				found = true
				break
			}
		}

		// 如果不存在，添加到过滤列表
		if !found {
			s.filteredDevices = append(s.filteredDevices, device)
		}
	}

	return device, nil
}

// TestAndAddDevice 测试设备是否在线并添加设备
func (s *Service) TestAndAddDevice(ip string, region string) (models.Device, error) {
	// 首先测试设备是否在线
	device, err := s.Scanner.TestDevice(ip)
	if err != nil {
		return models.Device{}, fmt.Errorf("设备测试失败: %w", err)
	}

	// 设置设备区域
	device.Region = region

	// 确保设备ID已设置
	if device.ID == "" {
		device.ID = models.GenerateDeviceID(device.Region, device.IP)
	}

	// 添加设备到列表
	return s.AddDevice(*device)
}

// RemoveDevice 移除设备
func (s *Service) RemoveDevice(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 从数据库删除设备
	_, err := s.db.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("从数据库删除设备失败: %w", err)
	}

	// 如果有区域过滤，从过滤后的设备列表中移除
	if s.currentRegion != "" {
		newFiltered := make([]models.Device, 0, len(s.filteredDevices))
		for _, device := range s.filteredDevices {
			if device.ID != id {
				newFiltered = append(newFiltered, device)
			}
		}
		s.filteredDevices = newFiltered
	}

	return nil
}
func (s *Service) GetDeviceByRegionAndIP(region string, ip string) (models.Device, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var device models.Device
	err := s.db.QueryRow(
		"SELECT id, ip, build_time, status, region FROM devices WHERE region = ? AND ip = ?",
		region, ip).Scan(&device.ID, &device.IP, &device.BuildTime, &device.Status, &device.Region)

	if err != nil {
		return models.Device{}, false
	}

	return device, true
}

// SetDeviceRegion 设置设备区域
func (s *Service) SetDeviceRegion(deviceID, region string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 查找指定ID的设备
	var device models.Device
	err := s.db.QueryRow(
		"SELECT id, ip, build_time, status, region FROM devices WHERE id = ?",
		deviceID).Scan(&device.ID, &device.IP, &device.BuildTime, &device.Status, &device.Region)

	if err != nil {
		return fmt.Errorf("未找到ID为 %s 的设备: %w", deviceID, err)
	}

	// 保存原来的区域以便比较
	oldRegion := device.Region

	// 更新设备区域
	_, err = s.db.Exec("UPDATE devices SET region = ? WHERE id = ?", region, deviceID)
	if err != nil {
		return fmt.Errorf("更新设备区域失败: %w", err)
	}

	// 如果有区域过滤，更新过滤后的设备列表
	if s.currentRegion != "" {
		// 从过滤列表移除旧设备，如果设备区域既不是当前过滤区域也不是空区域
		if oldRegion == s.currentRegion && region != s.currentRegion && region != "" {
			newFiltered := make([]models.Device, 0, len(s.filteredDevices))
			for _, d := range s.filteredDevices {
				if d.ID != deviceID {
					newFiltered = append(newFiltered, d)
				}
			}
			s.filteredDevices = newFiltered
		}

		// 添加新设备到过滤列表（如果符合当前区域或无区域）
		if region == s.currentRegion || region == "" {
			// 获取更新后的设备
			var updatedDevice models.Device
			err := s.db.QueryRow(
				"SELECT id, ip, build_time, status, region FROM devices WHERE id = ?",
				deviceID).Scan(&updatedDevice.ID, &updatedDevice.IP, &updatedDevice.BuildTime, &updatedDevice.Status, &updatedDevice.Region)

			if err == nil {
				// 检查是否已存在
				exists := false
				for i, filteredDevice := range s.filteredDevices {
					if filteredDevice.ID == deviceID {
						// 更新现有设备
						s.filteredDevices[i] = updatedDevice
						exists = true
						break
					}
				}
				// 如果不存在，添加到过滤列表
				if !exists {
					s.filteredDevices = append(s.filteredDevices, updatedDevice)
				}
			}
		}
	}

	return nil
}

// SetDevicesRegion 批量设置设备区域
func (s *Service) SetDevicesRegion(deviceIDs []string, region string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 更新设备区域
	stmt, err := tx.Prepare("UPDATE devices SET region = ? WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("准备更新语句失败: %w", err)
	}
	defer stmt.Close()

	// 用于旧版本的IP地址兼容
	ipStmt, err := tx.Prepare("UPDATE devices SET region = ? WHERE ip = ?")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("准备IP更新语句失败: %w", err)
	}
	defer ipStmt.Close()

	for _, id := range deviceIDs {
		if isIPAddress(id) {
			// 如果是IP地址，使用IP进行更新
			_, err := ipStmt.Exec(region, id)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("使用IP更新区域失败: %w", err)
			}
		} else {
			// 否则使用ID更新
			_, err := stmt.Exec(region, id)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("更新区域失败: %w", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 更新过滤后的设备列表
	if s.currentRegion != "" {
		// 重新获取所有设备
		allDevices := s.getAllDevicesFromDB()

		// 过滤设备
		newFiltered := make([]models.Device, 0, len(allDevices))
		for _, device := range allDevices {
			if device.Region == s.currentRegion || device.Region == "" {
				newFiltered = append(newFiltered, device)
			}
		}
		s.filteredDevices = newFiltered
	}

	return nil
}

// isIPAddress 简单检查字符串是否看起来像IP地址
func isIPAddress(s string) bool {
	// 简单判断是否符合IP地址格式
	ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	return ipPattern.MatchString(s)
}

// ClearDevices 清空设备列表
func (s *Service) ClearDevices() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 从数据库删除所有设备
	_, err := s.db.Exec("DELETE FROM devices")
	if err != nil {
		return fmt.Errorf("清空设备列表失败: %w", err)
	}

	// 清空过滤后的设备列表
	s.filteredDevices = []models.Device{}

	return nil
}

// ScanIPRange delegates to the Scanner implementation to scan an IP range for devices.
// It enhances the result by setting device IDs and updating the device cache.
func (s *Service) ScanIPRange(ctx context.Context, startIP, endIP string) []models.Device {
	devices := s.Scanner.ScanIPRange(ctx, startIP, endIP)

	// 过滤掉没有IP的设备
	validDevices := make([]models.Device, 0, len(devices))
	for _, device := range devices {
		if device.IP == "" {
			fmt.Printf("警告: 扫描到没有IP的设备，已忽略\n")
			continue
		}
		validDevices = append(validDevices, device)
	}

	// 确保每个设备都有ID
	for i := range validDevices {
		if validDevices[i].ID == "" {
			validDevices[i].ID = models.GenerateDeviceID(validDevices[i].Region, validDevices[i].IP)
		}
	}

	// 添加或更新设备到数据库
	for i := range validDevices {
		deviceCopy := validDevices[i] // 创建副本以避免引用问题

		// 检查设备是否已存在 - 只有当region不为空时才根据region和IP查询
		var existingDevice models.Device
		var exists bool

		if deviceCopy.Region != "" {
			existingDevice, exists = s.GetDeviceByRegionAndIP(deviceCopy.Region, deviceCopy.IP)
		} else {
			// 如果region为空，只根据IP查询
			var rows *sql.Rows
			rows, err := s.db.Query("SELECT id, ip, build_time, status, region FROM devices WHERE ip = ?", deviceCopy.IP)
			if err == nil && rows.Next() {
				err = rows.Scan(&existingDevice.ID, &existingDevice.IP, &existingDevice.BuildTime, &existingDevice.Status, &existingDevice.Region)
				if err == nil {
					exists = true
				}
				rows.Close()
			}
		}

		if exists {
			// 更新状态但保留其他信息
			s.UpdateDeviceStatus(existingDevice.ID, deviceCopy.Status)
			// 确保返回列表中包含最新状态
			validDevices[i] = existingDevice
			validDevices[i].Status = deviceCopy.Status
		} else {
			// 添加新设备
			added, err := s.AddDevice(deviceCopy)
			if err != nil {
				fmt.Printf("添加设备失败 %s: %v\n", deviceCopy.IP, err)
			} else {
				// 用添加后的设备替换原来的设备（可能包含数据库生成的信息）
				validDevices[i] = added
			}
		}
	}

	// 如果有区域过滤，更新过滤后的设备列表
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.currentRegion != "" {
		// 添加新扫描的设备到过滤列表
		for _, device := range validDevices {
			if device.Region == s.currentRegion || device.Region == "" {
				// 检查是否已存在
				exists := false
				for i, filteredDevice := range s.filteredDevices {
					if filteredDevice.IP == device.IP {
						// 更新现有设备
						s.filteredDevices[i] = device
						exists = true
						break
					}
				}

				// 如果不存在，添加到过滤列表
				if !exists {
					s.filteredDevices = append(s.filteredDevices, device)
				}
			}
		}
	}

	return validDevices
}
func (s *Service) UpdateDeviceStatus(id string, status string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.db.Exec("UPDATE devices SET status = ? WHERE id = ?", status, id)
}

// TestDevice delegates to the Scanner implementation to test if a device is reachable.
func (s *Service) TestDevice(ip string) (*models.Device, error) {
	return s.Scanner.TestDevice(ip)
}

// LoginToDevice 登录到设备
func (s *Service) LoginToDevice(ip, username, password string) (string, error) {
	return s.Auth.LoginToDevice(ip, username, password)
}

// Close 关闭服务并释放资源
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetRegions 获取所有区域
func (s *Service) GetRegions() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	rows, err := s.db.Query("SELECT DISTINCT region FROM devices WHERE region != ''")
	if err != nil {
		return nil, fmt.Errorf("查询区域失败: %w", err)
	}
	defer rows.Close()

	regions := []string{}
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			continue
		}
		if region != "" {
			regions = append(regions, region)
		}
	}

	return regions, nil
}

// UpdateDevicesFile 上传更新文件到设备
func (s *Service) UpdateDevicesFile(filePath string, selectedBuildTime int64) ([]models.UpdateResult, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 检查MD5文件是否存在
	md5Path := filePath + ".md5"
	if _, err := os.Stat(md5Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("MD5文件不存在: %s", md5Path)
	}

	// 获取所有在线设备
	s.mutex.RLock()
	devices := make([]models.Device, 0)
	for _, device := range s.filteredDevices {
		// 跳过离线设备
		if device.Status != "online" {
			continue
		}

		// 如果指定了 BuildTime 则过滤设备
		if selectedBuildTime > 0 {
			// 转换 BuildTime 字符串为 int64 进行比较
			deviceBuildTime, err := strconv.ParseInt(device.BuildTime, 10, 64)
			if err == nil && deviceBuildTime < selectedBuildTime {
				devices = append(devices, device)
			}
		} else if selectedBuildTime <= 0 {
			// 如果没有指定 BuildTime，则添加所有在线设备
			devices = append(devices, device)
		}
	}
	s.mutex.RUnlock()

	if len(devices) == 0 {
		return nil, fmt.Errorf("没有需要更新的设备")
	}

	results := make([]models.UpdateResult, 0, len(devices))
	resultChan := make(chan models.UpdateResult, len(devices))

	// 限制并发数量为8
	maxConcurrent := 8
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	// 为每个设备启动一个goroutine执行更新
	for _, device := range devices {
		wg.Add(1)
		go func(device models.Device) {
			defer wg.Done()

			// 占用信号量
			semaphore <- struct{}{}
			defer func() {
				// 释放信号量
				<-semaphore
			}()

			result, err := s.uploadUpdateFile(device.IP, filePath)
			if err != nil {
				resultChan <- models.UpdateResult{
					IP:      device.IP,
					Success: false,
					Message: err.Error(),
				}
				return
			}
			resultChan <- result
		}(device)
	}

	// 等待所有更新完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		results = append(results, result)
	}

	return results, nil
}

// uploadUpdateFile 上传更新文件到单个设备
func (s *Service) uploadUpdateFile(ip string, filePath string) (models.UpdateResult, error) {
	result := models.UpdateResult{
		IP:      ip,
		Success: false,
		Message: "",
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	// 打开MD5文件
	md5File, err := os.Open(filePath + ".md5")
	if err != nil {
		return result, fmt.Errorf("无法打开MD5文件: %w", err)
	}
	defer md5File.Close()

	// 获取文件名
	fileName := filepath.Base(filePath)

	// 创建multipart表单
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 添加file字段
	fileWriter, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return result, fmt.Errorf("创建表单文件字段失败: %w", err)
	}
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return result, fmt.Errorf("复制文件到表单失败: %w", err)
	}

	// 添加md5字段
	md5Writer, err := writer.CreateFormFile("md5file", fileName+".md5")
	if err != nil {
		return result, fmt.Errorf("创建MD5表单字段失败: %w", err)
	}
	_, err = io.Copy(md5Writer, md5File)
	if err != nil {
		return result, fmt.Errorf("复制MD5文件到表单失败: %w", err)
	}

	// 关闭writer
	err = writer.Close()
	if err != nil {
		return result, fmt.Errorf("关闭表单writer失败: %w", err)
	}

	// 创建请求
	url := fmt.Sprintf("http://%s:8080/api/update", ip)
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return result, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{
		Timeout: 5 * time.Minute, // 设置较长的超时时间
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("更新失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 尝试解析JSON响应
	var jsonResp struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		if jsonResp.Status != 0 {
			return result, fmt.Errorf("更新失败: %s", jsonResp.Message)
		}
	}

	// 更新成功
	result.Success = true
	result.Message = "更新成功"
	return result, nil
}
