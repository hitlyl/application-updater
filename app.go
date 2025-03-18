package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Device represents a device in the system
type Device struct {
	IP        string `json:"ip"`
	BuildTime string `json:"buildTime"`
	Status    string `json:"status"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		Token string `json:"token"`
	} `json:"result"`
}

// BuildTimeResponse represents the buildTime response
type BuildTimeResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		BuildTime string `json:"buildTime"`
	} `json:"result"`
}

// UpdateResult represents the update operation result
type UpdateResult struct {
	IP      string `json:"ip"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Camera represents a camera configuration
type Camera struct {
	TaskID     string `json:"taskId"`
	DeviceName string `json:"deviceName"`
	URL        string `json:"url"`
	Types      []int  `json:"types"`
}

// CameraTaskResponse represents the task list response from device
type CameraTaskResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		Total     int `json:"total"`
		PageSize  int `json:"pageSize"`
		PageCount int `json:"pageCount"`
		PageNo    int `json:"pageNo"`
		Items     []struct {
			TaskID      string   `json:"taskId"`
			DeviceName  string   `json:"deviceName"`
			URL         string   `json:"url"`
			Status      int      `json:"status"`
			ErrorReason string   `json:"errorReason"`
			Abilities   []string `json:"abilities"`
			Types       []int    `json:"types"`
			Width       int      `json:"width"`
			Height      int      `json:"height"`
			CodeName    string   `json:"codeName"`
		} `json:"items"`
	} `json:"result"`
}

// CameraConfigResult represents the result of camera configuration
type CameraConfigResult struct {
	DeviceIP   string `json:"deviceIp"`
	CameraName string `json:"cameraName"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// ExcelSheetData represents the data of a sheet in the Excel file
type ExcelSheetData struct {
	SheetName string     `json:"sheetName"`
	Rows      []ExcelRow `json:"rows"`
}

// ExcelRow represents a row in the Excel file
type ExcelRow struct {
	DeviceIP    string `json:"deviceIp"`
	CameraName  string `json:"cameraName"`
	CameraInfo  string `json:"cameraInfo"`
	DeviceIndex int    `json:"deviceIndex"`
	Selected    bool   `json:"selected"`
}

// DeviceInfo represents device information in camera configuration
type DeviceInfo struct {
	CodeName   string `json:"codeName"`
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	URL        string `json:"url"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// CameraConfig represents the camera configuration
type CameraConfig struct {
	Device     DeviceInfo  `json:"device"`
	Algorithms []Algorithm `json:"algorithms"`
}

// Algorithm represents a camera algorithm configuration
type Algorithm struct {
	Type           int          `json:"Type"`
	TrackInterval  int          `json:"TrackInterval"`
	DetectInterval int          `json:"DetectInterval"`
	AlarmInterval  int          `json:"AlarmInterval"`
	Threshold      int          `json:"threshold"`
	TargetSize     TargetSize   `json:"TargetSize"`
	DetectInfos    []DetectInfo `json:"DetectInfos"`
	TripWire       interface{}  `json:"TripWire"`
	ExtraConfig    ExtraConfig  `json:"ExtraConfig"`
}

// TargetSize represents the target size configuration
type TargetSize struct {
	MinDetect int `json:"MinDetect"`
	MaxDetect int `json:"MaxDetect"`
}

// DetectInfo represents detection information
type DetectInfo struct {
	Id      int            `json:"Id"`
	HotArea []HotAreaPoint `json:"HotArea"`
}

// HotAreaPoint represents a point in the hot area
type HotAreaPoint struct {
	X int `json:"X"`
	Y int `json:"Y"`
}

// ExtraConfig represents extra configuration for algorithms
type ExtraConfig struct {
	CameraIndex string     `json:"camera_index"`
	Defs        []ExtraDef `json:"defs"`
}

// ExtraDef represents a definition in extra configuration
type ExtraDef struct {
	Name    string `json:"Name"`
	Desc    string `json:"Desc"`
	Type    string `json:"Type"`
	Unit    string `json:"Unit"`
	Default string `json:"Default"`
}

// CameraConfigResponse represents the response for camera configuration
type CameraConfigResponse struct {
	Code   int          `json:"code"`
	Msg    string       `json:"msg"`
	Result CameraConfig `json:"result"`
}

// App struct
type App struct {
	ctx         context.Context
	devices     []Device
	mutex       sync.RWMutex       // 使用读写锁而不是互斥锁，优化并发性能
	client      *http.Client       // 重用HTTP客户端
	deviceCache map[string]*Device // 设备缓存，提高查找性能
	configDir   string             // 配置目录预计算
}

// 创建一个优化的http传输层
func createOptimizedTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,  // 拨号超时
			KeepAlive: 30 * time.Second, // 连接保持时间
			DualStack: true,             // 支持IPv4和IPv6
		}).DialContext,
		MaxIdleConns:          100,              // 最大空闲连接数
		IdleConnTimeout:       90 * time.Second, // 空闲连接超时
		TLSHandshakeTimeout:   5 * time.Second,  // TLS握手超时
		ExpectContinueTimeout: 1 * time.Second,  // 100-continue超时
		MaxIdleConnsPerHost:   10,               // 每个主机的最大空闲连接数
		DisableKeepAlives:     false,            // 启用连接复用
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 创建一个优化的HTTP客户端用于整个应用程序生命周期
	client := &http.Client{
		Transport: createOptimizedTransport(),
		Timeout:   10 * time.Second,
	}

	// 获取可执行文件所在的目录作为配置目录
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("警告: 无法获取程序可执行文件路径: %v\n", err)
		// 如果获取失败，回退到当前工作目录
		execPath, err = os.Getwd()
		if err != nil {
			fmt.Printf("警告: 无法获取工作目录: %v\n", err)
			execPath = "."
		}
	}

	configDir := filepath.Join(filepath.Dir(execPath), "config")
	fmt.Printf("配置目录: %s\n", configDir)

	return &App{
		devices:     make([]Device, 0, 50),    // 预分配容量避免频繁扩容
		client:      client,                   // 使用优化的客户端
		deviceCache: make(map[string]*Device), // 初始化设备缓存
		configDir:   configDir,                // 预先计算配置目录
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确保配置目录存在
	if err := os.MkdirAll(a.configDir, 0755); err != nil {
		fmt.Printf("错误: 无法创建配置目录 %s: %v\n", a.configDir, err)

		// 尝试在当前目录创建一个备用配置目录
		currentDir, err := os.Getwd()
		if err == nil {
			backupConfigDir := filepath.Join(currentDir, "config")
			fmt.Printf("尝试在当前目录创建备用配置目录: %s\n", backupConfigDir)

			if err := os.MkdirAll(backupConfigDir, 0755); err == nil {
				a.configDir = backupConfigDir
				fmt.Printf("成功创建备用配置目录: %s\n", a.configDir)
			} else {
				fmt.Printf("无法创建备用配置目录: %v\n", err)
			}
		}
	} else {
		fmt.Printf("配置目录已就绪: %s\n", a.configDir)
	}

	// 清空uploads临时文件夹
	a.cleanUploadsDirectory()

	a.LoadDevices() // Load devices from file on startup
}

// cleanUploadsDirectory 清空uploads临时文件夹的内容，但保留文件夹本身
func (a *App) cleanUploadsDirectory() {
	// 获取程序执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("警告: 无法获取程序可执行文件路径: %v\n", err)
		// 如果获取失败，回退到当前工作目录
		execPath, err = os.Getwd()
		if err != nil {
			fmt.Printf("警告: 无法获取工作目录: %v\n", err)
			return
		}
	}

	// 使用程序执行目录确定uploads目录路径
	execDir := filepath.Dir(execPath)
	uploadsDir := filepath.Join(execDir, "uploads")

	// 检查目录是否存在，如果不存在则不需要清空
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return
	}

	// 打开目录
	dir, err := os.Open(uploadsDir)
	if err != nil {
		fmt.Printf("警告: 无法打开uploads目录进行清理: %v\n", err)
		return
	}
	defer dir.Close()

	// 读取目录中的所有条目
	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Printf("警告: 无法读取uploads目录内容: %v\n", err)
		return
	}

	// 删除每个文件/子目录
	for _, entry := range entries {
		entryPath := filepath.Join(uploadsDir, entry.Name())
		err := os.RemoveAll(entryPath)
		if err != nil {
			fmt.Printf("警告: 无法删除uploads目录中的项目 %s: %v\n", entryPath, err)
		} else {
			fmt.Printf("已删除临时文件: %s\n", entryPath)
		}
	}

	fmt.Printf("已清空uploads临时目录: %s\n", uploadsDir)

	// 同样检查当前工作目录中的uploads目录
	currentDir, err := os.Getwd()
	if err == nil && currentDir != execDir {
		currentUploadsDir := filepath.Join(currentDir, "uploads")
		if _, err := os.Stat(currentUploadsDir); err == nil {
			// 目录存在，清空它
			if err := a.cleanDirectory(currentUploadsDir); err != nil {
				fmt.Printf("警告: 无法清空当前目录中的uploads目录: %v\n", err)
			} else {
				fmt.Printf("已清空当前目录中的uploads临时目录: %s\n", currentUploadsDir)
			}
		}
	}
}

// cleanDirectory 清空指定目录的内容，但保留目录本身
func (a *App) cleanDirectory(dirPath string) error {
	// 打开目录
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	// 读取目录中的所有条目
	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	// 删除每个文件/子目录
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		err := os.RemoveAll(entryPath)
		if err != nil {
			fmt.Printf("警告: 无法删除目录中的项目 %s: %v\n", entryPath, err)
		} else {
			fmt.Printf("已删除临时文件: %s\n", entryPath)
		}
	}

	return nil
}

// GetDevices returns the list of devices
func (a *App) GetDevices() []Device {
	a.mutex.RLock() // 使用读锁而不是写锁，提高并发性能
	defer a.mutex.RUnlock()

	// 返回设备的副本，避免外部修改内部状态
	result := make([]Device, len(a.devices))
	copy(result, a.devices)
	return result
}

// SaveDevices saves the devices to a JSON file
func (a *App) SaveDevices() error {
	a.mutex.RLock() // 使用读锁获取数据
	data, err := json.MarshalIndent(a.devices, "", "  ")
	a.mutex.RUnlock()

	if err != nil {
		return fmt.Errorf("序列化设备数据失败: %w", err)
	}

	devicesPath := filepath.Join(a.configDir, "devices.json")
	fmt.Printf("保存设备列表到: %s\n", devicesPath)

	// 确保配置目录存在
	if err := os.MkdirAll(a.configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 使用临时文件并重命名的方式，确保写入操作的原子性
	tempFile := devicesPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败 %s: %w", tempFile, err)
	}

	// 在Windows上，如果目标文件已存在，重命名可能会失败
	// 先尝试删除目标文件
	_ = os.Remove(devicesPath)

	if err := os.Rename(tempFile, devicesPath); err != nil {
		// 如果重命名失败，直接复制文件内容
		if tempData, readErr := os.ReadFile(tempFile); readErr == nil {
			if writeErr := os.WriteFile(devicesPath, tempData, 0644); writeErr == nil {
				// 写入成功，删除临时文件
				_ = os.Remove(tempFile)
				return nil
			} else {
				return fmt.Errorf("写入目标文件失败: %w", writeErr)
			}
		}
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	fmt.Printf("成功保存了 %d 个设备\n", len(a.devices))
	return nil
}

// LoadDevices loads the devices from a JSON file
func (a *App) LoadDevices() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	devicesPath := filepath.Join(a.configDir, "devices.json")
	fmt.Printf("尝试从 %s 加载设备列表\n", devicesPath)

	if _, err := os.Stat(devicesPath); os.IsNotExist(err) {
		// 文件不存在，初始化为空设备列表
		fmt.Printf("设备列表文件不存在，将使用空列表\n")
		a.devices = make([]Device, 0, 50)        // 预分配容量
		a.deviceCache = make(map[string]*Device) // 清空缓存
		return nil
	}

	data, err := os.ReadFile(devicesPath)
	if err != nil {
		fmt.Printf("读取设备列表文件失败: %v\n", err)
		// 如果读取失败，也使用空列表
		a.devices = make([]Device, 0, 50)
		a.deviceCache = make(map[string]*Device)
		return fmt.Errorf("读取设备列表文件失败: %w", err)
	}

	// 如果文件是空的，使用空列表
	if len(data) == 0 {
		fmt.Printf("设备列表文件为空\n")
		a.devices = make([]Device, 0, 50)
		a.deviceCache = make(map[string]*Device)
		return nil
	}

	// 清空当前数据并重新加载
	a.devices = make([]Device, 0, 50)
	if err := json.Unmarshal(data, &a.devices); err != nil {
		fmt.Printf("解析设备列表JSON失败: %v\n", err)
		// 解析失败时也使用空列表
		a.devices = make([]Device, 0, 50)
		a.deviceCache = make(map[string]*Device)
		return fmt.Errorf("解析设备列表JSON失败: %w", err)
	}

	// 重建设备缓存
	a.deviceCache = make(map[string]*Device, len(a.devices))
	for i := range a.devices {
		a.deviceCache[a.devices[i].IP] = &a.devices[i]
	}

	fmt.Printf("成功加载了 %d 个设备\n", len(a.devices))
	return nil
}

// TestDevice tests if a device is reachable and returns its build time
func (a *App) TestDevice(ip string) (*Device, error) {
	// 使用函数内的上下文，可以在需要时取消
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://%s:8089/api/buildTime", ip)

	// 创建一个带有上下文的请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 使用流式解码，避免读取整个响应到内存
	var response BuildTimeResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("error response from device: %s", response.Msg)
	}

	return &Device{
		IP:        ip,
		BuildTime: response.Result.BuildTime,
		Status:    "online",
	}, nil
}

// AddDevice adds a device to the list after testing it
func (a *App) AddDevice(ip string) (*Device, error) {
	// 测试设备是否可达，这是一个网络操作，不需要持有锁
	device, err := a.TestDevice(ip)
	if err != nil {
		return nil, fmt.Errorf("设备测试失败: %v", err)
	}

	// 加锁处理设备列表的更新
	a.mutex.Lock()

	// 先检查设备是否已存在
	existingDevice, exists := a.deviceCache[ip]
	if exists {
		// 更新现有设备
		*existingDevice = *device
		a.mutex.Unlock() // 在调用 SaveDevices 前释放锁

		// 保存设备列表
		if err := a.SaveDevices(); err != nil {
			return nil, fmt.Errorf("保存设备列表失败: %v", err)
		}
		return existingDevice, nil
	}

	// 添加新设备
	a.devices = append(a.devices, *device)
	// 更新缓存，指向实际存储位置
	a.deviceCache[ip] = &a.devices[len(a.devices)-1]

	// 保存前释放锁
	a.mutex.Unlock()

	// 在锁外进行文件操作
	if err := a.SaveDevices(); err != nil {
		return nil, fmt.Errorf("保存设备列表失败: %v", err)
	}

	return device, nil
}

// RemoveDevice removes a device from the list
func (a *App) RemoveDevice(ip string) error {
	a.mutex.Lock()

	// 直接通过缓存检查设备是否存在
	if _, ok := a.deviceCache[ip]; !ok {
		a.mutex.Unlock()
		return fmt.Errorf("device with IP %s not found", ip)
	}

	// 使用索引重写策略，避免多次内存移动
	newDevices := make([]Device, 0, len(a.devices)-1)
	for _, device := range a.devices {
		if device.IP != ip {
			newDevices = append(newDevices, device)
		}
	}
	a.devices = newDevices

	// 删除缓存中的设备
	delete(a.deviceCache, ip)

	// 在释放锁之前保存我们要修改的设备列表的副本
	a.mutex.Unlock()

	// 在锁外部调用SaveDevices，避免持有锁时进行IO操作
	return a.SaveDevices()
}

// ScanIPRange scans a range of IPs for devices
func (a *App) ScanIPRange(startIP string, endIP string) ([]Device, error) {
	// 解析IP范围
	parts := strings.Split(startIP, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid IP format: %s", startIP)
	}

	endParts := strings.Split(endIP, ".")
	if len(endParts) != 4 {
		return nil, fmt.Errorf("invalid IP format: %s", endIP)
	}

	// 检查前3个段是否相同（必须在同一子网）
	for i := 0; i < 3; i++ {
		if parts[i] != endParts[i] {
			return nil, fmt.Errorf("start IP and end IP must be in the same subnet")
		}
	}

	// 获取起始和结束的第4段
	startOctet, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid start IP: %s", err)
	}

	endOctet, err := strconv.Atoi(endParts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid end IP: %s", err)
	}

	if startOctet > endOctet {
		return nil, fmt.Errorf("start IP must be less than or equal to end IP")
	}

	// 并行扫描设备
	baseIP := fmt.Sprintf("%s.%s.%s", parts[0], parts[1], parts[2])
	ipCount := endOctet - startOctet + 1

	// 使用带缓冲的通道进行设备收集
	results := make(chan *Device, ipCount)
	var wg sync.WaitGroup

	// 创建工作池，限制并发数量避免打开过多连接
	maxWorkers := 16
	if ipCount < maxWorkers {
		maxWorkers = ipCount
	}

	// 创建IP队列
	ipQueue := make(chan int, ipCount)
	for i := startOctet; i <= endOctet; i++ {
		ipQueue <- i
	}
	close(ipQueue)

	// 启动工作线程
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for octet := range ipQueue {
				ip := fmt.Sprintf("%s.%d", baseIP, octet)
				device, err := a.TestDevice(ip)
				if err == nil && device != nil {
					results <- device
				}
			}
		}()
	}

	// 等待所有扫描完成并关闭结果通道
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集设备结果
	foundDevices := make([]Device, 0, ipCount)
	deviceMap := make(map[string]bool) // 用于快速检查设备是否已添加

	// 添加结果到设备列表
	a.mutex.Lock()
	for device := range results {
		foundDevices = append(foundDevices, *device)
		deviceMap[device.IP] = true

		// 检查设备是否存在于当前列表中
		if _, exists := a.deviceCache[device.IP]; !exists {
			a.devices = append(a.devices, *device)
			a.deviceCache[device.IP] = &a.devices[len(a.devices)-1]
		} else {
			// 更新现有设备
			*a.deviceCache[device.IP] = *device
		}
	}
	a.mutex.Unlock()

	// 保存设备列表
	if err := a.SaveDevices(); err != nil {
		return foundDevices, err
	}

	return foundDevices, nil
}

// RefreshDevices refreshes the status of all devices
func (a *App) RefreshDevices() []Device {
	// 获取设备的只读副本
	a.mutex.RLock()
	devices := make([]Device, len(a.devices))
	ips := make([]string, len(a.devices))
	for i, device := range a.devices {
		devices[i] = device
		ips[i] = device.IP
	}
	a.mutex.RUnlock()

	// 如果没有设备，直接返回
	if len(devices) == 0 {
		return devices
	}

	// 创建一个并发限制的工作池
	maxConcurrent := 16
	if len(devices) < maxConcurrent {
		maxConcurrent = len(devices)
	}

	// 使用通道控制并发
	ipChan := make(chan string, len(ips))
	resultChan := make(chan Device, len(ips))

	// 填充IP通道
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	// 启动工作线程
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				updatedDevice, err := a.TestDevice(ip)
				var result Device
				if err != nil {
					// 设备离线
					result = Device{
						IP:     ip,
						Status: "offline",
					}
					// 查找原始构建时间
					a.mutex.RLock()
					if cachedDevice, ok := a.deviceCache[ip]; ok {
						result.BuildTime = cachedDevice.BuildTime
					}
					a.mutex.RUnlock()
				} else {
					result = *updatedDevice
				}
				resultChan <- result
			}
		}()
	}

	// 所有工作完成后关闭结果通道
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	updatedDevices := make([]Device, 0, len(devices))
	deviceMap := make(map[string]Device)

	for device := range resultChan {
		updatedDevices = append(updatedDevices, device)
		deviceMap[device.IP] = device
	}

	// 更新设备列表
	a.mutex.Lock()
	for i := range a.devices {
		if updated, ok := deviceMap[a.devices[i].IP]; ok {
			a.devices[i] = updated
			a.deviceCache[updated.IP] = &a.devices[i]
		}
	}
	a.mutex.Unlock()

	a.SaveDevices()
	return updatedDevices
}

// LoginToDevice logs into a device and returns a token
func (a *App) LoginToDevice(ip, username, password string) (string, error) {
	fmt.Printf("DEBUG: 开始登录设备: IP=%s, 用户名=%s\n", ip, username)

	url := fmt.Sprintf("http://%s:8089/api/login", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	data := map[string]string{
		"username": username,
		"password": password,
	}

	// 预分配缓冲区
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return "", err
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(jsonData))

	// 创建一个带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	fmt.Printf("DEBUG: 请求头: Content-Type=%s\n", req.Header.Get("Content-Type"))

	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return "", err
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 解析响应
	var response LoginResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return "", err
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 登录失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return "", fmt.Errorf("login failed: %s", response.Msg)
	}

	fmt.Printf("DEBUG: 登录成功，获取到Token: %s\n", response.Result.Token)
	return response.Result.Token, nil
}

// UploadFilePath stores the path to the file to be uploaded
var UploadFilePath string

// SetUploadFile sets the path of the file to upload
// 只接收文件名，但会在程序执行目录中创建一个专用目录来存放上传文件
func (a *App) SetUploadFile(filename string) string {
	// 获取程序执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("警告: 无法获取程序可执行文件路径: %v\n", err)
		// 如果获取失败，回退到当前工作目录
		execPath, err = os.Getwd()
		if err != nil {
			fmt.Printf("警告: 无法获取工作目录: %v\n", err)
			execPath = "."
		}
	}

	// 直接使用程序执行目录创建uploads目录
	execDir := filepath.Dir(execPath)
	uploadsDir := filepath.Join(execDir, "uploads")
	fmt.Printf("创建上传文件目录: %s (程序执行目录: %s)\n", uploadsDir, execDir)

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		fmt.Printf("无法在程序目录创建上传目录: %v\n", err)

		// 尝试在当前目录创建uploads目录
		currentDir, _ := os.Getwd()
		if currentDir != execDir {
			uploadsDir = filepath.Join(currentDir, "uploads")
			fmt.Printf("尝试在当前目录创建上传目录: %s\n", uploadsDir)

			if err := os.MkdirAll(uploadsDir, 0755); err != nil {
				fmt.Printf("无法在当前目录创建上传目录: %v\n", err)
				return ""
			}
		} else {
			return ""
		}
	}

	// 预期完整路径位置
	expectedPath := filepath.Join(uploadsDir, filename)
	UploadFilePath = expectedPath
	fmt.Printf("设置上传文件路径: %s\n", expectedPath)

	// 检查预期路径是否已经存在有效文件
	if _, err := os.Stat(expectedPath); err == nil {
		// 文件已存在于预期位置
		fileInfo, err := os.Stat(expectedPath)
		if err == nil && fileInfo.Size() > 0 {
			fmt.Printf("文件已存在并有效: %s (大小: %d 字节)\n", expectedPath, fileInfo.Size())
			return expectedPath
		} else {
			fmt.Printf("文件存在但可能无效: %s\n", expectedPath)
			// 继续寻找有效文件
		}
	}

	// 尝试在多个位置寻找文件
	possibleLocations := []string{
		filename,                         // 相对路径
		filepath.Join(".", filename),     // 当前目录
		filepath.Join(execDir, filename), // 可执行文件所在目录
	}

	// 添加绝对路径（如果提供的是绝对路径）
	if filepath.IsAbs(filename) {
		possibleLocations = append(possibleLocations, filename)
	}

	// 添加用户目录
	if homeDir, err := os.UserHomeDir(); err == nil {
		possibleLocations = append(possibleLocations, filepath.Join(homeDir, filename))
		possibleLocations = append(possibleLocations, filepath.Join(homeDir, "Downloads", filename))
	}

	// 添加工作目录
	if workDir, err := os.Getwd(); err == nil && workDir != execDir {
		possibleLocations = append(possibleLocations, filepath.Join(workDir, filename))
	}

	fmt.Printf("在以下位置寻找文件 '%s':\n", filename)
	for _, loc := range possibleLocations {
		fmt.Printf("- 检查: %s\n", loc)
		if fileInfo, err := os.Stat(loc); err == nil && fileInfo.Size() > 0 {
			// 发现文件，复制到应用目录
			fmt.Printf("  找到文件 (大小: %d 字节)\n", fileInfo.Size())
			if err := copyFile(loc, expectedPath); err == nil {
				fmt.Printf("已复制文件: %s -> %s\n", loc, expectedPath)
				return expectedPath
			} else {
				fmt.Printf("复制文件失败: %v\n", err)
			}
		}
	}

	// 如果找不到文件，返回预期路径并记录警告
	fmt.Printf("警告: 未能找到文件: %s\n", filename)
	fmt.Printf("预期路径: %s\n", expectedPath)
	fmt.Printf("请将文件 '%s' 复制到以下位置之一:\n", filename)
	for _, loc := range possibleLocations {
		fmt.Printf("- %s\n", loc)
	}

	return expectedPath
}

// copyFile 将源文件复制到目标路径
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// GetUploadFile gets the path of the file to upload
func (a *App) GetUploadFile() string {
	return UploadFilePath
}

// UpdateDevices updates all devices with the uploaded file
func (a *App) UpdateDevices(username, password string, selectedBuildTime string) []UpdateResult {
	if UploadFilePath == "" {
		return []UpdateResult{{
			IP:      "",
			Success: false,
			Message: "No file selected for upload",
		}}
	}

	// 获取设备列表的只读副本
	a.mutex.RLock()
	allDevices := make([]Device, len(a.devices))
	copy(allDevices, a.devices)
	a.mutex.RUnlock()

	// 如果没有设备，直接返回
	if len(allDevices) == 0 {
		return []UpdateResult{{
			IP:      "",
			Success: false,
			Message: "No devices to update",
		}}
	}

	// 筛选具有指定buildTime的设备
	var devices []Device
	if selectedBuildTime == "" {
		devices = allDevices
	} else {
		devices = make([]Device, 0)
		for _, device := range allDevices {
			if device.BuildTime == selectedBuildTime {
				devices = append(devices, device)
			}
		}

		if len(devices) == 0 {
			return []UpdateResult{{
				IP:      "",
				Success: false,
				Message: "No devices with selected build time to update",
			}}
		}
	}

	// 使用结果通道收集结果
	resultChan := make(chan UpdateResult, len(devices))

	// 创建工作池控制并发
	maxConcurrent := 8 // 限制并发更新数量，避免网络和系统负载过高
	if len(devices) < maxConcurrent {
		maxConcurrent = len(devices)
	}

	// 使用通道控制并发
	deviceChan := make(chan Device, len(devices))
	for _, device := range devices {
		deviceChan <- device
	}
	close(deviceChan)

	// 启动工作线程
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for device := range deviceChan {
				// 尝试登录并更新设备
				token, err := a.LoginToDevice(device.IP, username, password)
				if err != nil {
					resultChan <- UpdateResult{
						IP:      device.IP,
						Success: false,
						Message: fmt.Sprintf("Login failed: %v", err),
					}
					continue
				}

				// 上传文件
				success, message := a.uploadBinary(device.IP, token)
				resultChan <- UpdateResult{
					IP:      device.IP,
					Success: success,
					Message: message,
				}
			}
		}()
	}

	// 等待所有更新完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]UpdateResult, 0, len(devices))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// uploadBinary uploads a binary file to a device
func (a *App) uploadBinary(ip, token string) (bool, string) {
	// 获取上传文件路径
	filePath := a.GetUploadFile()
	fmt.Printf("DEBUG: 开始上传文件: %s 到设备 %s\n", filePath, ip)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("ERROR: 文件不存在或无法访问: %s, 错误: %v\n", filePath, err)
		return false, fmt.Sprintf("File not found or inaccessible: %v", err)
	}

	fmt.Printf("DEBUG: 文件大小: %d 字节\n", fileInfo.Size())

	// 设置超时（基于文件大小，每MB至少10秒，最少30秒）
	fileSize := fileInfo.Size()
	timeoutSeconds := int(math.Max(30, float64(fileSize)/(1024*1024)*10))
	fmt.Printf("DEBUG: 设置上传超时为 %d 秒\n", timeoutSeconds)

	// 创建上传URL
	url := fmt.Sprintf("http://%s:8089/api/system/upgrade", ip)
	fmt.Printf("DEBUG: 上传URL: %s\n", url)

	// 创建一个缓冲区来存储multipart数据
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: 无法打开文件: %s, 错误: %v\n", filePath, err)
		return false, fmt.Sprintf("Cannot open file: %v", err)
	}
	defer file.Close()

	fmt.Printf("DEBUG: 创建multipart表单字段 'binary'\n")
	part, err := writer.CreateFormFile("binary", filepath.Base(filePath))
	if err != nil {
		fmt.Printf("ERROR: 创建表单文件字段失败: %v\n", err)
		return false, fmt.Sprintf("Failed to create form file: %v", err)
	}

	// 复制文件内容到表单字段
	bytesWritten, err := io.Copy(part, file)
	if err != nil {
		fmt.Printf("ERROR: 复制文件内容失败: %v\n", err)
		return false, fmt.Sprintf("Failed to copy file content: %v", err)
	}
	fmt.Printf("DEBUG: 已写入 %d 字节到表单\n", bytesWritten)

	// 完成multipart表单
	err = writer.Close()
	if err != nil {
		fmt.Printf("ERROR: 关闭multipart writer失败: %v\n", err)
		return false, fmt.Sprintf("Failed to close multipart writer: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// 设置Content-Type头，包含动态生成的boundary
	contentType := writer.FormDataContentType()
	fmt.Printf("DEBUG: Content-Type: %s\n", contentType)
	req.Header.Set("Content-Type", contentType)

	// 使用Token头部
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: Token: %s\n", token)

	// 打印完整请求头
	fmt.Printf("DEBUG: 请求头:\n")
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 设置一个专用于上传的客户端，具有更长的超时时间
	uploadClient := &http.Client{
		Transport: a.client.Transport,
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
	}

	fmt.Printf("DEBUG: 发送HTTP请求...\n")
	resp, err := uploadClient.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("Upload failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 收到响应状态码: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("DEBUG: 响应头:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应以获取更详细的信息
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		fmt.Printf("ERROR: 上传失败. 响应体: %s\n", string(errBody))
		return false, fmt.Sprintf("Upload failed with status: %s, details: %s", resp.Status, string(errBody))
	}

	// 读取成功响应
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	fmt.Printf("DEBUG: 上传成功. 响应体: %s\n", string(respBody))

	return true, "Update successful"
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SaveExcelData saves the Excel data to a temporary file
func (a *App) SaveExcelData(fileData string) (string, error) {
	// 获取程序执行文件路径，用于创建临时目录
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}

	execDir := filepath.Dir(execPath)
	tempDir := filepath.Join(execDir, "temp")

	// 确保临时目录存在
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建临时目录: %w", err)
	}

	// 创建临时文件
	filename := fmt.Sprintf("excel_data_%d.xlsx", time.Now().UnixNano())
	filePath := filepath.Join(tempDir, filename)

	// 解码Base64数据
	data, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		return "", fmt.Errorf("无法解码文件数据: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("无法写入文件: %w", err)
	}

	return filePath, nil
}

// GetCameraTasks gets the camera tasks from a device
func (a *App) GetCameraTasks(ip, username, password string) ([]Camera, error) {
	fmt.Printf("DEBUG: 开始获取摄像头任务列表: IP=%s\n", ip)

	// 1. 先登录设备获取token
	token, err := a.LoginToDevice(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 登录设备失败: %v\n", err)
		return nil, fmt.Errorf("登录设备失败: %w", err)
	}
	fmt.Printf("DEBUG: 成功登录设备，获取到Token\n")

	// 2. 获取摄像头任务列表
	url := fmt.Sprintf("http://%s:8089/api/task/list", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]interface{}{
		"pageNo":   1,
		"pageSize": 100,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 打印响应体，但限制长度以避免日志过大
	respBodyStr := string(respBody)
	if len(respBodyStr) > 1000 {
		fmt.Printf("DEBUG: 响应体(部分): %s...(已截断，总长度 %d 字节)\n", respBodyStr[:1000], len(respBodyStr))
	} else {
		fmt.Printf("DEBUG: 响应体: %s\n", respBodyStr)
	}

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 解析响应
	var response CameraTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 获取任务列表失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return nil, fmt.Errorf("获取任务列表失败: %s", response.Msg)
	}

	// 转换为Camera结构体
	cameras := make([]Camera, 0, len(response.Result.Items))
	for _, item := range response.Result.Items {
		cameras = append(cameras, Camera{
			TaskID:     item.TaskID,
			DeviceName: item.DeviceName,
			URL:        item.URL,
			Types:      item.Types,
		})
	}

	fmt.Printf("DEBUG: 成功获取到 %d 个摄像头任务\n", len(cameras))
	// 打印每个任务的ID
	for i, camera := range cameras {
		fmt.Printf("DEBUG: 任务 %d: ID=%s, 设备名=%s\n", i+1, camera.TaskID, camera.DeviceName)
	}

	return cameras, nil
}

// ConfigureCamera configures a camera on a device
func (a *App) ConfigureCamera(ip, username, password, cameraName, cameraURL string, algorithmType int) (bool, string) {
	fmt.Printf("DEBUG: 开始配置摄像头: IP=%s, 摄像头名称=%s, URL=%s, 算法类型=%d\n", ip, cameraName, cameraURL, algorithmType)

	// 1. 先登录设备获取token
	token, err := a.LoginToDevice(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 登录设备失败: %v\n", err)
		return false, fmt.Sprintf("登录设备失败: %v", err)
	}
	fmt.Printf("DEBUG: 成功登录设备，获取到Token\n")

	// 2. 获取当前摄像头配置
	cameras, err := a.GetCameraTasks(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 获取摄像头任务列表失败: %v\n", err)
		return false, fmt.Sprintf("获取摄像头任务列表失败: %v", err)
	}
	fmt.Printf("DEBUG: 获取到 %d 个摄像头任务\n", len(cameras))

	// 检查摄像头是否已存在
	existingCamera := false
	for _, camera := range cameras {
		if camera.TaskID == cameraName {
			existingCamera = true
			fmt.Printf("DEBUG: 找到已存在的摄像头任务: %s\n", cameraName)
			break
		}
	}

	// 3. 根据摄像头是否存在，选择添加或修改
	var url string
	if existingCamera {
		url = fmt.Sprintf("http://%s:8089/api/task/modify", ip)
		fmt.Printf("DEBUG: 摄像头任务已存在，使用修改API\n")
	} else {
		url = fmt.Sprintf("http://%s:8089/api/task/add", ip)
		fmt.Printf("DEBUG: 摄像头任务不存在，使用添加API\n")
	}
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]interface{}{
		"taskId":     cameraName,
		"deviceName": cameraName,
		"url":        cameraURL,
		"types":      []int{algorithmType},
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("创建HTTP请求失败: %v", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return false, fmt.Sprintf("读取响应体失败: %v", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 读取响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return false, fmt.Sprintf("解析响应失败: %v", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 配置摄像头失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return false, fmt.Sprintf("配置摄像头失败: %s", response.Msg)
	}

	if existingCamera {
		fmt.Printf("DEBUG: 成功修改摄像头配置: %s\n", cameraName)
		return true, "修改摄像头配置成功"
	} else {
		fmt.Printf("DEBUG: 成功添加摄像头配置: %s\n", cameraName)
		return true, "添加摄像头配置成功"
	}
}

// GetCameraConfig gets the configuration of a camera
func (a *App) GetCameraConfig(ip, username, password, taskId string) (*CameraConfig, error) {
	fmt.Printf("DEBUG: 开始获取摄像头配置: IP=%s, 任务ID=%s\n", ip, taskId)

	// 1. 先登录设备获取token
	token, err := a.LoginToDevice(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 登录设备失败: %v\n", err)
		return nil, fmt.Errorf("登录设备失败: %w", err)
	}
	fmt.Printf("DEBUG: 成功登录设备，获取到Token\n")

	// 2. 获取摄像头配置
	url := fmt.Sprintf("http://%s:8089/api/config/get", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]string{
		"taskId": taskId,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 解析响应
	var response CameraConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 获取摄像头配置失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return nil, fmt.Errorf("获取摄像头配置失败: %s", response.Msg)
	}

	fmt.Printf("DEBUG: 成功获取摄像头配置\n")
	return &response.Result, nil
}

// SetCameraIndex sets the index of a camera
func (a *App) SetCameraIndex(ip, username, password, taskId string, index int) (bool, string) {
	fmt.Printf("DEBUG: 开始设置摄像头索引: IP=%s, 任务ID=%s, 索引=%d\n", ip, taskId, index)

	// 1. 获取摄像头配置
	config, err := a.GetCameraConfig(ip, username, password, taskId)
	if err != nil {
		fmt.Printf("ERROR: 获取摄像头配置失败: %v\n", err)
		return false, fmt.Sprintf("获取摄像头配置失败: %v", err)
	}
	fmt.Printf("DEBUG: 成功获取摄像头配置\n")

	// 2. 修改摄像头索引
	var modified bool
	for i := range config.Algorithms {
		// 修改ExtraConfig中的camera_index
		if config.Algorithms[i].ExtraConfig.CameraIndex != fmt.Sprintf("%d", index) {
			fmt.Printf("DEBUG: 更新算法 %d 的摄像头索引: %s -> %d\n", i, config.Algorithms[i].ExtraConfig.CameraIndex, index)
			config.Algorithms[i].ExtraConfig.CameraIndex = fmt.Sprintf("%d", index)
			modified = true
		}
	}

	if !modified {
		fmt.Printf("DEBUG: 摄像头索引已经是正确的值 %d，无需修改\n", index)
		return true, "摄像头索引已经是正确的值，无需修改"
	}

	// 3. 更新摄像头配置
	token, err := a.LoginToDevice(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 登录设备失败: %v\n", err)
		return false, fmt.Sprintf("登录设备失败: %v", err)
	}
	fmt.Printf("DEBUG: 成功登录设备，获取到Token\n")

	url := fmt.Sprintf("http://%s:8089/api/config/mod", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 按照FEATURE.md中的示例格式构造请求载荷
	requestData := map[string]interface{}{
		"TaskID":    taskId,
		"Algorithm": config.Algorithms[0], // 使用第一个算法（通常只有一个）
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	// 打印请求体，但限制长度以避免日志过大
	requestBodyStr := string(requestBody)
	if len(requestBodyStr) > 1000 {
		fmt.Printf("DEBUG: 请求体(部分): %s...(已截断，总长度 %d 字节)\n", requestBodyStr[:1000], len(requestBodyStr))
	} else {
		fmt.Printf("DEBUG: 请求体: %s\n", requestBodyStr)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("创建HTTP请求失败: %v", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return false, fmt.Sprintf("读取响应体失败: %v", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 读取响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return false, fmt.Sprintf("解析响应失败: %v", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 设置摄像头索引失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return false, fmt.Sprintf("设置摄像头索引失败: %s", response.Msg)
	}

	fmt.Printf("DEBUG: 成功将摄像头索引设置为 %d\n", index)
	return true, fmt.Sprintf("成功将摄像头索引设置为 %d", index)
}

// ConfigureCameraWithToken configures a camera on a device using an existing token
func (a *App) ConfigureCameraWithToken(ip, token, cameraName, cameraURL string, algorithmType int) (bool, string) {
	fmt.Printf("DEBUG: 开始配置摄像头: IP=%s, 摄像头名称=%s, URL=%s, 算法类型=%d\n", ip, cameraName, cameraURL, algorithmType)

	// 获取当前摄像头配置，使用已有的 token
	cameras, err := a.GetCameraTasksWithToken(ip, token)
	if err != nil {
		fmt.Printf("ERROR: 获取摄像头任务列表失败: %v\n", err)
		return false, fmt.Sprintf("获取摄像头任务列表失败: %v", err)
	}
	fmt.Printf("DEBUG: 获取到 %d 个摄像头任务\n", len(cameras))

	// 检查摄像头是否已存在
	existingCamera := false
	for _, camera := range cameras {
		if camera.TaskID == cameraName {
			existingCamera = true
			fmt.Printf("DEBUG: 找到已存在的摄像头任务: %s\n", cameraName)
			break
		}
	}

	// 根据摄像头是否存在，选择添加或修改
	var url string
	if existingCamera {
		url = fmt.Sprintf("http://%s:8089/api/task/modify", ip)
		fmt.Printf("DEBUG: 摄像头任务已存在，使用修改API\n")
	} else {
		url = fmt.Sprintf("http://%s:8089/api/task/add", ip)
		fmt.Printf("DEBUG: 摄像头任务不存在，使用添加API\n")
	}
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]interface{}{
		"taskId":     cameraName,
		"deviceName": cameraName,
		"url":        cameraURL,
		"types":      []int{algorithmType},
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("创建HTTP请求失败: %v", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return false, fmt.Sprintf("读取响应体失败: %v", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 读取响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return false, fmt.Sprintf("解析响应失败: %v", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 配置摄像头失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return false, fmt.Sprintf("配置摄像头失败: %s", response.Msg)
	}

	if existingCamera {
		fmt.Printf("DEBUG: 成功修改摄像头配置: %s\n", cameraName)
		return true, "修改摄像头配置成功"
	} else {
		fmt.Printf("DEBUG: 成功添加摄像头配置: %s\n", cameraName)
		return true, "添加摄像头配置成功"
	}
}

// GetCameraTasksWithToken gets the camera tasks from a device using an existing token
func (a *App) GetCameraTasksWithToken(ip, token string) ([]Camera, error) {
	fmt.Printf("DEBUG: 开始获取摄像头任务列表: IP=%s\n", ip)

	// 获取摄像头任务列表
	url := fmt.Sprintf("http://%s:8089/api/task/list", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]interface{}{
		"pageNo":   1,
		"pageSize": 100,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 打印响应体，但限制长度以避免日志过大
	respBodyStr := string(respBody)
	if len(respBodyStr) > 1000 {
		fmt.Printf("DEBUG: 响应体(部分): %s...(已截断，总长度 %d 字节)\n", respBodyStr[:1000], len(respBodyStr))
	} else {
		fmt.Printf("DEBUG: 响应体: %s\n", respBodyStr)
	}

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 解析响应
	var response CameraTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 获取任务列表失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return nil, fmt.Errorf("获取任务列表失败: %s", response.Msg)
	}

	// 转换为Camera结构体
	cameras := make([]Camera, 0, len(response.Result.Items))
	for _, item := range response.Result.Items {
		cameras = append(cameras, Camera{
			TaskID:     item.TaskID,
			DeviceName: item.DeviceName,
			URL:        item.URL,
			Types:      item.Types,
		})
	}

	fmt.Printf("DEBUG: 成功获取到 %d 个摄像头任务\n", len(cameras))
	// 打印每个任务的ID
	for i, camera := range cameras {
		fmt.Printf("DEBUG: 任务 %d: ID=%s, 设备名=%s\n", i+1, camera.TaskID, camera.DeviceName)
	}

	return cameras, nil
}

// GetCameraConfigWithToken gets the configuration of a camera using an existing token
func (a *App) GetCameraConfigWithToken(ip, token, taskId string) (*CameraConfig, error) {
	fmt.Printf("DEBUG: 开始获取摄像头配置: IP=%s, 任务ID=%s\n", ip, taskId)

	// 获取摄像头配置
	url := fmt.Sprintf("http://%s:8089/api/config/get", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 创建请求体
	requestData := map[string]string{
		"taskId": taskId,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	fmt.Printf("DEBUG: 请求体: %s\n", string(requestBody))

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 解析响应
	var response CameraConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 获取摄像头配置失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return nil, fmt.Errorf("获取摄像头配置失败: %s", response.Msg)
	}

	fmt.Printf("DEBUG: 成功获取摄像头配置\n")
	return &response.Result, nil
}

// SetCameraIndexWithToken sets the index of a camera using an existing token
func (a *App) SetCameraIndexWithToken(ip, token, taskId string, index int) (bool, string) {
	fmt.Printf("DEBUG: 开始设置摄像头索引: IP=%s, 任务ID=%s, 索引=%d\n", ip, taskId, index)

	// 1. 获取摄像头配置
	config, err := a.GetCameraConfigWithToken(ip, token, taskId)
	if err != nil {
		fmt.Printf("ERROR: 获取摄像头配置失败: %v\n", err)
		return false, fmt.Sprintf("获取摄像头配置失败: %v", err)
	}
	fmt.Printf("DEBUG: 成功获取摄像头配置\n")

	// 2. 修改摄像头索引
	var modified bool
	for i := range config.Algorithms {
		// 修改ExtraConfig中的camera_index
		if config.Algorithms[i].ExtraConfig.CameraIndex != fmt.Sprintf("%d", index) {
			fmt.Printf("DEBUG: 更新算法 %d 的摄像头索引: %s -> %d\n", i, config.Algorithms[i].ExtraConfig.CameraIndex, index)
			config.Algorithms[i].ExtraConfig.CameraIndex = fmt.Sprintf("%d", index)
			modified = true
		}
	}

	if !modified {
		fmt.Printf("DEBUG: 摄像头索引已经是正确的值 %d，无需修改\n", index)
		return true, "摄像头索引已经是正确的值，无需修改"
	}

	url := fmt.Sprintf("http://%s:8089/api/config/mod", ip)
	fmt.Printf("DEBUG: 请求URL: %s\n", url)

	// 按照FEATURE.md中的示例格式构造请求载荷
	requestData := map[string]interface{}{
		"TaskID":    taskId,
		"Algorithm": config.Algorithms[0], // 使用第一个算法（通常只有一个）
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("ERROR: 创建请求失败: %v\n", err)
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	// 打印请求体，但限制长度以避免日志过大
	requestBodyStr := string(requestBody)
	if len(requestBodyStr) > 1000 {
		fmt.Printf("DEBUG: 请求体(部分): %s...(已截断，总长度 %d 字节)\n", requestBodyStr[:1000], len(requestBodyStr))
	} else {
		fmt.Printf("DEBUG: 请求体: %s\n", requestBodyStr)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("创建HTTP请求失败: %v", err)
	}

	// 设置header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: 请求头: Content-Type=%s, Token=%s\n", req.Header.Get("Content-Type"), req.Header.Get("Token"))

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 响应状态码: %d\n", resp.StatusCode)

	// 读取完整响应体以便记录日志
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: 读取响应体失败: %v\n", err)
		return false, fmt.Sprintf("读取响应体失败: %v", err)
	}

	fmt.Printf("DEBUG: 响应体: %s\n", string(respBody))

	// 将读取的响应体转回io.Reader用于解析
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 读取响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return false, fmt.Sprintf("解析响应失败: %v", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 设置摄像头索引失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return false, fmt.Sprintf("设置摄像头索引失败: %s", response.Msg)
	}

	fmt.Printf("DEBUG: 成功将摄像头索引设置为 %d\n", index)
	return true, fmt.Sprintf("成功将摄像头索引设置为 %d", index)
}

// ConfigureCamerasFromData configures cameras based on the provided data
func (a *App) ConfigureCamerasFromData(deviceConfigs []ExcelRow, username, password, urlTemplate string, algorithmType int) []CameraConfigResult {
	// 创建一个带缓冲的结果通道，用于收集所有设备的结果
	resultChan := make(chan []CameraConfigResult, len(deviceConfigs))

	// 按设备IP分组
	deviceGroups := make(map[string][]ExcelRow)
	for _, config := range deviceConfigs {
		if config.DeviceIP != "" && config.CameraName != "" && config.CameraInfo != "/" {
			deviceGroups[config.DeviceIP] = append(deviceGroups[config.DeviceIP], config)
		}
	}

	// 控制最大并发数量
	maxConcurrent := 8 // 最多同时处理8个设备
	if len(deviceGroups) < maxConcurrent {
		maxConcurrent = len(deviceGroups)
	}

	// 使用通道控制并发数量
	deviceIPChan := make(chan string, len(deviceGroups))
	for deviceIP := range deviceGroups {
		deviceIPChan <- deviceIP
	}
	close(deviceIPChan)

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup

	// 启动工作协程池
	fmt.Printf("DEBUG: 启动 %d 个工作协程处理设备配置\n", maxConcurrent)
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()

			for deviceIP := range deviceIPChan {
				configs := deviceGroups[deviceIP]
				deviceResults := a.configureCamerasForDevice(deviceIP, configs, username, password, urlTemplate, algorithmType, workerId)
				resultChan <- deviceResults
			}
		}(i)
	}

	// 等待所有goroutine完成，然后关闭结果通道
	go func() {
		wg.Wait()
		close(resultChan)
		fmt.Printf("DEBUG: 所有设备处理完成\n")
	}()

	// 收集结果
	var results []CameraConfigResult
	for deviceResults := range resultChan {
		results = append(results, deviceResults...)
	}

	return results
}

// configureCamerasForDevice 处理单个设备的所有摄像头配置
func (a *App) configureCamerasForDevice(deviceIP string, configs []ExcelRow, username, password, urlTemplate string, algorithmType int, workerId int) []CameraConfigResult {
	results := make([]CameraConfigResult, 0, len(configs))

	// 为每个设备只获取一次token
	fmt.Printf("DEBUG: [Worker-%d] 开始为设备 %s 配置摄像头，共 %d 个\n", workerId, deviceIP, len(configs))
	token, err := a.LoginToDevice(deviceIP, username, password)
	if err != nil {
		fmt.Printf("ERROR: [Worker-%d] 登录设备 %s 失败: %v\n", workerId, deviceIP, err)
		// 如果登录失败，将此设备下的所有摄像头标记为失败
		for _, config := range configs {
			results = append(results, CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    false,
				Message:    fmt.Sprintf("登录设备失败: %v", err),
			})
		}
		return results
	}
	fmt.Printf("DEBUG: [Worker-%d] 成功登录设备 %s，获取到Token\n", workerId, deviceIP)

	for _, config := range configs {
		// 使用传入的设备内索引，如果没有则使用默认值1
		cameraIndex := config.DeviceIndex
		if cameraIndex <= 0 {
			cameraIndex = 1
		}

		// 从摄像头信息中提取IP
		parts := strings.Split(config.CameraInfo, "/")
		if len(parts) >= 1 {
			cameraIP := parts[0]

			// 使用模板替换IP
			cameraURL := strings.Replace(urlTemplate, "<ip>", cameraIP, -1)

			// 配置摄像头，使用已获取的token
			fmt.Printf("DEBUG: [Worker-%d] 配置摄像头: %s 在设备 %s\n", workerId, config.CameraName, deviceIP)
			success, message := a.ConfigureCameraWithToken(deviceIP, token, config.CameraName, cameraURL, algorithmType)

			// 如果配置成功，设置摄像头索引
			if success {
				fmt.Printf("DEBUG: [Worker-%d] 摄像头配置成功，等待500毫秒后设置索引...\n", workerId)
				// 等待500毫秒，确保摄像头任务已初始化
				time.Sleep(500 * time.Millisecond)

				fmt.Printf("DEBUG: [Worker-%d] 设置摄像头索引: %s -> %d\n", workerId, config.CameraName, cameraIndex)
				indexSuccess, indexMessage := a.SetCameraIndexWithToken(deviceIP, token, config.CameraName, cameraIndex)
				if !indexSuccess {
					message += ". " + indexMessage
				} else {
					message += ". " + indexMessage
				}
			}

			// 记录结果
			results = append(results, CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    success,
				Message:    message,
			})
		} else {
			// 摄像头信息格式错误
			results = append(results, CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    false,
				Message:    "摄像头信息格式错误",
			})
		}
	}

	fmt.Printf("DEBUG: [Worker-%d] 设备 %s 的所有摄像头配置完成\n", workerId, deviceIP)
	return results
}
