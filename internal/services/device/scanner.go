package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"application-updater/internal/models"
)

// DeviceScanner implements the Scanner interface for device discovery and testing operations.
// It provides methods to scan IP ranges and test individual devices.
type DeviceScanner struct {
	Client *http.Client
}

// NewScanner creates a new Scanner instance that can scan and test devices.
// This returns a DeviceScanner that implements the Scanner interface.
func NewScanner(client *http.Client) Scanner {
	return &DeviceScanner{
		Client: client,
	}
}

// ScanIPRange scans an IP range to find devices.
// This method implements the Scanner interface.
func (s *DeviceScanner) ScanIPRange(ctx context.Context, startIP string, endIP string) []models.Device {
	// 解析起始IP
	ipStart := net.ParseIP(startIP).To4()
	if ipStart == nil {
		fmt.Printf("无效的起始IP地址: %s\n", startIP)
		return nil
	}

	// 解析结束IP
	ipEnd := net.ParseIP(endIP).To4()
	if ipEnd == nil {
		fmt.Printf("无效的结束IP地址: %s\n", endIP)
		return nil
	}

	// 验证IP范围
	if !less(ipStart, ipEnd) {
		fmt.Printf("起始IP必须小于结束IP\n")
		return nil
	}

	// 计算扫描的IP总数
	totalIPs := calculateTotalIPs(ipStart, ipEnd)
	fmt.Printf("开始扫描IP范围: %s - %s, 共 %d 个IP\n", startIP, endIP, totalIPs)

	// 限制最大扫描数量，防止扫描范围过大
	if totalIPs > 1000 {
		fmt.Printf("IP范围过大，最多扫描1000个IP\n")
		return nil
	}

	// 使用通道收集结果
	results := make(chan *models.Device)
	limitCh := make(chan struct{}, 32) // 限制并发数量
	var wg sync.WaitGroup

	// 如果没有提供上下文，创建一个默认的
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
	}

	// 创建设备映射，用于去重
	deviceMap := make(map[string]bool)
	foundDevices := make([]models.Device, 0)

	// 遍历IP范围
	currentIP := make(net.IP, len(ipStart))
	copy(currentIP, ipStart)

	for less(currentIP, ipEnd) || equal(currentIP, ipEnd) {
		select {
		case <-ctx.Done():
			fmt.Printf("扫描被取消或超时\n")
			return foundDevices
		default:
			wg.Add(1)
			limitCh <- struct{}{} // 获取令牌
			ip := make(net.IP, len(currentIP))
			copy(ip, currentIP)

			// 异步扫描
			go func(ip net.IP) {
				defer wg.Done()
				defer func() { <-limitCh }() // 释放令牌

				ipStr := ip.String()
				device, err := s.TestDevice(ipStr)
				if err == nil && device != nil {
					results <- device
				}
			}(ip)

			// 递增IP
			incrementIP(currentIP)
		}
	}

	// 启动收集结果的goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for device := range results {
		if !deviceMap[device.IP] {
			foundDevices = append(foundDevices, *device)
			deviceMap[device.IP] = true
		}
	}

	fmt.Printf("扫描完成，找到 %d 个设备\n", len(foundDevices))
	return foundDevices
}

// RefreshDevices refreshes the status of all provided devices.
// This is an additional method not required by the Scanner interface.
func (s *DeviceScanner) RefreshDevices(devices []models.Device) []models.Device {
	// 如果没有设备，直接返回
	if len(devices) == 0 {
		return devices
	}

	// 创建一个并发限制的工作池
	maxConcurrent := 16
	if len(devices) < maxConcurrent {
		maxConcurrent = len(devices)
	}

	// 创建一个通道用于传递设备，而不只是IP
	deviceChan := make(chan models.Device, len(devices))
	resultChan := make(chan models.Device, len(devices))

	// 填充设备通道
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
			for originalDevice := range deviceChan {
				// 确保设备有ID
				if originalDevice.ID == "" && originalDevice.IP != "" {
					originalDevice.ID = models.GenerateDeviceID(originalDevice.Region, originalDevice.IP)
				}

				// 跳过无效设备
				if originalDevice.IP == "" {
					fmt.Printf("警告: 刷新设备时发现无效设备 (ID: %s)，已跳过\n", originalDevice.ID)
					continue
				}

				// 测试设备
				updatedDevice, err := s.TestDevice(originalDevice.IP)
				var result models.Device

				if err != nil {
					// 设备离线 - 保留原始设备的所有信息，只更新状态
					result = originalDevice
					result.Status = "offline"
				} else {
					// 设备在线 - 使用原始设备的ID和Region，更新BuildTime和Status
					result = *updatedDevice

					// 始终保留原始设备ID
					result.ID = originalDevice.ID

					// 保留区域信息
					result.Region = originalDevice.Region
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

	// 收集结果，使用map确保按设备ID去重
	updatedDeviceMap := make(map[string]models.Device)
	for device := range resultChan {
		// 确保设备有ID
		if device.ID == "" && device.IP != "" {
			device.ID = models.GenerateDeviceID(device.Region, device.IP)
		}

		// 使用ID作为键保存设备，确保每个设备只被处理一次
		if device.ID != "" {
			updatedDeviceMap[device.ID] = device
		} else if device.IP != "" {
			// 如果仍然没有ID但有IP，使用生成的ID
			id := models.GenerateDeviceID(device.Region, device.IP)
			device.ID = id
			updatedDeviceMap[id] = device
		}
	}

	// 转换回切片
	updatedDevices := make([]models.Device, 0, len(updatedDeviceMap))
	for _, device := range updatedDeviceMap {
		updatedDevices = append(updatedDevices, device)
	}

	return updatedDevices
}

// TestDevice tests if a device is reachable and gets its build time.
// This method implements the Scanner interface.
func (s *DeviceScanner) TestDevice(ip string) (*models.Device, error) {
	// 使用函数内的上下文，可以在需要时取消
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://%s:8089/api/buildTime", ip)

	// 创建一个带有上下文的请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 使用流式解码，避免读取整个响应到内存
	var response models.BuildTimeResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("error response from device: %s", response.Msg)
	}

	return &models.Device{
		IP:        ip,
		BuildTime: response.Result.BuildTime,
		Status:    "online",
	}, nil
}

// Helper function: compares two IP addresses
func less(ip1, ip2 net.IP) bool {
	for i := 0; i < len(ip1); i++ {
		if ip1[i] < ip2[i] {
			return true
		} else if ip1[i] > ip2[i] {
			return false
		}
	}
	return false
}

// equal compares two IP addresses for equality
func equal(ip1, ip2 net.IP) bool {
	for i := 0; i < len(ip1); i++ {
		if ip1[i] != ip2[i] {
			return false
		}
	}
	return true
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

// calculateTotalIPs calculates the total number of IPs in a range
func calculateTotalIPs(ipStart, ipEnd net.IP) int {
	total := 0
	for i := 0; i < len(ipStart); i++ {
		total = (total << 8) + int(ipEnd[i]-ipStart[i])
	}
	return total + 1
}
