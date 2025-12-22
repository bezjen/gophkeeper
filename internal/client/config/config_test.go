package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewDefaultManager использует мокирование os.UserHomeDir через подмену переменной окружения
func TestNewDefaultManager(t *testing.T) {
	// Сохраняем оригинальное значение HOME
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	t.Run("successful creation", func(t *testing.T) {
		// Используем временную директорию
		tempDir := t.TempDir()

		// Устанавливаем переменные окружения для разных ОС
		os.Setenv("HOME", tempDir)
		os.Setenv("USERPROFILE", tempDir) // Для Windows

		manager, err := NewDefaultManager()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if manager == nil {
			t.Fatal("expected manager to be not nil")
		}

		// На Windows проверяем по-другому
		if runtime.GOOS == "windows" {
			// На Windows os.UserHomeDir() может использовать другие переменные
			// Поэтому просто проверяем что менеджер создан без ошибок
			return
		}

		// Для Unix-систем проверяем создание директории
		expectedConfigDir := filepath.Join(tempDir, ".gophkeeper")
		if _, err := os.Stat(expectedConfigDir); os.IsNotExist(err) {
			t.Fatalf("expected config directory to be created at %s", expectedConfigDir)
		}
	})

	t.Run("error when home directory not found", func(t *testing.T) {
		// На Windows этот тест может не работать как ожидается
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test on Windows due to different os.UserHomeDir behavior")
		}

		// Устанавливаем пустую HOME
		os.Setenv("HOME", "")
		os.Setenv("USERPROFILE", "")

		manager, err := NewDefaultManager()
		if err == nil {
			t.Fatal("expected error when home directory not found")
		}
		if manager != nil {
			t.Fatal("expected manager to be nil when error occurs")
		}
	})
}

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("successful creation", func(t *testing.T) {
		manager, err := NewManager(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if manager == nil {
			t.Fatal("expected manager to be not nil")
		}

		// Проверяем пути
		expectedConfigFile := filepath.Join(tempDir, "config.json")
		if manager.configFile != expectedConfigFile {
			t.Errorf("expected config file path %s, got %s", expectedConfigFile, manager.configFile)
		}

		expectedSyncFile := filepath.Join(tempDir, "sync.json")
		if manager.syncFile != expectedSyncFile {
			t.Errorf("expected sync file path %s, got %s", expectedSyncFile, manager.syncFile)
		}

		expectedDataDir := filepath.Join(tempDir, "data")
		if manager.dataDir != expectedDataDir {
			t.Errorf("expected data dir path %s, got %s", expectedDataDir, manager.dataDir)
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "nonexistent")

		manager, err := NewManager(nonExistentDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if manager == nil {
			t.Fatal("expected manager to be not nil")
		}

		// Проверяем, что директория создана
		if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
			t.Fatalf("expected directory to be created at %s", nonExistentDir)
		}
	})
}

func TestManager_LoadClientConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		dataDir:    filepath.Join(tempDir, "data"),
		configFile: filepath.Join(tempDir, "config.json"),
		syncFile:   filepath.Join(tempDir, "sync.json"),
	}

	t.Run("file does not exist", func(t *testing.T) {
		config, err := manager.LoadClientConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be not nil")
		}
		if config.Token != "" || config.UserID != "" {
			t.Errorf("expected empty config, got %+v", config)
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		expectedConfig := &ClientConfig{
			Token:  "test-token-123",
			UserID: "user-456",
		}

		data, err := json.Marshal(expectedConfig)
		if err != nil {
			t.Fatalf("failed to marshal test config: %v", err)
		}

		if err := os.WriteFile(manager.configFile, data, 0600); err != nil {
			t.Fatalf("failed to write test config file: %v", err)
		}

		config, err := manager.LoadClientConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if config.Token != expectedConfig.Token {
			t.Errorf("expected token %s, got %s", expectedConfig.Token, config.Token)
		}
		if config.UserID != expectedConfig.UserID {
			t.Errorf("expected userID %s, got %s", expectedConfig.UserID, config.UserID)
		}
	})

	t.Run("invalid json in config file", func(t *testing.T) {
		invalidJSON := []byte(`{invalid json}`)
		if err := os.WriteFile(manager.configFile, invalidJSON, 0600); err != nil {
			t.Fatalf("failed to write invalid json file: %v", err)
		}

		config, err := manager.LoadClientConfig()
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
		if config != nil {
			t.Fatal("expected config to be nil when error occurs")
		}
	})

	t.Run("empty config file", func(t *testing.T) {
		if err := os.WriteFile(manager.configFile, []byte(""), 0600); err != nil {
			t.Fatalf("failed to write empty file: %v", err)
		}

		config, err := manager.LoadClientConfig()
		if err == nil {
			t.Fatal("expected error for empty json")
		}
		if config != nil {
			t.Fatal("expected config to be nil when error occurs")
		}
	})
}

func TestManager_SaveClientConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		dataDir:    filepath.Join(tempDir, "data"),
		configFile: filepath.Join(tempDir, "config.json"),
		syncFile:   filepath.Join(tempDir, "sync.json"),
	}

	t.Run("successful save", func(t *testing.T) {
		config := &ClientConfig{
			Token:  "save-test-token",
			UserID: "save-test-user",
		}

		err := manager.SaveClientConfig(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Проверяем, что файл создан
		if _, err := os.Stat(manager.configFile); os.IsNotExist(err) {
			t.Fatal("expected config file to be created")
		}

		// Читаем и проверяем содержимое
		data, err := os.ReadFile(manager.configFile)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		var loadedConfig ClientConfig
		if err := json.Unmarshal(data, &loadedConfig); err != nil {
			t.Fatalf("failed to unmarshal config: %v", err)
		}

		if loadedConfig.Token != config.Token {
			t.Errorf("expected token %s, got %s", config.Token, loadedConfig.Token)
		}
		if loadedConfig.UserID != config.UserID {
			t.Errorf("expected userID %s, got %s", config.UserID, loadedConfig.UserID)
		}
	})

	t.Run("permissions are set correctly", func(t *testing.T) {
		// На Windows права доступа работают иначе, пропускаем тест
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permissions test on Windows")
		}

		config := &ClientConfig{
			Token:  "permission-test",
			UserID: "permission-user",
		}

		err := manager.SaveClientConfig(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Проверяем права доступа
		info, err := os.Stat(manager.configFile)
		if err != nil {
			t.Fatalf("failed to stat config file: %v", err)
		}

		expectedPerm := os.FileMode(0600)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("expected file mode %v, got %v", expectedPerm, info.Mode().Perm())
		}
	})
}

func TestManager_LoadSyncConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		dataDir:    filepath.Join(tempDir, "data"),
		configFile: filepath.Join(tempDir, "config.json"),
		syncFile:   filepath.Join(tempDir, "sync.json"),
	}

	t.Run("file does not exist", func(t *testing.T) {
		config, err := manager.LoadSyncConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be not nil")
		}
		if config.LastSync != 0 {
			t.Errorf("expected LastSync 0, got %d", config.LastSync)
		}
	})

	t.Run("valid sync file", func(t *testing.T) {
		expectedConfig := &SyncConfig{
			LastSync: 1234567890,
		}

		data, err := json.Marshal(expectedConfig)
		if err != nil {
			t.Fatalf("failed to marshal test config: %v", err)
		}

		if err := os.WriteFile(manager.syncFile, data, 0600); err != nil {
			t.Fatalf("failed to write test config file: %v", err)
		}

		config, err := manager.LoadSyncConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if config.LastSync != expectedConfig.LastSync {
			t.Errorf("expected LastSync %d, got %d", expectedConfig.LastSync, config.LastSync)
		}
	})

	t.Run("invalid json in sync file", func(t *testing.T) {
		invalidJSON := []byte(`{invalid json}`)
		if err := os.WriteFile(manager.syncFile, invalidJSON, 0600); err != nil {
			t.Fatalf("failed to write invalid json file: %v", err)
		}

		config, err := manager.LoadSyncConfig()
		// Note: json.Unmarshal error is ignored in the implementation
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return zero value when unmarshal fails
		if config.LastSync != 0 {
			t.Errorf("expected LastSync 0 when invalid json, got %d", config.LastSync)
		}
	})
}

func TestManager_SaveSyncConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		dataDir:    filepath.Join(tempDir, "data"),
		configFile: filepath.Join(tempDir, "config.json"),
		syncFile:   filepath.Join(tempDir, "sync.json"),
	}

	t.Run("successful save", func(t *testing.T) {
		config := &SyncConfig{
			LastSync: 9876543210,
		}

		err := manager.SaveSyncConfig(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Проверяем, что файл создан
		if _, err := os.Stat(manager.syncFile); os.IsNotExist(err) {
			t.Fatal("expected sync file to be created")
		}

		// Читаем и проверяем содержимое
		data, err := os.ReadFile(manager.syncFile)
		if err != nil {
			t.Fatalf("failed to read sync file: %v", err)
		}

		var loadedConfig SyncConfig
		if err := json.Unmarshal(data, &loadedConfig); err != nil {
			t.Fatalf("failed to unmarshal sync config: %v", err)
		}

		if loadedConfig.LastSync != config.LastSync {
			t.Errorf("expected LastSync %d, got %d", config.LastSync, loadedConfig.LastSync)
		}
	})
}

func TestManager_GetDataDir(t *testing.T) {
	tempDir := t.TempDir()
	expectedDataDir := filepath.Join(tempDir, "data")

	manager := &Manager{
		dataDir:    expectedDataDir,
		configFile: filepath.Join(tempDir, "config.json"),
		syncFile:   filepath.Join(tempDir, "sync.json"),
	}

	dataDir := manager.GetDataDir()
	if dataDir != expectedDataDir {
		t.Errorf("expected data dir %s, got %s", expectedDataDir, dataDir)
	}
}

func TestGetConfigDir(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir) // Для Windows

	t.Run("successful get config dir", func(t *testing.T) {
		configDir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// На Windows os.UserHomeDir() может вести себя по-другому
		// Поэтому просто проверяем что функция отработала без ошибок
		if configDir == "" {
			t.Error("expected config dir not to be empty")
		}
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		// На Windows права доступа работают иначе, пропускаем тест
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permissions test on Windows")
		}

		configDir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("failed to stat config directory: %v", err)
		}

		// Проверяем, что директория имеет правильные права доступа (0700)
		if info.Mode().Perm() != 0700 {
			t.Errorf("expected directory permissions 0700, got %v", info.Mode().Perm())
		}
	})

	t.Run("error when home directory not found", func(t *testing.T) {
		// На Windows этот тест может не работать как ожидается
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test on Windows due to different os.UserHomeDir behavior")
		}

		// Устанавливаем пустые переменные окружения
		os.Setenv("HOME", "")
		os.Setenv("USERPROFILE", "")

		_, err := GetConfigDir()
		if err == nil {
			t.Fatal("expected error when home directory not found")
		}
	})
}

func TestClientConfigJSON(t *testing.T) {
	t.Run("marshaling and unmarshaling", func(t *testing.T) {
		config := &ClientConfig{
			Token:  "json-test-token",
			UserID: "json-test-user",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ClientConfig
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Token != config.Token {
			t.Errorf("expected token %s, got %s", config.Token, unmarshaled.Token)
		}
		if unmarshaled.UserID != config.UserID {
			t.Errorf("expected userID %s, got %s", config.UserID, unmarshaled.UserID)
		}
	})
}

func TestSyncConfigJSON(t *testing.T) {
	t.Run("marshaling and unmarshaling", func(t *testing.T) {
		config := &SyncConfig{
			LastSync: 1234567890,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled SyncConfig
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.LastSync != config.LastSync {
			t.Errorf("expected LastSync %d, got %d", config.LastSync, unmarshaled.LastSync)
		}
	})
}
