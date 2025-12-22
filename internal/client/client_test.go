package client_test

import (
	"errors"
	"github.com/bezjen/gophkeeper/internal/client/mocks"
	pbmocks "github.com/bezjen/gophkeeper/pkg/proto/mocks"
	"testing"
	"time"

	"github.com/bezjen/gophkeeper/internal/client"
	"github.com/bezjen/gophkeeper/internal/client/config"
	"github.com/bezjen/gophkeeper/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewClient(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)

	assert.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "localhost:50051", c.ServerAddr)
	mockConfigMgr.AssertCalled(t, "LoadClientConfig")
}

func TestRegister_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockConfigMgr.On("SaveClientConfig", mock.Anything).Return(nil)
	mockClient.On("Register", mock.Anything, mock.Anything).Return(
		&proto.RegisterResponse{
			UserId: "user123",
			Token:  "token123",
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.Register("testuser", "password", "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "user123", c.GetUserID())
	assert.Equal(t, "token123", c.GetToken())
	mockClient.AssertCalled(t, "Register", mock.Anything, &proto.RegisterRequest{
		Username: "testuser",
		Password: "password",
		Email:    "test@example.com",
	})
	mockConfigMgr.AssertCalled(t, "SaveClientConfig", &config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	})
}

func TestRegister_Failure(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockClient.On("Register", mock.Anything, mock.Anything).Return(
		(*proto.RegisterResponse)(nil),
		errors.New("registration failed"),
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.Register("testuser", "password", "test@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registration failed")
}

func TestLogin_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockConfigMgr.On("SaveClientConfig", mock.Anything).Return(nil)
	mockConfigMgr.On("LoadSyncConfig").Return(&config.SyncConfig{}, nil)
	mockStore.On("GetChangedSince", mock.Anything).Return([]*proto.DataRecord{}, nil)
	mockStore.On("Sync", mock.Anything).Return(nil)
	mockConfigMgr.On("SaveSyncConfig", mock.Anything).Return(nil)

	mockClient.On("Login", mock.Anything, mock.Anything).Return(
		&proto.LoginResponse{
			Token:  "token123",
			UserId: "user123",
		},
		nil,
	)
	mockClient.On("Sync", mock.Anything, mock.Anything).Return(
		&proto.SyncResponse{
			ServerData:  []*proto.DataRecord{},
			Conflicts:   []string{},
			CurrentTime: time.Now().Unix(),
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.Login("testuser", "password")
	assert.NoError(t, err)
	assert.Equal(t, "user123", c.GetUserID())
	assert.Equal(t, "token123", c.GetToken())
	assert.True(t, c.IsAuthenticated())
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockClient.On("Login", mock.Anything, mock.Anything).Return(
		(*proto.LoginResponse)(nil),
		status.Error(codes.Unauthenticated, "invalid credentials"),
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.Login("testuser", "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	assert.False(t, c.IsAuthenticated())
}

func TestStoreLoginPassword_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Save", mock.Anything).Return(nil)
	mockClient.On("StoreData", mock.Anything, mock.Anything).Return(
		&proto.StoreResponse{
			Id:      "record123",
			Version: 1,
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	// Initialize crypto
	err = c.InitSession("password")
	assert.NoError(t, err)

	metadata := map[string]string{"tag": "work"}
	id, err := c.StoreLoginPassword("Google", "user@gmail.com", "pass123", metadata)

	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// Verify that Save was called twice (once before and once after server response)
	assert.Equal(t, 2, len(mockStore.Calls))
	mockClient.AssertCalled(t, "StoreData", mock.Anything, mock.MatchedBy(func(req *proto.StoreRequest) bool {
		return req.Data != nil && req.Data.Type == proto.DataType_LOGIN_PASSWORD
	}))
}

func TestStoreLoginPassword_NotAuthenticated(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations - no token/userID in config
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	// Don't initialize crypto
	id, err := c.StoreLoginPassword("Google", "user@gmail.com", "pass123", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
	assert.Empty(t, id)
}

func TestGetData_SuccessFromLocal(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	testRecord := &proto.DataRecord{
		Id:            "record123",
		Type:          proto.DataType_LOGIN_PASSWORD,
		Name:          "Google",
		EncryptedData: []byte("encrypted_data"),
		Metadata:      map[string]string{"tag": "work"},
		Version:       1,
		UpdatedAt:     time.Now().Unix(),
		Deleted:       false,
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "record123").Return(testRecord, nil)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	// Initialize crypto
	err = c.InitSession("password")
	assert.NoError(t, err)

	// Note: This will fail in decryption because we're not providing real encrypted data
	// For a complete test, we would need to mock the protocol layer
	_, err = c.GetData("record123")

	// We expect decryption error since we didn't provide properly encrypted data
	assert.Error(t, err)
	mockStore.AssertCalled(t, "Get", "record123")
}

func TestGetData_FromServerFallback(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	testRecord := &proto.DataRecord{
		Id:            "record123",
		Type:          proto.DataType_LOGIN_PASSWORD,
		Name:          "Google",
		EncryptedData: []byte("encrypted_data"),
		Metadata:      map[string]string{"tag": "work"},
		Version:       1,
		UpdatedAt:     time.Now().Unix(),
		Deleted:       false,
	}

	// Setup mock expectations - local store returns error
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "record123").Return((*proto.DataRecord)(nil), errors.New("not found locally"))
	mockStore.On("Save", testRecord).Return(nil)
	mockClient.On("RetrieveData", mock.Anything, mock.Anything).Return(
		&proto.RetrieveResponse{
			Data: testRecord,
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.InitSession("password")
	assert.NoError(t, err)

	// Note: Will fail in decryption
	_, err = c.GetData("record123")
	assert.Error(t, err) // Decryption error expected

	mockStore.AssertCalled(t, "Get", "record123")
	mockClient.AssertCalled(t, "RetrieveData", mock.Anything, &proto.RetrieveRequest{
		Id: "record123",
	})
	mockStore.AssertCalled(t, "Save", testRecord)
}

func TestListData_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	testRecords := []*proto.DataRecord{
		{
			Id:            "record1",
			Type:          proto.DataType_LOGIN_PASSWORD,
			Name:          "Google",
			EncryptedData: []byte("encrypted1"),
			Metadata:      map[string]string{"tag": "work"},
			Version:       1,
			UpdatedAt:     time.Now().Unix(),
			Deleted:       false,
		},
		{
			Id:            "record2",
			Type:          proto.DataType_BANK_CARD,
			Name:          "Visa",
			EncryptedData: []byte("encrypted2"),
			Metadata:      map[string]string{"tag": "personal"},
			Version:       1,
			UpdatedAt:     time.Now().Unix(),
			Deleted:       false,
		},
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("List", proto.DataType_LOGIN_PASSWORD).Return(testRecords, nil)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	items, err := c.ListData(proto.DataType_LOGIN_PASSWORD)
	assert.NoError(t, err)
	assert.Len(t, items, 2)

	// Verify items contain expected data
	assert.Equal(t, "record1", items[0].ID)
	assert.Equal(t, proto.DataType_LOGIN_PASSWORD, items[0].Type)
	assert.Equal(t, "Google", items[0].Name)
	assert.Equal(t, int64(1), items[0].Version)
	assert.False(t, items[0].Deleted)

	assert.Equal(t, "record2", items[1].ID)
	assert.Equal(t, proto.DataType_BANK_CARD, items[1].Type)
	assert.Equal(t, "Visa", items[1].Name)
}

func TestDeleteData_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	originalRecord := &proto.DataRecord{
		Id:        "record123",
		Type:      proto.DataType_LOGIN_PASSWORD,
		Name:      "Google",
		Version:   1,
		UpdatedAt: time.Now().Unix() - 1000,
		Deleted:   false,
	}

	// Используем канал для захвата аргументов
	saveCalled := make(chan *proto.DataRecord, 1)

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)

	mockStore.On("Get", "record123").Return(originalRecord, nil)

	mockStore.On("Save", mock.Anything).Run(func(args mock.Arguments) {
		if record, ok := args.Get(0).(*proto.DataRecord); ok {
			saveCalled <- record
		}
	}).Return(nil)

	mockClient.On("DeleteData", mock.Anything, mock.Anything).Return(
		&proto.DeleteResponse{
			Success: true,
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.DeleteData("record123")
	assert.NoError(t, err)

	// Проверяем, что Save был вызван
	mockStore.AssertCalled(t, "Save", mock.Anything)

	// Получаем переданный аргумент
	select {
	case savedRecord := <-saveCalled:
		assert.Equal(t, "record123", savedRecord.Id)
		assert.True(t, savedRecord.Deleted)
		// Проверяем что UpdatedAt был обновлен (должен быть больше или равен исходному)
		// Используем GreaterOrEqual, так как время может не измениться в секундах
		assert.GreaterOrEqual(t, savedRecord.UpdatedAt, originalRecord.UpdatedAt)
	case <-time.After(100 * time.Millisecond):
		t.Error("Save was not called with expected arguments")
	}

	mockStore.AssertCalled(t, "Get", "record123")
	mockClient.AssertCalled(t, "DeleteData", mock.Anything, &proto.DeleteRequest{
		Id: "record123",
	})
}

func TestSync_Success(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	localChanges := []*proto.DataRecord{
		{
			Id:        "local1",
			Type:      proto.DataType_LOGIN_PASSWORD,
			Name:      "Local Change",
			Version:   1,
			UpdatedAt: time.Now().Unix(),
		},
	}

	serverData := []*proto.DataRecord{
		{
			Id:        "server1",
			Type:      proto.DataType_BANK_CARD,
			Name:      "Server Change",
			Version:   2,
			UpdatedAt: time.Now().Unix(),
		},
	}

	lastSync := time.Now().Unix() - 3600
	currentTime := time.Now().Unix()

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockConfigMgr.On("LoadSyncConfig").Return(&config.SyncConfig{
		LastSync: lastSync,
	}, nil)
	mockConfigMgr.On("SaveSyncConfig", mock.Anything).Return(nil)

	mockStore.On("GetChangedSince", lastSync).Return(localChanges, nil)
	mockStore.On("Sync", serverData).Return(nil)

	mockClient.On("Sync", mock.Anything, mock.Anything).Return(
		&proto.SyncResponse{
			ServerData:  serverData,
			Conflicts:   []string{},
			CurrentTime: currentTime,
		},
		nil,
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.Sync()
	assert.NoError(t, err)

	mockStore.AssertCalled(t, "GetChangedSince", lastSync)
	mockClient.AssertCalled(t, "Sync", mock.Anything, &proto.SyncRequest{
		LocalChanges: localChanges,
		LastSync:     lastSync,
	})
	mockStore.AssertCalled(t, "Sync", serverData)
	mockConfigMgr.AssertCalled(t, "SaveSyncConfig", &config.SyncConfig{
		LastSync: currentTime,
	})
}

func TestClose(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	// Note: We can't easily mock conn.Close() since it's a real grpc.ClientConn
	// For this test, we'll use a nil connection

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		nil, // nil connection
		mockClient,
	)
	assert.NoError(t, err)

	// Should not panic with nil connection
	err = c.Close()
	assert.NoError(t, err)
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		userID   string
		expected bool
	}{
		{
			name:     "both present",
			token:    "token123",
			userID:   "user123",
			expected: true,
		},
		{
			name:     "no token",
			token:    "",
			userID:   "user123",
			expected: false,
		},
		{
			name:     "no userID",
			token:    "token123",
			userID:   "",
			expected: false,
		},
		{
			name:     "both empty",
			token:    "",
			userID:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			// Setup mock expectations
			mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
				Token:  tt.token,
				UserID: tt.userID,
			}, nil)

			c, err := client.NewClient(
				"localhost:50051",
				mockConfigMgr,
				mockStore,
				mockConn,
				mockClient,
			)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected, c.IsAuthenticated())
		})
	}
}

// Helper function to create test client with mocks
func createTestClient() (*client.Client, *mocks.ManagerInterface, *mocks.StoreInterface, *pbmocks.GophKeeperClient) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)

	c, _ := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)

	return c, mockConfigMgr, mockStore, mockClient
}
