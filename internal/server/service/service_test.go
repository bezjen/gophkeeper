package service

import (
	"context"
	"errors"
	errors2 "github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/middleware"
	"github.com/bezjen/gophkeeper/internal/server/mocks"
	"github.com/bezjen/gophkeeper/internal/server/models"
	"testing"
	"time"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestService_Register(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	req := &pb.RegisterRequest{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}

	authReq := &models.AuthRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	}

	authResp := &models.AuthResponse{
		UserId: "user123",
		Token:  "jwt.token.here",
	}

	mockAuth.On("Register", mock.Anything, authReq).Return(authResp, nil)

	resp, err := service.Register(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, authResp.UserId, resp.UserId)
	assert.Equal(t, authResp.Token, resp.Token)
	mockAuth.AssertExpectations(t)
}

func TestService_Login(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	req := &pb.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	authReq := &models.AuthRequest{
		Username: req.Username,
		Password: req.Password,
	}

	authResp := &models.AuthResponse{
		UserId: "user123",
		Token:  "jwt.token.here",
	}

	mockAuth.On("Login", mock.Anything, authReq).Return(authResp, nil)

	resp, err := service.Login(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, authResp.UserId, resp.UserId)
	assert.Equal(t, authResp.Token, resp.Token)
	mockAuth.AssertExpectations(t)
}

func TestService_StoreData(t *testing.T) {
	t.Run("Успешное сохранение данных", func(t *testing.T) {
		mockAuth := &mocks.AuthServiceInterface{}
		mockStorage := &mocks.Storage{}
		service := NewService(mockStorage, mockAuth)

		ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

		req := &pb.StoreRequest{
			Data: &pb.DataRecord{
				Name:          "Test Data",
				Type:          pb.DataType_TEXT_DATA,
				EncryptedData: []byte("encrypted"),
			},
		}

		mockStorage.On("StoreData", mock.Anything, "user123", mock.AnythingOfType("*v1.DataRecord")).
			Return(nil)

		resp, err := service.StoreData(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Id)
		assert.Equal(t, int64(1), resp.Version)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Сохранение с существующим ID", func(t *testing.T) {
		mockAuth := &mocks.AuthServiceInterface{}
		mockStorage := &mocks.Storage{}
		service := NewService(mockStorage, mockAuth)

		ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

		req := &pb.StoreRequest{
			Data: &pb.DataRecord{
				Id:            "existing-id",
				Name:          "Test Data",
				Type:          pb.DataType_TEXT_DATA,
				EncryptedData: []byte("encrypted"),
				Version:       1,
			},
		}

		mockStorage.On("StoreData", mock.Anything, "user123", mock.AnythingOfType("*v1.DataRecord")).
			Return(nil)

		resp, err := service.StoreData(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "existing-id", resp.Id)
		assert.Equal(t, int64(2), resp.Version)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Конфликт версий", func(t *testing.T) {
		mockAuth := &mocks.AuthServiceInterface{}
		mockStorage := &mocks.Storage{}
		service := NewService(mockStorage, mockAuth)

		ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

		req := &pb.StoreRequest{
			Data: &pb.DataRecord{
				Id:      "conflict-id",
				Version: 2,
			},
		}

		// Используем более точное сопоставление аргументов
		mockStorage.On("StoreData", mock.Anything, "user123", mock.MatchedBy(func(data *pb.DataRecord) bool {
			return data.Id == "conflict-id" && data.Version == 3
		})).Return(errors2.ErrVersionConflict)

		resp, err := service.StoreData(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		mockStorage.AssertExpectations(t)
	})

	t.Run("Отсутствие аутентификации", func(t *testing.T) {
		mockAuth := &mocks.AuthServiceInterface{}
		mockStorage := &mocks.Storage{}
		service := NewService(mockStorage, mockAuth)

		req := &pb.StoreRequest{
			Data: &pb.DataRecord{},
		}

		emptyCtx := context.Background()
		resp, err := service.StoreData(emptyCtx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestService_RetrieveData(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

	t.Run("Успешное получение данных", func(t *testing.T) {
		req := &pb.RetrieveRequest{
			Id: "data123",
		}

		expectedData := &pb.DataRecord{
			Id:   "data123",
			Name: "Test Data",
			Type: pb.DataType_LOGIN_PASSWORD,
		}

		mockStorage.On("RetrieveData", mock.Anything, "user123", "data123").
			Return(expectedData, nil)

		resp, err := service.RetrieveData(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, resp.Data)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Данные не найдены", func(t *testing.T) {
		req := &pb.RetrieveRequest{
			Id: "nonexistent",
		}

		mockStorage.On("RetrieveData", mock.Anything, "user123", "nonexistent").
			Return((*pb.DataRecord)(nil), errors2.ErrNotFound)

		resp, err := service.RetrieveData(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockStorage.AssertExpectations(t)
	})

	t.Run("Данные удалены", func(t *testing.T) {
		req := &pb.RetrieveRequest{
			Id: "deleted",
		}

		mockStorage.On("RetrieveData", mock.Anything, "user123", "deleted").
			Return((*pb.DataRecord)(nil), errors2.ErrDeleted)

		resp, err := service.RetrieveData(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockStorage.AssertExpectations(t)
	})
}

func TestService_DeleteData(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

	t.Run("Успешное удаление", func(t *testing.T) {
		req := &pb.DeleteRequest{
			Id: "data123",
		}

		mockStorage.On("DeleteData", mock.Anything, "user123", "data123").
			Return(nil)

		resp, err := service.DeleteData(ctx, req)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Данные не найдены для удаления", func(t *testing.T) {
		req := &pb.DeleteRequest{
			Id: "nonexistent",
		}

		mockStorage.On("DeleteData", mock.Anything, "user123", "nonexistent").
			Return(errors2.ErrNotFound)

		resp, err := service.DeleteData(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockStorage.AssertExpectations(t)
	})
}

func TestService_ListData(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

	t.Run("Список данных с фильтром", func(t *testing.T) {
		req := &pb.ListRequest{
			FilterType: pb.DataType_LOGIN_PASSWORD,
		}

		expectedItems := []*pb.DataRecord{
			{Id: "1", Name: "Login 1", Type: pb.DataType_LOGIN_PASSWORD},
			{Id: "2", Name: "Login 2", Type: pb.DataType_LOGIN_PASSWORD},
		}

		mockStorage.On("ListData", mock.Anything, "user123", pb.DataType_LOGIN_PASSWORD).
			Return(expectedItems, nil)

		resp, err := service.ListData(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, expectedItems, resp.Items)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Список данных без фильтра", func(t *testing.T) {
		req := &pb.ListRequest{
			FilterType: -1, // noFilter
		}

		expectedItems := []*pb.DataRecord{
			{Id: "1", Name: "Item 1", Type: pb.DataType_LOGIN_PASSWORD},
			{Id: "2", Name: "Item 2", Type: pb.DataType_BANK_CARD},
		}

		mockStorage.On("ListData", mock.Anything, "user123", pb.DataType(-1)).
			Return(expectedItems, nil)

		resp, err := service.ListData(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		mockStorage.AssertExpectations(t)
	})
}

func TestService_Sync(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	ctx := context.WithValue(context.Background(), middleware.UserIDKey{}, "user123")

	t.Run("Успешная синхронизация", func(t *testing.T) {
		req := &pb.SyncRequest{
			LocalChanges: []*pb.DataRecord{
				{Id: "local1", Name: "Local Data", Version: 1},
			},
			LastSync: time.Now().Add(-1 * time.Hour).Unix(),
		}

		serverData := []*pb.DataRecord{
			{Id: "server1", Name: "Server Data", Version: 2},
		}

		mockStorage.On("ProcessSync", mock.Anything, "user123", req.LocalChanges, req.LastSync).
			Return(serverData, nil)

		resp, err := service.Sync(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, resp.ServerData, 1)
		assert.Equal(t, "Server Data", resp.ServerData[0].Name)
		assert.NotZero(t, resp.CurrentTime)
		mockStorage.AssertExpectations(t)
	})
}

func TestService_Ping(t *testing.T) {
	mockAuth := &mocks.AuthServiceInterface{}
	mockStorage := &mocks.Storage{}
	service := NewService(mockStorage, mockAuth)

	t.Run("Успешный ping", func(t *testing.T) {
		req := &pb.PingRequest{
			Message: "test",
		}

		mockStorage.On("Ping", mock.Anything).Return(nil)

		resp, err := service.Ping(context.Background(), req)
		assert.NoError(t, err)
		assert.Contains(t, resp.Message, "pong: test")
		assert.NotZero(t, resp.Timestamp)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Ошибка ping базы данных", func(t *testing.T) {
		// Очищаем моки перед следующим тестом
		mockStorage.ExpectedCalls = nil
		mockStorage.Calls = nil

		req := &pb.PingRequest{
			Message: "test",
		}

		mockStorage.On("Ping", mock.Anything).Return(errors.New("db error"))

		resp, err := service.Ping(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
		mockStorage.AssertExpectations(t)
	})
}
