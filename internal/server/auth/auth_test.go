package auth

import (
	"context"
	"github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/mocks"
	"github.com/bezjen/gophkeeper/internal/server/models"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(storage *mocks.UserStorage)
		req           *models.AuthRequest
		expectedError bool
	}{
		{
			name: "Успешная регистрация",
			setupMock: func(m *mocks.UserStorage) {
				m.On("GetUserByUsernameOrEmail", mock.Anything, "testuser", "test@example.com").
					Return((*models.User)(nil), nil)
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).
					Return(nil)
			},
			req: &models.AuthRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			expectedError: false,
		},
		{
			name: "Пользователь уже существует",
			setupMock: func(m *mocks.UserStorage) {
				m.On("GetUserByUsernameOrEmail", mock.Anything, "existinguser", "existing@example.com").
					Return(&models.User{ID: "123"}, nil)
			},
			req: &models.AuthRequest{
				Username: "existinguser",
				Password: "password123",
				Email:    "existing@example.com",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mocks.UserStorage{}
			tt.setupMock(mockStorage)

			auth := NewAuthService("test-secret-key", mockStorage)
			resp, err := auth.Register(context.Background(), tt.req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.UserId)
				assert.NotEmpty(t, resp.Token)
				mockStorage.AssertExpectations(t)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	mockStorage := &mocks.UserStorage{}
	auth := NewAuthService("test-secret-key", mockStorage)

	// Создаем пользователя для теста
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	testUser := &models.User{
		ID:       "user123",
		Username: "testuser",
		Password: string(hashedPassword),
		Email:    "test@example.com",
	}

	tests := []struct {
		name          string
		setupMock     func()
		req           *models.AuthRequest
		expectedError bool
	}{
		{
			name: "Успешный логин",
			setupMock: func() {
				mockStorage.On("GetUserByUsername", mock.Anything, "testuser").
					Return(testUser, nil)
			},
			req: &models.AuthRequest{
				Username: "testuser",
				Password: "correctpassword",
			},
			expectedError: false,
		},
		{
			name: "Неверный пароль",
			setupMock: func() {
				mockStorage.On("GetUserByUsername", mock.Anything, "testuser").
					Return(testUser, nil)
			},
			req: &models.AuthRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedError: true,
		},
		{
			name: "Пользователь не найден",
			setupMock: func() {
				mockStorage.On("GetUserByUsername", mock.Anything, "nonexistent").
					Return((*models.User)(nil), errors.ErrNotFound)
			},
			req: &models.AuthRequest{
				Username: "nonexistent",
				Password: "password",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.ExpectedCalls = nil
			tt.setupMock()

			resp, err := auth.Login(context.Background(), tt.req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
				assert.Equal(t, testUser.ID, resp.UserId)
			}
		})
	}
}

func TestAuthService_TokenValidation(t *testing.T) {
	auth := NewAuthService("test-secret-key", nil)

	t.Run("Генерация и валидация токена", func(t *testing.T) {
		userID := "user123"
		username := "testuser"

		token, err := auth.GenerateToken(userID, username)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		validatedUserID, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, validatedUserID)
	})

	t.Run("Невалидный токен", func(t *testing.T) {
		_, err := auth.ValidateToken("invalid.token.here")
		assert.Error(t, err)
		assert.Equal(t, errors.ErrInvalidToken, err)
	})

	t.Run("Истекший токен", func(t *testing.T) {
		// Создаем токен с истекшим сроком
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  "user123",
			"username": "testuser",
			"exp":      time.Now().Add(-1 * time.Hour).Unix(),
			"iat":      time.Now().Add(-2 * time.Hour).Unix(),
		})

		tokenString, _ := token.SignedString([]byte("test-secret-key"))
		_, err := auth.ValidateToken(tokenString)
		assert.Error(t, err)
	})
}
