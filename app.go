package main

import (
	"bytes"
	"context"
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
	url := fmt.Sprintf("http://%s:8089/api/login", ip)
	data := map[string]string{
		"username": username,
		"password": password,
	}

	// 预分配缓冲区
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// 创建一个带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 使用流式解码，避免读取整个响应到内存
	var response LoginResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", fmt.Errorf("login failed: %s", response.Msg)
	}

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
