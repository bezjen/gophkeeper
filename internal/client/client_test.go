package client_test

import (
	"errors"
	pbmocks "github.com/bezjen/gophkeeper/api/gophkeeper/v1/mocks"
	"github.com/bezjen/gophkeeper/internal/client/mocks"
	"github.com/bezjen/gophkeeper/internal/client/models"
	"os"
	"testing"
	"time"

	proto "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
	"github.com/bezjen/gophkeeper/internal/client"
	"github.com/bezjen/gophkeeper/internal/client/config"
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
	assert.Equal(t, "user123", c.UserID)
	assert.Equal(t, "token123", c.Token)
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
	assert.Equal(t, "user123", c.UserID)
	assert.Equal(t, "token123", c.Token)
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

	// Setup mock expectations - no Token/UserID in config
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
			name:     "no Token",
			token:    "",
			userID:   "user123",
			expected: false,
		},
		{
			name:     "no UserID",
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

// Test для LoadConfig при разных сценариях
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.ClientConfig
		loadErr     error
		expectToken string
		expectUser  string
	}{
		{
			name:        "config exists",
			config:      &config.ClientConfig{Token: "token1", UserID: "user1"},
			expectToken: "token1",
			expectUser:  "user1",
		},
		{
			name:        "config not exists",
			loadErr:     os.ErrNotExist,
			expectToken: "",
			expectUser:  "",
		},
		{
			name:        "config load error",
			loadErr:     errors.New("load error"),
			expectToken: "",
			expectUser:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			mockConfigMgr.On("LoadClientConfig").Return(tt.config, tt.loadErr)

			c, err := client.NewClient(
				"localhost:50051",
				mockConfigMgr,
				mockStore,
				mockConn,
				mockClient,
			)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectToken, c.Token)
			assert.Equal(t, tt.expectUser, c.UserID)
		})
	}
}

// Test для InitSession
func TestInitSession(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{"with user id", "user123", false},
		{"empty user id", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
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

			err = c.InitSession("password")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c) // Проверяем, что клиент не nil
			}
		})
	}
}

// Test для StoreText
func TestStoreText_Success(t *testing.T) {
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
			Id:      "text123",
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

	id, err := c.StoreText("Note", "This is a secret note", map[string]string{"category": "personal"})
	assert.NoError(t, err)
	assert.Equal(t, "text123", id)
	assert.Equal(t, 2, len(mockStore.Calls))
}

// Test для StoreBinary
func TestStoreBinary_Success(t *testing.T) {
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
			Id:      "binary123",
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

	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	id, err := c.StoreBinary("SecretFile", data, map[string]string{"type": "encrypted"})
	assert.NoError(t, err)
	assert.Equal(t, "binary123", id)
	assert.Equal(t, 2, len(mockStore.Calls))
}

// Test для StoreCard
func TestStoreCard_Success(t *testing.T) {
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
			Id:      "card123",
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

	id, err := c.StoreCard(
		"Visa Card",
		"4111111111111111",
		"John Doe",
		"12/25",
		map[string]string{"bank": "Chase"},
	)
	assert.NoError(t, err)
	assert.Equal(t, "card123", id)
	assert.Equal(t, 2, len(mockStore.Calls))
}

// Test для GetData с различными типами данных
func TestGetData_DifferentTypes(t *testing.T) {
	tests := []struct {
		name     string
		dataType proto.DataType
		content  interface{}
	}{
		{"login password", proto.DataType_LOGIN_PASSWORD, &models.LoginPassword{}},
		{"bank card", proto.DataType_BANK_CARD, &models.BankCard{}},
		{"text data", proto.DataType_TEXT_DATA, &models.TextData{}},
		{"binary data", proto.DataType_BINARY_DATA, &models.BinaryData{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			testRecord := &proto.DataRecord{
				Id:            "test123",
				Type:          tt.dataType,
				Name:          "Test",
				EncryptedData: []byte("encrypted"),
				Metadata:      map[string]string{"test": "true"},
				Version:       1,
				UpdatedAt:     time.Now().Unix(),
				Deleted:       false,
			}

			// Setup mock expectations
			mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
				Token:  "token123",
				UserID: "user123",
			}, nil)
			mockStore.On("Get", "test123").Return(testRecord, nil)

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

			// Note: Will fail in decryption, but we're testing type handling
			item, err := c.GetData("test123")
			assert.Error(t, err) // Decryption error expected
			assert.Nil(t, item)
		})
	}
}

// Test для GetData с удаленной записью
func TestGetData_DeletedRecord(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	testRecord := &proto.DataRecord{
		Id:        "deleted123",
		Type:      proto.DataType_LOGIN_PASSWORD,
		Name:      "Deleted",
		Deleted:   true,
		UpdatedAt: time.Now().Unix(),
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "deleted123").Return(testRecord, nil)

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

	item, err := c.GetData("deleted123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record was deleted")
	assert.Nil(t, item)
}

// Test для ListData без фильтра
func TestListData_NoFilter(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	testRecords := []*proto.DataRecord{
		{
			Id:        "record1",
			Type:      proto.DataType_LOGIN_PASSWORD,
			Name:      "Google",
			Version:   1,
			UpdatedAt: time.Now().Unix(),
			Deleted:   false,
		},
		{
			Id:        "record2",
			Type:      proto.DataType_BANK_CARD,
			Name:      "Visa",
			Version:   1,
			UpdatedAt: time.Now().Unix(),
			Deleted:   false,
		},
		{
			Id:        "record3",
			Type:      proto.DataType_LOGIN_PASSWORD,
			Name:      "GitHub",
			Version:   1,
			UpdatedAt: time.Now().Unix(),
			Deleted:   true, // Должен быть пропущен
		},
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("List", proto.DataType(0)).Return(testRecords, nil)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	items, err := c.ListData(0) // 0 = все типы
	assert.NoError(t, err)
	assert.Len(t, items, 2) // Только 2 не удаленных записи
	assert.Equal(t, "record1", items[0].ID)
	assert.Equal(t, "record2", items[1].ID)
}

// Test для ListData с ошибкой хранилища
func TestListData_StoreError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("List", proto.DataType_LOGIN_PASSWORD).Return(
		([]*proto.DataRecord)(nil), errors.New("store error"),
	)

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	items, err := c.ListData(proto.DataType_LOGIN_PASSWORD)
	assert.Error(t, err)
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), "store error")
}

// Test для DeleteData с ошибкой сервера
func TestDeleteData_ServerError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	originalRecord := &proto.DataRecord{
		Id:        "record123",
		Type:      proto.DataType_LOGIN_PASSWORD,
		Name:      "Google",
		Version:   1,
		UpdatedAt: time.Now().Unix(),
		Deleted:   false,
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "record123").Return(originalRecord, nil)
	mockStore.On("Save", mock.Anything).Return(nil)
	mockClient.On("DeleteData", mock.Anything, mock.Anything).Return(
		(*proto.DeleteResponse)(nil), errors.New("server error"),
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
	// Local store should still have saved the delete mark
	mockStore.AssertCalled(t, "Save", mock.Anything)
}

// Test для Sync с ошибками
func TestSync_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		getChangesErr error
		syncErr       error
		storeSyncErr  error
		wantErr       bool
	}{
		{"get changes error", errors.New("get changes error"), nil, nil, true},
		{"sync server error", nil, errors.New("sync error"), nil, true},
		{"store sync error", nil, nil, errors.New("store sync error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			lastSync := time.Now().Unix() - 3600

			// Setup mock expectations
			mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
				Token:  "token123",
				UserID: "user123",
			}, nil)
			mockConfigMgr.On("LoadSyncConfig").Return(&config.SyncConfig{
				LastSync: lastSync,
			}, nil)

			if tt.getChangesErr != nil {
				mockStore.On("GetChangedSince", lastSync).Return(
					([]*proto.DataRecord)(nil), tt.getChangesErr,
				)
			} else {
				mockStore.On("GetChangedSince", lastSync).Return(
					[]*proto.DataRecord{}, nil,
				)
			}

			if tt.syncErr != nil {
				mockClient.On("Sync", mock.Anything, mock.Anything).Return(
					(*proto.SyncResponse)(nil), tt.syncErr,
				)
			} else if tt.storeSyncErr != nil {
				mockClient.On("Sync", mock.Anything, mock.Anything).Return(
					&proto.SyncResponse{
						ServerData:  []*proto.DataRecord{},
						Conflicts:   []string{},
						CurrentTime: time.Now().Unix(),
					},
					nil,
				)
				mockStore.On("Sync", mock.Anything).Return(tt.storeSyncErr)
			}

			c, err := client.NewClient(
				"localhost:50051",
				mockConfigMgr,
				mockStore,
				mockConn,
				mockClient,
			)
			assert.NoError(t, err)

			err = c.Sync()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test для сохранения конфигурации
func TestSaveConfig(t *testing.T) {
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

	// Set some token and userID
	c.Token = "test-token"
	c.UserID = "test-user"

	// Mock save to return error
	mockConfigMgr.On("SaveClientConfig", mock.Anything).Return(errors.New("save error"))

	// Тестируем через Register, так как saveConfig приватный
	mockClient.On("Register", mock.Anything, mock.Anything).Return(
		&proto.RegisterResponse{
			UserId: "user123",
			Token:  "token123",
		},
		nil,
	)

	err = c.Register("test", "pass", "test@test.com")
	// В методе Register ошибка saveConfig не обрабатывается через fmt.Errorf,
	// поэтому проверяем что ошибка есть, но текст может быть разный
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save")
}

// Test для loadLastSync с ошибкой
func TestLoadLastSync_Error(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	currentTime := time.Now().Unix()

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockConfigMgr.On("LoadSyncConfig").Return((*config.SyncConfig)(nil), errors.New("load error"))
	mockConfigMgr.On("SaveSyncConfig", mock.Anything).Return(nil) // Добавляем мок для SaveSyncConfig

	mockStore.On("GetChangedSince", int64(0)).Return([]*proto.DataRecord{}, nil)
	mockStore.On("Sync", mock.Anything).Return(nil)
	mockClient.On("Sync", mock.Anything, mock.Anything).Return(
		&proto.SyncResponse{
			ServerData:  []*proto.DataRecord{},
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
	assert.NoError(t, err) // Should handle error gracefully and use 0
	mockStore.AssertCalled(t, "GetChangedSince", int64(0))
	mockConfigMgr.AssertCalled(t, "SaveSyncConfig", mock.Anything)
}

// Test для saveLastSync с ошибкой
func TestSaveLastSync_Error(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

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
	mockConfigMgr.On("SaveSyncConfig", mock.Anything).Return(errors.New("save error"))

	mockStore.On("GetChangedSince", lastSync).Return([]*proto.DataRecord{}, nil)
	mockStore.On("Sync", mock.Anything).Return(nil)
	mockClient.On("Sync", mock.Anything, mock.Anything).Return(
		&proto.SyncResponse{
			ServerData:  []*proto.DataRecord{},
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
	assert.Error(t, err)
	// Теперь ошибка должна содержать "failed to save sync config"
	assert.Contains(t, err.Error(), "failed to save sync config")
}

// Test для authContext (косвенно через другие методы)
func TestAuthContext(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "test-Token",
		UserID: "test-user",
	}, nil)
	mockClient.On("RetrieveData", mock.Anything, mock.Anything).Return(
		&proto.RetrieveResponse{
			Data: &proto.DataRecord{
				Id:            "test123",
				EncryptedData: []byte("encrypted"),
				Type:          proto.DataType_LOGIN_PASSWORD,
			},
		},
		nil,
	)
	mockStore.On("Get", "test123").Return((*proto.DataRecord)(nil), errors.New("not found"))
	mockStore.On("Save", mock.Anything).Return(nil)

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

	// GetData will call authContext internally
	_, err = c.GetData("test123")
	assert.Error(t, err) // Decryption error, but auth should work
}

// Test для методов Store с ошибками при сохранении локально
func TestStoreMethods_LocalSaveError(t *testing.T) {
	storeMethods := []struct {
		name   string
		method func(*client.Client) (string, error)
	}{
		{
			name: "StoreLoginPassword",
			method: func(c *client.Client) (string, error) {
				return c.StoreLoginPassword("test", "user", "pass", nil)
			},
		},
		{
			name: "StoreText",
			method: func(c *client.Client) (string, error) {
				return c.StoreText("test", "text", nil)
			},
		},
		{
			name: "StoreBinary",
			method: func(c *client.Client) (string, error) {
				return c.StoreBinary("test", []byte{1, 2, 3}, nil)
			},
		},
		{
			name: "StoreCard",
			method: func(c *client.Client) (string, error) {
				return c.StoreCard("test", "1234", "holder", "12/25", nil)
			},
		},
	}

	for _, sm := range storeMethods {
		t.Run(sm.name, func(t *testing.T) {
			mockConfigMgr := &mocks.ManagerInterface{}
			mockStore := &mocks.StoreInterface{}
			mockConn := &grpc.ClientConn{}
			mockClient := &pbmocks.GophKeeperClient{}

			// Setup mock expectations
			mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
				Token:  "token123",
				UserID: "user123",
			}, nil)

			// Для метода StoreLoginPassword нужно мокировать protocol
			// Для остальных методов тоже, но мы можем просто убедиться что Save будет вызван
			mockStore.On("Save", mock.Anything).Return(errors.New("local save error"))

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

			id, err := sm.method(c)
			assert.Error(t, err)

			// Для StoreLoginPassword и других методов ошибка должна содержать "failed to save locally"
			// Но StoreLoginPassword уже содержит эту обертку, а другие методы теперь тоже должны
			assert.Contains(t, err.Error(), "failed to save locally")
			assert.Empty(t, id)
		})
	}
}

// Test для методов Store с ошибками сервера
func TestStoreMethods_ServerError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Save", mock.Anything).Return(nil).Once()
	mockClient.On("StoreData", mock.Anything, mock.Anything).Return(
		(*proto.StoreResponse)(nil), errors.New("server error"),
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

	id, err := c.StoreLoginPassword("test", "user", "pass", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store on server")
	assert.Empty(t, id)
}

// Test для GetData с неизвестным типом данных
func TestGetData_UnknownType(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Используем неизвестный тип
	testRecord := &proto.DataRecord{
		Id:            "unknown123",
		Type:          proto.DataType(999), // Неизвестный тип
		Name:          "Unknown",
		EncryptedData: []byte("encrypted"),
		Metadata:      map[string]string{},
		Version:       1,
		UpdatedAt:     time.Now().Unix(),
		Deleted:       false,
	}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "unknown123").Return(testRecord, nil)

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

	item, err := c.GetData("unknown123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown data type")
	assert.Nil(t, item)
}

// Test для Login с ошибкой сохранения конфига
func TestLogin_SaveConfigError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockConfigMgr.On("SaveClientConfig", mock.Anything).Return(errors.New("save config error"))
	mockClient.On("Login", mock.Anything, mock.Anything).Return(
		&proto.LoginResponse{
			Token:  "token123",
			UserId: "user123",
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save config")
}

func TestRegister_SaveConfigError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{}, nil)
	mockConfigMgr.On("SaveClientConfig", mock.Anything).Return(errors.New("save config error"))
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save config")
}

// Test для GetData с ошибкой получения с сервера
func TestGetData_ServerError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "missing123").Return((*proto.DataRecord)(nil), errors.New("not found locally"))
	mockClient.On("RetrieveData", mock.Anything, mock.Anything).Return(
		(*proto.RetrieveResponse)(nil), errors.New("server error"),
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

	item, err := c.GetData("missing123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data not found")
	assert.Nil(t, item)
}

// Test для DeleteData с ошибкой при получении записи
func TestDeleteData_GetRecordError(t *testing.T) {
	mockConfigMgr := &mocks.ManagerInterface{}
	mockStore := &mocks.StoreInterface{}
	mockConn := &grpc.ClientConn{}
	mockClient := &pbmocks.GophKeeperClient{}

	// Setup mock expectations
	mockConfigMgr.On("LoadClientConfig").Return(&config.ClientConfig{
		Token:  "token123",
		UserID: "user123",
	}, nil)
	mockStore.On("Get", "missing123").Return((*proto.DataRecord)(nil), errors.New("record not found"))

	c, err := client.NewClient(
		"localhost:50051",
		mockConfigMgr,
		mockStore,
		mockConn,
		mockClient,
	)
	assert.NoError(t, err)

	err = c.DeleteData("missing123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record not found")
}
