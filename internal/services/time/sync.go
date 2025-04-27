package time

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"application-updater/internal/models"

	"golang.org/x/crypto/ssh"
)

// Service handles time synchronization operations
type Service struct {
	mutex sync.Mutex
}

// NewService creates a new time sync service
func NewService() *Service {
	return &Service{}
}

// SyncDeviceTime synchronizes the time of the devices with the current machine's time
func (s *Service) SyncDeviceTime(username, password string, deviceIPs []string) []models.TimeSyncResult {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 创建结果通道
	resultChan := make(chan models.TimeSyncResult, len(deviceIPs))

	// 控制最大并发数量
	maxConcurrent := 8
	if len(deviceIPs) < maxConcurrent {
		maxConcurrent = len(deviceIPs)
	}

	// 使用通道控制并发
	deviceChan := make(chan string, len(deviceIPs))
	for _, ip := range deviceIPs {
		deviceChan <- ip
	}
	close(deviceChan)

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup

	// 获取当前系统时间
	currentTime := time.Now()
	dateTimeString := currentTime.Format("20060102150405") // YYYYMMDDHHmmss

	fmt.Printf("DEBUG: 开始同步设备时间，当前系统时间: %s\n", currentTime.Format("2006-01-02 15:04:05"))

	// 启动工作协程池
	fmt.Printf("DEBUG: 启动 %d 个工作协程处理时间同步\n", maxConcurrent)
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for deviceIP := range deviceChan {
				fmt.Printf("DEBUG: [Worker-%d] 开始同步设备 %s 的时间\n", workerID, deviceIP)
				result := s.syncSingleDeviceTime(deviceIP, username, password, dateTimeString, currentTime, workerID)
				resultChan <- result
			}
		}(i)
	}

	// 等待所有goroutine完成并关闭结果通道
	go func() {
		wg.Wait()
		close(resultChan)
		fmt.Printf("DEBUG: 所有设备时间同步处理完成\n")
	}()

	// 收集结果
	var results []models.TimeSyncResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// syncSingleDeviceTime synchronizes the time of a single device
func (s *Service) syncSingleDeviceTime(deviceIP, username, password, dateTimeString string, currentTime time.Time, workerID int) models.TimeSyncResult {
	result := models.TimeSyncResult{
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
}
