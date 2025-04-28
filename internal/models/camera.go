package models

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
