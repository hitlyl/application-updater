package models

// ExcelSheetData represents the data of a sheet in the Excel file
type ExcelSheetData struct {
	SheetName string     `json:"sheetName"`
	Rows      []ExcelRow `json:"rows"`
}

// ExcelRow represents a row in the Excel file
type ExcelRow struct {
	DeviceIP    string `json:"deviceIp"`
	CameraName  string `json:"cameraName"`
	CameraInfo  string `json:"cameraInfo"`
	DeviceIndex int    `json:"deviceIndex"`
	Selected    bool   `json:"selected"`
}
