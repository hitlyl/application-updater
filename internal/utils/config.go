package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig loads configuration from a JSON file into a struct
func LoadConfig(filePath string, config interface{}) error {
	if !FileExists(filePath) {
		return fmt.Errorf("config file does not exist: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// SaveConfig saves a struct as JSON to a file
func SaveConfig(filePath string, config interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := EnsureDirExists(dir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigDir returns the configuration directory for the application
func GetConfigDir() string {
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		return filepath.Join(".", "configs")
	}

	execDir := filepath.Dir(execPath)
	configDir := filepath.Join(execDir, "configs")

	// Ensure the config directory exists
	if err := EnsureDirExists(configDir); err != nil {
		fmt.Printf("Failed to create config directory: %v\n", err)
		// Fallback to current directory
		return "configs"
	}

	return configDir
}

// GetDefaultConfigPath returns the default path for a configuration file
func GetDefaultConfigPath(filename string) string {
	// Check if we can use the app's config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if user config dir is not available
		return filepath.Join("configs", filename)
	}

	// Use the app's name in the path
	appConfigDir := filepath.Join(configDir, "application-updater")
	return filepath.Join(appConfigDir, filename)
}

// MergeConfig merges default config values with loaded config
// Any zero values in loaded config will be replaced with defaults
func MergeConfig(loadedConfig, defaultConfig interface{}) error {
	// Convert both configs to map[string]interface{}
	loadedMap := make(map[string]interface{})
	defaultMap := make(map[string]interface{})

	// Marshal and unmarshal to convert structs to maps
	loadedData, err := json.Marshal(loadedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal loaded config: %w", err)
	}

	defaultData, err := json.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := json.Unmarshal(loadedData, &loadedMap); err != nil {
		return fmt.Errorf("failed to unmarshal loaded config: %w", err)
	}

	if err := json.Unmarshal(defaultData, &defaultMap); err != nil {
		return fmt.Errorf("failed to unmarshal default config: %w", err)
	}

	// Merge maps
	mergedMap := mergeMapValues(loadedMap, defaultMap)

	// Marshal merged map
	mergedData, err := json.Marshal(mergedMap)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	// Unmarshal back to the loaded config struct
	if err := json.Unmarshal(mergedData, loadedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal merged config: %w", err)
	}

	return nil
}

// mergeMapValues merges a loaded map with a default map
// If a value in loaded is nil/zero, it will be replaced with the default value
func mergeMapValues(loaded, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// First, copy all values from loaded
	for k, v := range loaded {
		result[k] = v
	}

	// Then replace nil/zero values with defaults
	for k, defaultVal := range defaults {
		loadedVal, exists := loaded[k]

		if !exists || isZeroValue(loadedVal) {
			// Use default value if key doesn't exist or value is zero
			result[k] = defaultVal
		} else if nestedMapLoaded, ok := loadedVal.(map[string]interface{}); ok {
			// If both are maps, recursively merge them
			if nestedMapDefault, ok := defaultVal.(map[string]interface{}); ok {
				result[k] = mergeMapValues(nestedMapLoaded, nestedMapDefault)
			}
		}
	}

	return result
}

// isZeroValue checks if a value is a zero value (nil, empty string, 0, etc.)
func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return val == 0
	case bool:
		return val == false
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	default:
		return false
	}
}
