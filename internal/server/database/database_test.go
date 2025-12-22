package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitDatabase_Success тестирует успешное подключение к БД
func TestInitDatabase_Success(t *testing.T) {
	// Создаем временный каталог для теста
	tmpDir, err := ioutil.TempDir("", "database_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Создаем временный файл для БД
	dbPath := filepath.ToSlash(filepath.Join(tmpDir, "test.db"))

	// Создаем папку для миграций
	migrationsDir := filepath.Join(tmpDir, "migrations")
	err = os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Создаем простую миграцию
	migrationFile := filepath.Join(migrationsDir, "1_test.up.sql")
	err = ioutil.WriteFile(migrationFile, []byte("CREATE TABLE test (id INTEGER PRIMARY KEY);"), 0644)
	require.NoError(t, err)

	// Меняем рабочую директорию
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Инициализируем БД
	db, err := InitDatabase(dbPath)
	require.NoError(t, err, "InitDatabase should not fail")
	defer db.Close()

	// Проверяем, что подключение работает
	err = db.Ping()
	assert.NoError(t, err)

	// Проверяем, что таблица создана
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "test", tableName)
}
