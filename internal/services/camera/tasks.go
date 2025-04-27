package camera

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"application-updater/internal/models"
)

// Tasks 摄像头任务结构体
type Tasks struct {
	client *http.Client
}

// NewTasks 创建任务服务实例
func NewTasks(client *http.Client) *Tasks {
	return &Tasks{
		client: client,
	}
}

// GetCameraTasks 获取摄像头任务列表
func (t *Tasks) GetCameraTasks(ip, username, password string, getTokenFunc func(string, string, string) (string, error)) ([]models.Camera, error) {
	fmt.Printf("DEBUG: 开始获取摄像头任务列表: IP=%s\n", ip)

	// 1. 先登录设备获取token
	token, err := getTokenFunc(ip, username, password)
	if err != nil {
		fmt.Printf("ERROR: 登录设备失败: %v\n", err)
		return nil, fmt.Errorf("登录设备失败: %w", err)
	}
	fmt.Printf("DEBUG: 成功登录设备，获取到Token\n")

	return t.GetCameraTasksWithToken(ip, token)
}

// GetCameraTasksWithToken 使用已有的token获取摄像头任务列表
func (t *Tasks) GetCameraTasksWithToken(ip, token string) ([]models.Camera, error) {
	fmt.Printf("DEBUG: 开始使用Token获取摄像头任务列表: IP=%s\n", ip)

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
	resp, err := t.client.Do(req)
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
	var response models.CameraTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if response.Code != 0 {
		fmt.Printf("ERROR: 获取任务列表失败: 代码=%d, 消息=%s\n", response.Code, response.Msg)
		return nil, fmt.Errorf("获取任务列表失败: %s", response.Msg)
	}

	// 转换为Camera结构体
	cameras := make([]models.Camera, 0, len(response.Result.Items))
	for _, item := range response.Result.Items {
		cameras = append(cameras, models.Camera{
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
