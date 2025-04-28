package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"application-updater/internal/models"
)

// Auth 设备认证结构体
type Auth struct {
	client *http.Client
}

// NewAuth 创建认证服务实例
func NewAuth(client *http.Client) *Auth {
	return &Auth{
		client: client,
	}
}

// LoginToDevice 登录到设备并获取令牌
func (a *Auth) LoginToDevice(ip, username, password string) (string, error) {
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
	var loginResp models.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		fmt.Printf("ERROR: 解析响应失败: %v\n", err)
		return "", err
	}

	// 检查响应码
	if loginResp.Code != 0 {
		fmt.Printf("ERROR: 登录失败: %s\n", loginResp.Msg)
		return "", fmt.Errorf("登录失败: %s", loginResp.Msg)
	}

	// 验证令牌
	token := loginResp.Result.Token
	if token == "" {
		fmt.Printf("ERROR: 获取到空令牌\n")
		return "", fmt.Errorf("获取到空令牌")
	}

	fmt.Printf("DEBUG: 登录成功，获取到令牌: %s\n", token)
	return token, nil
}
