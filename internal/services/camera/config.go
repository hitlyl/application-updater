package camera

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"application-updater/internal/models"
	"application-updater/internal/services/device"
)

// Config 摄像头配置结构体
type Config struct {
	client        *http.Client
	DeviceService *device.Service
}

// NewConfig 创建配置服务实例
func NewConfig(client *http.Client) *Config {
	return &Config{
		client: client,
	}
}

// SetDeviceManager 设置设备管理器
func (c *Config) SetDeviceService(service *device.Service) {
	c.DeviceService = service
}

// ConfigureCamera 配置摄像头
func (c *Config) ConfigureCamera(ip, token, cameraName, cameraURL string, algorithmType int, existingCamera bool) (bool, string) {
	fmt.Printf("DEBUG: 开始配置摄像头: IP=%s, 摄像头名称=%s, URL=%s, 算法类型=%d\n", ip, cameraName, cameraURL, algorithmType)

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
	resp, err := c.client.Do(req)
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

// GetCameraConfig 获取摄像头配置
func (c *Config) GetCameraConfig(ip, token, taskId string) (*models.CameraConfig, error) {
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
	resp, err := c.client.Do(req)
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
	var response models.CameraConfigResponse
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

// SetCameraIndex 设置摄像头索引
func (c *Config) SetCameraIndex(ip, token, taskId string, config *models.CameraConfig, index int) (bool, string) {
	fmt.Printf("DEBUG: 开始设置摄像头索引: IP=%s, 任务ID=%s, 索引=%d\n", ip, taskId, index)

	// 修改摄像头索引
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
	resp, err := c.client.Do(req)
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

// ConfigureCamerasFromData 批量配置摄像头
func (c *Config) ConfigureCamerasFromData(deviceConfigs []models.ExcelRow, getTokenFunc func(string, string, string) (string, error), username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	// 创建一个带缓冲的结果通道，用于收集所有设备的结果
	resultChan := make(chan []models.CameraConfigResult, len(deviceConfigs))

	// 按设备IP分组
	deviceGroups := make(map[string][]models.ExcelRow)
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
				deviceResults := c.configureCamerasForDevice(deviceIP, configs, getTokenFunc, username, password, urlTemplate, algorithmType, workerId, region)
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
	var results []models.CameraConfigResult
	for deviceResults := range resultChan {
		results = append(results, deviceResults...)
	}

	return results
}

// configureCamerasForDevice 处理单个设备的所有摄像头配置
func (c *Config) configureCamerasForDevice(deviceIP string, configs []models.ExcelRow, getTokenFunc func(string, string, string) (string, error), username, password, urlTemplate string, algorithmType int, workerId int, region string) []models.CameraConfigResult {
	results := make([]models.CameraConfigResult, 0, len(configs))

	// 为每个设备只获取一次token
	fmt.Printf("DEBUG: [Worker-%d] 开始为设备 %s 配置摄像头，共 %d 个\n", workerId, deviceIP, len(configs))
	token, err := getTokenFunc(deviceIP, username, password)
	if err != nil {
		fmt.Printf("ERROR: [Worker-%d] 登录设备 %s 失败: %v\n", workerId, deviceIP, err)
		// 如果登录失败，将此设备下的所有摄像头标记为失败
		for _, config := range configs {
			results = append(results, models.CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    false,
				Message:    fmt.Sprintf("登录设备失败: %v", err),
			})
		}
		return results
	}
	fmt.Printf("DEBUG: [Worker-%d] 成功登录设备 %s，获取到Token\n", workerId, deviceIP)

	// 尝试将设备添加到设备管理中
	if c.DeviceService != nil {
		// 检查设备是否已存在
		_, exists := c.DeviceService.GetDeviceByRegionAndIP(region, deviceIP)
		if !exists {
			// 设备不存在，测试并添加设备
			deviceInfo := &models.Device{
				IP:        deviceIP,
				Status:    "online",
				Region:    region,
				BuildTime: time.Now().Format("2006-01-02 15:04:05"),
			}

			// 生成设备ID
			deviceInfo.ID = models.GenerateDeviceID(region, deviceIP)

			// 添加设备
			_, err := c.DeviceService.AddDevice(*deviceInfo)
			if err != nil {
				fmt.Printf("WARN: [Worker-%d] 将设备 %s 添加到设备管理失败: %v\n", workerId, deviceIP, err)
			} else {
				fmt.Printf("INFO: [Worker-%d] 已将设备 %s 添加到设备管理中，区域: %s\n", workerId, deviceIP, region)
			}
		} else {
			fmt.Printf("INFO: [Worker-%d] 设备 %s 已存在于设备管理中\n", workerId, deviceIP)
		}
	}

	// 获取摄像头任务列表
	tasksClient := NewTasks(c.client)
	cameras, err := tasksClient.GetCameraTasksWithToken(deviceIP, token)
	if err != nil {
		fmt.Printf("ERROR: [Worker-%d] 获取摄像头任务列表失败: %v\n", workerId, deviceIP)
		// 如果获取任务列表失败，将此设备下的所有摄像头标记为失败
		for _, config := range configs {
			results = append(results, models.CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    false,
				Message:    fmt.Sprintf("获取摄像头任务列表失败: %v", err),
			})
		}
		return results
	}

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

			// 检查摄像头是否已存在
			existingCamera := false
			for _, camera := range cameras {
				if camera.TaskID == config.CameraName {
					existingCamera = true
					fmt.Printf("DEBUG: [Worker-%d] 找到已存在的摄像头任务: %s\n", workerId, config.CameraName)
					break
				}
			}

			// 配置摄像头，使用已获取的token
			fmt.Printf("DEBUG: [Worker-%d] 配置摄像头: %s 在设备 %s\n", workerId, config.CameraName, deviceIP)
			success, message := c.ConfigureCamera(deviceIP, token, config.CameraName, cameraURL, algorithmType, existingCamera)

			// 如果配置成功，设置摄像头索引
			if success {
				fmt.Printf("DEBUG: [Worker-%d] 摄像头配置成功，等待500毫秒后设置索引...\n", workerId)
				// 等待500毫秒，确保摄像头任务已初始化
				time.Sleep(500 * time.Millisecond)

				fmt.Printf("DEBUG: [Worker-%d] 设置摄像头索引: %s -> %d\n", workerId, config.CameraName, cameraIndex)

				// 获取摄像头配置
				cameraConfig, err := c.GetCameraConfig(deviceIP, token, config.CameraName)
				if err != nil {
					fmt.Printf("ERROR: [Worker-%d] 获取摄像头配置失败: %v\n", workerId, err)
					success = false
					message += fmt.Sprintf(". 获取摄像头配置失败: %v", err)
				} else {
					// 设置摄像头索引
					indexSuccess, indexMessage := c.SetCameraIndex(deviceIP, token, config.CameraName, cameraConfig, cameraIndex)
					if !indexSuccess {
						message += ". " + indexMessage
					} else {
						message += ". " + indexMessage
					}
				}
			}

			// 记录结果
			results = append(results, models.CameraConfigResult{
				DeviceIP:   deviceIP,
				CameraName: config.CameraName,
				Success:    success,
				Message:    message,
			})
		} else {
			// 摄像头信息格式错误
			results = append(results, models.CameraConfigResult{
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
