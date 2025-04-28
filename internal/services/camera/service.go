package camera

import (
	"net/http"

	"application-updater/internal/models"
	"application-updater/internal/services/device"
)

// Service 摄像头服务接口
type Service struct {
	Tasks  *Tasks
	Config *Config
}

// NewService 创建摄像头服务实例
func NewService(client *http.Client) *Service {
	config := NewConfig(client)
	return &Service{
		Tasks:  NewTasks(client),
		Config: config,
	}
}

// SetDeviceManager 设置设备管理器
func (s *Service) SetDeviceService(service *device.Service) {
	s.Config.SetDeviceService(service)
}

// GetCameraTasks 获取摄像头任务列表
func (s *Service) GetCameraTasks(ip, username, password string, getTokenFunc func(string, string, string) (string, error)) ([]models.Camera, error) {
	return s.Tasks.GetCameraTasks(ip, username, password, getTokenFunc)
}

// GetCameraTasksWithToken 使用已有的token获取摄像头任务列表
func (s *Service) GetCameraTasksWithToken(ip, token string) ([]models.Camera, error) {
	return s.Tasks.GetCameraTasksWithToken(ip, token)
}

// ConfigureCamera 配置摄像头
func (s *Service) ConfigureCamera(ip, token, cameraName, cameraURL string, algorithmType int, existingCamera bool) (bool, string) {
	return s.Config.ConfigureCamera(ip, token, cameraName, cameraURL, algorithmType, existingCamera)
}

// GetCameraConfig 获取摄像头配置
func (s *Service) GetCameraConfig(ip, token, taskId string) (*models.CameraConfig, error) {
	return s.Config.GetCameraConfig(ip, token, taskId)
}

// SetCameraIndex 设置摄像头索引
func (s *Service) SetCameraIndex(ip, token, taskId string, config *models.CameraConfig, index int) (bool, string) {
	return s.Config.SetCameraIndex(ip, token, taskId, config, index)
}

// ConfigureCamerasFromData 批量配置摄像头
func (s *Service) ConfigureCamerasFromData(deviceConfigs []models.ExcelRow, getTokenFunc func(string, string, string) (string, error), username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	return s.Config.ConfigureCamerasFromData(deviceConfigs, getTokenFunc, username, password, urlTemplate, algorithmType, region)
}
