// Package config provides configuration management for the GophKeeper client.
// It handles loading and saving client settings, synchronization state, and data directory management.
//
//go:generate mockery --name=ManagerInterface --output=../mocks --outpkg=mocks --case=underscore
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClientConfig represents the client configuration including authentication tokens and user ID.
type ClientConfig struct {
	Token  string `json:"token"`
	UserID string `json:"userID"`
}

// SyncConfig represents synchronization configuration including the last sync timestamp.
type SyncConfig struct {
	LastSync int64 `json:"last_sync"`
}

// ManagerInterface defines the interface for configuration management operations.
type ManagerInterface interface {
	// LoadClientConfig loads the client configuration from persistent storage.
	LoadClientConfig() (*ClientConfig, error)

	// SaveClientConfig saves the client configuration to persistent storage.
	SaveClientConfig(config *ClientConfig) error

	// LoadSyncConfig loads the synchronization configuration.
	LoadSyncConfig() (*SyncConfig, error)

	// SaveSyncConfig saves the synchronization configuration.
	SaveSyncConfig(config *SyncConfig) error

	// GetDataDir returns the path to the data storage directory.
	GetDataDir() string
}

// Manager implements the ManagerInterface for configuration management.
type Manager struct {
	dataDir    string
	configFile string
	syncFile   string
}

// NewDefaultManager creates a new Manager instance using the default configuration directory.
func NewDefaultManager() (*Manager, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	return NewManager(configDir)
}

// NewManager creates a new Manager instance with the specified configuration directory.
func NewManager(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Manager{
		dataDir:    filepath.Join(configDir, "data"),
		configFile: filepath.Join(configDir, "config.json"),
		syncFile:   filepath.Join(configDir, "sync.json"),
	}, nil
}

// GetConfigDir returns the platform-specific configuration directory for GophKeeper.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".gophkeeper")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

// LoadClientConfig implements ManagerInterface.LoadClientConfig.
func (m *Manager) LoadClientConfig() (*ClientConfig, error) {
	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		return &ClientConfig{}, nil
	}

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// SaveClientConfig implements ManagerInterface.SaveClientConfig.
func (m *Manager) SaveClientConfig(config *ClientConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadSyncConfig implements ManagerInterface.LoadSyncConfig.
func (m *Manager) LoadSyncConfig() (*SyncConfig, error) {
	if _, err := os.Stat(m.syncFile); os.IsNotExist(err) {
		return &SyncConfig{LastSync: 0}, nil
	}

	data, err := os.ReadFile(m.syncFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read sync config: %w", err)
	}

	var config SyncConfig
	json.Unmarshal(data, &config)

	return &config, nil
}

// SaveSyncConfig implements ManagerInterface.SaveSyncConfig.
func (m *Manager) SaveSyncConfig(config *SyncConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync config: %w", err)
	}

	if err := os.WriteFile(m.syncFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write sync config: %w", err)
	}

	return nil
}

// GetDataDir implements ManagerInterface.GetDataDir.
func (m *Manager) GetDataDir() string {
	return m.dataDir
}
