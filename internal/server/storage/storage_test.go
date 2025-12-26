package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/models"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
)

func TestSQLStorage_UserOperations(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	storage := NewStorage(db).(*SQLStorage)

	t.Run("Создание пользователя", func(t *testing.T) {
		user := &models.User{
			ID:        "user123",
			Username:  "testuser",
			Password:  "hashedpassword",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
		}

		mock.ExpectExec("INSERT INTO t_user").
			WithArgs(user.ID, user.Username, user.Password, user.Email, user.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := storage.CreateUser(context.Background(), user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Получение пользователя по имени", func(t *testing.T) {
		expectedUser := &models.User{
			ID:        "user123",
			Username:  "testuser",
			Password:  "hashedpassword",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
		}

		rows := sqlmock.NewRows([]string{"id", "username", "password", "email", "created_at"}).
			AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Password,
				expectedUser.Email, expectedUser.CreatedAt)

		mock.ExpectQuery("SELECT id, username, password, email, created_at FROM t_user WHERE username = \\$1").
			WithArgs("testuser").
			WillReturnRows(rows)

		user, err := storage.GetUserByUsername(context.Background(), "testuser")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Пользователь не найден", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, username, password, email, created_at FROM t_user WHERE username = \\$1").
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		user, err := storage.GetUserByUsername(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, errors.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_DataOperations(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	storage := NewStorage(db).(*SQLStorage)
	ctx := context.Background()
	userID := "user123"

	t.Run("Сохранение данных", func(t *testing.T) {
		data := &pb.DataRecord{
			Id:            "data123",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Test Login",
			EncryptedData: []byte("encrypted"),
			Metadata:      map[string]string{"url": "example.com"},
			Version:       1, // Version = 1, поэтому проверка версии не должна выполняться
			UpdatedAt:     time.Now().Unix(),
			Deleted:       false,
		}

		metadataJSON, _ := json.Marshal(data.Metadata)

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO t_user_data`).
			WithArgs(data.Id, userID, data.Type, data.Name, data.EncryptedData,
				metadataJSON, data.Version, data.UpdatedAt, data.Deleted).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := storage.StoreData(ctx, userID, data)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Получение данных", func(t *testing.T) {
		dataID := "data123"
		metadataJSON, _ := json.Marshal(map[string]string{"url": "example.com"})

		rows := sqlmock.NewRows([]string{"id", "type", "name", "encrypted_data", "metadata",
			"version", "updated_at", "deleted"}).
			AddRow("data123", pb.DataType_LOGIN_PASSWORD, "Test Login",
				[]byte("encrypted"), metadataJSON, 1, time.Now().Unix(), false)

		mock.ExpectQuery("SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted").
			WithArgs(dataID, userID).
			WillReturnRows(rows)

		data, err := storage.RetrieveData(ctx, userID, dataID)
		assert.NoError(t, err)
		assert.Equal(t, dataID, data.Id)
		assert.Equal(t, "Test Login", data.Name)
		assert.False(t, data.Deleted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Удаление данных", func(t *testing.T) {
		dataID := "data123"

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE t_user_data").
			WithArgs(sqlmock.AnyArg(), dataID, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := storage.DeleteData(ctx, userID, dataID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Список данных", func(t *testing.T) {
		metadataJSON, _ := json.Marshal(map[string]string{"url": "example.com"})

		rows := sqlmock.NewRows([]string{"id", "type", "name", "encrypted_data", "metadata",
			"version", "updated_at", "deleted"}).
			AddRow("data1", pb.DataType_LOGIN_PASSWORD, "Login 1",
				[]byte("encrypted1"), metadataJSON, 1, time.Now().Unix(), false).
			AddRow("data2", pb.DataType_BANK_CARD, "Card 1",
				[]byte("encrypted2"), metadataJSON, 1, time.Now().Unix(), false)

		mock.ExpectQuery("SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted").
			WithArgs(userID, pb.DataType_LOGIN_PASSWORD).
			WillReturnRows(rows)

		items, err := storage.ListData(ctx, userID, pb.DataType_LOGIN_PASSWORD)
		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, "Login 1", items[0].Name)
		assert.Equal(t, "Card 1", items[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Синхронизация данных", func(t *testing.T) {
		localChanges := []*pb.DataRecord{
			{
				Id:        "data1",
				Type:      pb.DataType_LOGIN_PASSWORD,
				Name:      "Updated Login",
				Version:   2,
				UpdatedAt: time.Now().Unix(),
				Deleted:   false,
			},
		}
		lastSync := time.Now().Add(-1 * time.Hour).Unix()

		// Мокаем транзакцию
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE t_user_data|INSERT INTO t_user_data").
			WillReturnResult(sqlmock.NewResult(1, 1))

		metadataJSON, _ := json.Marshal(map[string]string{})
		rows := sqlmock.NewRows([]string{"id", "type", "name", "encrypted_data", "metadata",
			"version", "updated_at", "deleted"}).
			AddRow("data2", pb.DataType_TEXT_DATA, "Text Data",
				[]byte("encrypted"), metadataJSON, 1, time.Now().Unix(), false)

		mock.ExpectQuery("SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted").
			WithArgs(userID, lastSync).
			WillReturnRows(rows)

		mock.ExpectCommit()

		serverData, err := storage.ProcessSync(ctx, userID, localChanges, lastSync)
		assert.NoError(t, err)
		assert.Len(t, serverData, 1)
		assert.Equal(t, "Text Data", serverData[0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ping базы данных", func(t *testing.T) {
		mock.ExpectPing()
		err := storage.Ping(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_ErrorCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	storage := NewStorage(db).(*SQLStorage)
	ctx := context.Background()

	t.Run("Конфликт версий при сохранении", func(t *testing.T) {
		data := &pb.DataRecord{
			Id:        "data123",
			Version:   2,
			UpdatedAt: time.Now().Unix(),
		}

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT version FROM t_user_data WHERE id = \\$1 AND user_id = \\$2").
			WithArgs(data.Id, "user123").
			WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))

		err := storage.StoreData(ctx, "user123", data)
		assert.Error(t, err)
		assert.Equal(t, errors.ErrVersionConflict, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Данные не найдены при получении", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted").
			WithArgs("nonexistent", "user123").
			WillReturnError(sql.ErrNoRows)

		data, err := storage.RetrieveData(ctx, "user123", "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Equal(t, errors.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Данные удалены", func(t *testing.T) {
		metadataJSON, _ := json.Marshal(map[string]string{})

		rows := sqlmock.NewRows([]string{"id", "type", "name", "encrypted_data", "metadata",
			"version", "updated_at", "deleted"}).
			AddRow("deleted", pb.DataType_LOGIN_PASSWORD, "Deleted",
				[]byte("encrypted"), metadataJSON, 1, time.Now().Unix(), true)

		mock.ExpectQuery("SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted").
			WithArgs("deleted", "user123").
			WillReturnRows(rows)

		data, err := storage.RetrieveData(ctx, "user123", "deleted")
		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Equal(t, errors.ErrDeleted, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
