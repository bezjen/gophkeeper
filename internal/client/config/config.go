//go:generate mockery --name=ManagerInterface --output=../mocks --outpkg=mocks --case=underscore
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ClientConfig struct {
	Token  string `json:"token"`
	UserID string `json:"userID"`
}

type SyncConfig struct {
	LastSync int64 `json:"last_sync"`
}

type ManagerInterface interface {
	LoadClientConfig() (*ClientConfig, error)
	SaveClientConfig(config *ClientConfig) error
	LoadSyncConfig() (*SyncConfig, error)
	SaveSyncConfig(config *SyncConfig) error
	GetDataDir() string
}

type Manager struct {
	dataDir    string
	configFile string
	syncFile   string
}

func NewDefaultManager() (*Manager, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	return NewManager(configDir)
}

func NewManager(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Manager{
		dataDir:    filepath.Join(configDir, "data"),
		configFile: filepath.Join(configDir, "config.json"),
	}, nil
}

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

func (m *Manager) LoadClientConfig() (*ClientConfig, error) {
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

func (m *Manager) LoadSyncConfig() (*SyncConfig, error) {
	data, err := os.ReadFile(m.syncFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read sync config: %w", err)
	}

	var config SyncConfig
	json.Unmarshal(data, &config)

	return &config, nil
}

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

func (m *Manager) GetDataDir() string {
	return m.dataDir
}
