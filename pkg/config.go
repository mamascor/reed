package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"

	"lms-tui/logger"
)

// AppConfig holds all application configuration settings
type AppConfig struct {
	CheckDuplicateCans       bool   `json:"check_duplicate_cans"`
	AutoSaveIntervalSeconds  int    `json:"auto_save_interval_seconds"`
	MaxSamplesPerJob         int    `json:"max_samples_per_job"`
	EnableNumericValidation  bool   `json:"enable_numeric_validation"`
	BackupOnSave             bool   `json:"backup_on_save"`
	LogLevel                 string `json:"log_level"`
}

// Default configuration values
var defaultConfig = AppConfig{
	CheckDuplicateCans:       true,
	AutoSaveIntervalSeconds:  30,
	MaxSamplesPerJob:         1000,
	EnableNumericValidation:  true,
	BackupOnSave:             true,
	LogLevel:                 "info",
}

// Global configuration instance
var Config AppConfig

// Configuration variables for backward compatibility
var (
	// CheckDuplicateCans controls whether to check for duplicate can numbers
	// Set to true to enable duplicate checking, false to disable
	CheckDuplicateCans = true
)

// LoadConfig loads configuration from config.json file
func LoadConfig(configPath string) error {
	// Set defaults first
	Config = defaultConfig
	CheckDuplicateCans = defaultConfig.CheckDuplicateCans

	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info.Printf("Config file not found at %s, using default settings", configPath)
			return nil
		}
		logger.Error.Printf("Failed to read config file: %v", err)
		return err
	}

	// Parse JSON
	if err := json.Unmarshal(data, &Config); err != nil {
		logger.Error.Printf("Failed to parse config file: %v", err)
		return err
	}

	// Update backward compatibility variable
	CheckDuplicateCans = Config.CheckDuplicateCans

	logger.Info.Printf("Configuration loaded successfully: DuplicateChecking=%v, NumericValidation=%v",
		Config.CheckDuplicateCans, Config.EnableNumericValidation)

	return nil
}

// SaveConfig saves current configuration to file
func SaveConfig(configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error.Printf("Failed to create config directory: %v", err)
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(Config, "", "  ")
	if err != nil {
		logger.Error.Printf("Failed to marshal config: %v", err)
		return err
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.Error.Printf("Failed to write config file: %v", err)
		return err
	}

	logger.Info.Printf("Configuration saved to %s", configPath)
	return nil
}
