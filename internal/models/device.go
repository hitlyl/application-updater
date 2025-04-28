package models

import "github.com/google/uuid"

// Device represents a device in the system
type Device struct {
	ID        string `json:"id"` // 设备唯一标识，由区域+ip组成
	IP        string `json:"ip"`
	BuildTime string `json:"buildTime"`
	Status    string `json:"status"`
	Region    string `json:"region,omitempty"` // 添加区域字段，omitempty使得该字段在为空时不会出现在JSON中，保持向后兼容
}

// 根据区域和IP创建设备ID
func GenerateDeviceID(region, ip string) string {
	return uuid.New().String()
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

// TimeSyncResult represents the result of a time sync operation
type TimeSyncResult struct {
	IP        string `json:"ip"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
