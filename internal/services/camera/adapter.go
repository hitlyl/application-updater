package camera

import (
	"application-updater/internal/models"
	"application-updater/internal/services/device"
)

// CameraServiceAdapter adapts the camera service to the excel.CameraService interface
type CameraServiceAdapter struct {
	service *Service
	auth    *device.Auth
}

// NewCameraServiceAdapter creates a new adapter for the camera service
func NewCameraServiceAdapter(service *Service, auth *device.Auth) *CameraServiceAdapter {
	return &CameraServiceAdapter{
		service: service,
		auth:    auth,
	}
}

// ConfigureCamerasFromData adapts the ConfigureCamerasFromData method to match the excel.CameraService interface
func (a *CameraServiceAdapter) ConfigureCamerasFromData(deviceConfigs []models.ExcelRow, username, password, urlTemplate string, algorithmType int, region string) []models.CameraConfigResult {
	// Create a function to get token that can be passed to the original method
	getTokenFunc := func(ip, user, pass string) (string, error) {
		return a.auth.LoginToDevice(ip, user, pass)
	}

	// Call the original method with the token function and region
	return a.service.Config.ConfigureCamerasFromData(deviceConfigs, getTokenFunc, username, password, urlTemplate, algorithmType, region)
}
