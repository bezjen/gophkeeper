package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestNewAuthInterceptor(t *testing.T) {
	mockAuth := &MockAuthServiceInterface{}
	interceptor := NewAuthInterceptor(mockAuth)

	// Тестовый handler
	var handlerContext context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerContext = ctx
		return "handler response", nil
	}

	tests := []struct {
		name          string
		fullMethod    string
		setupMetadata func() context.Context
		setupMock     func()
		expectedError bool
		expectedCode  codes.Code
		checkContext  bool
	}{
		{
			name:       "Публичный метод - Login",
			fullMethod: "/gophkeeper.proto.GophKeeper/Login",
			setupMetadata: func() context.Context {
				return context.Background()
			},
			setupMock:     func() {},
			expectedError: false,
		},
		{
			name:       "Публичный метод - Register",
			fullMethod: "/gophkeeper.proto.GophKeeper/Register",
			setupMetadata: func() context.Context {
				return context.Background()
			},
			setupMock:     func() {},
			expectedError: false,
		},
		{
			name:       "Публичный метод - Ping",
			fullMethod: "/gophkeeper.proto.GophKeeper/Ping",
			setupMetadata: func() context.Context {
				return context.Background()
			},
			setupMock:     func() {},
			expectedError: false,
		},
		{
			name:       "Защищенный метод - отсутствуют метаданные",
			fullMethod: "/gophkeeper.proto.GophKeeper/StoreData",
			setupMetadata: func() context.Context {
				return context.Background() // Нет метаданных
			},
			setupMock:     func() {},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name:       "Защищенный метод - отсутствует токен",
			fullMethod: "/gophkeeper.proto.GophKeeper/StoreData",
			setupMetadata: func() context.Context {
				md := metadata.New(map[string]string{})
				return metadata.NewIncomingContext(context.Background(), md)
			},
			setupMock:     func() {},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name:       "Защищенный метод - невалидный токен",
			fullMethod: "/gophkeeper.proto.GophKeeper/StoreData",
			setupMetadata: func() context.Context {
				md := metadata.New(map[string]string{
					"authorization": "invalid.token",
				})
				return metadata.NewIncomingContext(context.Background(), md)
			},
			setupMock: func() {
				mockAuth.On("ValidateToken", "invalid.token").
					Return("", ErrInvalidToken)
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name:       "Защищенный метод - валидный токен",
			fullMethod: "/gophkeeper.proto.GophKeeper/StoreData",
			setupMetadata: func() context.Context {
				md := metadata.New(map[string]string{
					"authorization": "valid.token",
				})
				return metadata.NewIncomingContext(context.Background(), md)
			},
			setupMock: func() {
				mockAuth.On("ValidateToken", "valid.token").
					Return("user123", nil)
			},
			expectedError: false,
			checkContext:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем мок и контекст
			mockAuth.ExpectedCalls = nil
			mockAuth.Calls = nil
			handlerContext = nil

			tt.setupMock()

			ctx := tt.setupMetadata()
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.fullMethod,
			}

			resp, err := interceptor(ctx, nil, info, handler)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "handler response", resp)

				// Проверяем контекст, если нужно
				if tt.checkContext && handlerContext != nil {
					userID, err := GetUserIDFromContext(handlerContext)
					assert.NoError(t, err)
					assert.Equal(t, "user123", userID)
				}
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("UserID присутствует в контексте", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey{}, "user123")

		userID, err := GetUserIDFromContext(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "user123", userID)
	})

	t.Run("UserID отсутствует в контексте", func(t *testing.T) {
		ctx := context.Background()

		userID, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, "", userID)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("Неправильный тип UserID в контексте", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey{}, 123) // int вместо string

		userID, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, "", userID)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}
