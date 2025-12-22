package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn       *grpc.ClientConn
	client     pb.GophKeeperClient
	token      string
	userID     string
	configDir  string
	localStore *FileStore
	crypto     *Crypto
	ServerAddr string
}

func NewClient(serverAddr string) (*Client, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	storeDir := filepath.Join(configDir, "data")
	localStore, err := NewFileStore(storeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create local storage: %w", err)
	}

	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*10)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		conn:       conn,
		client:     pb.NewGophKeeperClient(conn),
		configDir:  configDir,
		localStore: localStore,
	}, nil
}

func (c *Client) InitSession(password string) error {
	if c.userID == "" {
		return fmt.Errorf("user ID not found in config")
	}
	c.crypto = NewCrypto(password, c.userID)
	return nil
}

func (c *Client) IsAuthenticated() bool {
	return c.token != "" && c.userID != ""
}

func (c *Client) Register(username, password, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.Register(ctx, &pb.RegisterRequest{
		Username: username,
		Password: password,
		Email:    email,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	c.token = resp.Token
	c.userID = resp.UserId
	c.crypto = NewCrypto(password, c.userID)

	return c.saveConfig()
}

func (c *Client) Login(username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.Login(ctx, &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			return fmt.Errorf("invalid credentials")
		}
		return fmt.Errorf("login failed: %v", err)
	}

	c.token = resp.Token
	c.userID = resp.UserId
	c.crypto = NewCrypto(password, c.userID)

	return c.Sync()
}

func (c *Client) StoreLoginPassword(name, username, password string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := NewProtocol()
	loginData := &LoginPassword{
		Username: username,
		Password: password,
	}

	record, err := proto.CreateDataRecord(
		uuid.New().String(),
		pb.DataType_LOGIN_PASSWORD,
		name,
		loginData,
		metadata,
		c.crypto,
	)
	if err != nil {
		return "", err
	}

	if err := c.localStore.Save(record); err != nil {
		return "", fmt.Errorf("failed to save locally: %w", err)
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Token: c.token,
		Data:  record,
	})
	if err != nil {
		return "", fmt.Errorf("failed to store on server: %w", err)
	}

	record.Id = resp.Id
	record.Version = resp.Version
	err = c.localStore.Save(record)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func (c *Client) StoreText(name, text string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := NewProtocol()
	textData := &TextData{
		Text: text,
	}

	record, err := proto.CreateDataRecord(
		uuid.New().String(),
		pb.DataType_TEXT_DATA,
		name,
		textData,
		metadata,
		c.crypto,
	)
	if err != nil {
		return "", err
	}

	if err := c.localStore.Save(record); err != nil {
		return "", err
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Token: c.token,
		Data:  record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	err = c.localStore.Save(record)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func (c *Client) StoreBinary(name string, data []byte, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := NewProtocol()
	binaryData := &BinaryData{
		Data: data,
		Size: int64(len(data)),
	}

	record, err := proto.CreateDataRecord(
		uuid.New().String(),
		pb.DataType_BINARY_DATA,
		name,
		binaryData,
		metadata,
		c.crypto,
	)
	if err != nil {
		return "", err
	}

	if err := c.localStore.Save(record); err != nil {
		return "", err
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Token: c.token,
		Data:  record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	err = c.localStore.Save(record)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func (c *Client) StoreCard(name, number, holder, expiry string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := NewProtocol()
	cardData := &BankCard{
		Number: number,
		Holder: holder,
		Expiry: expiry,
	}

	record, err := proto.CreateDataRecord(
		uuid.New().String(),
		pb.DataType_BANK_CARD,
		name,
		cardData,
		metadata,
		c.crypto,
	)
	if err != nil {
		return "", err
	}

	if err := c.localStore.Save(record); err != nil {
		return "", err
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Token: c.token,
		Data:  record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	err = c.localStore.Save(record)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func (c *Client) GetData(id string) (*DataItem, error) {
	record, err := c.localStore.Get(id)
	if err != nil {
		ctx := c.authContext()
		resp, err := c.client.RetrieveData(ctx, &pb.RetrieveRequest{
			Token: c.token,
			Id:    id,
		})
		if err != nil {
			return nil, fmt.Errorf("data not found: %w", err)
		}
		record = resp.Data
		err = c.localStore.Save(record)
		if err != nil {
			return nil, err
		}
	}

	if record.Deleted {
		return nil, fmt.Errorf("record was deleted")
	}

	var content interface{}
	proto := NewProtocol()

	switch record.Type {
	case pb.DataType_LOGIN_PASSWORD:
		var lp LoginPassword
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &lp); err != nil {
			return nil, err
		}
		content = lp
	case pb.DataType_BANK_CARD:
		var bc BankCard
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &bc); err != nil {
			return nil, err
		}
		content = bc
	case pb.DataType_TEXT_DATA:
		var td TextData
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &td); err != nil {
			return nil, err
		}
		content = td
	case pb.DataType_BINARY_DATA:
		var bd BinaryData
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &bd); err != nil {
			return nil, err
		}
		content = bd
	default:
		return nil, fmt.Errorf("unknown data type: %v", record.Type)
	}

	return &DataItem{
		ID:        record.Id,
		Type:      record.Type,
		Name:      record.Name,
		Content:   content,
		Encrypted: record.EncryptedData,
		Metadata:  record.Metadata,
		Version:   record.Version,
		UpdatedAt: record.UpdatedAt,
		Deleted:   record.Deleted,
	}, nil
}

func (c *Client) ListData(filterType pb.DataType) ([]*DataItem, error) {
	records, err := c.localStore.List(filterType)
	if err != nil {
		return nil, err
	}

	var items []*DataItem
	for _, record := range records {
		if record.Deleted {
			continue
		}

		items = append(items, &DataItem{
			ID:        record.Id,
			Type:      record.Type,
			Name:      record.Name,
			Encrypted: record.EncryptedData,
			Metadata:  record.Metadata,
			Version:   record.Version,
			UpdatedAt: record.UpdatedAt,
			Deleted:   record.Deleted,
		})
	}

	return items, nil
}

func (c *Client) DeleteData(id string) error {
	record, err := c.localStore.Get(id)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	record.Deleted = true
	record.UpdatedAt = time.Now().Unix()
	err = c.localStore.Save(record)
	if err != nil {
		return err
	}

	ctx := c.authContext()
	_, err = c.client.DeleteData(ctx, &pb.DeleteRequest{
		Token: c.token,
		Id:    id,
	})
	return err
}

func (c *Client) Sync() error {
	lastSync, err := c.loadLastSync()
	if err != nil {
		return err
	}
	localChanges, err := c.localStore.GetChangedSince(lastSync)
	if err != nil {
		return fmt.Errorf("failed to get local changes: %w", err)
	}

	ctx := c.authContext()
	resp, err := c.client.Sync(ctx, &pb.SyncRequest{
		Token:        c.token,
		LocalChanges: localChanges,
		LastSync:     lastSync,
	})
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if err := c.localStore.Sync(resp.ServerData); err != nil {
		return fmt.Errorf("failed to apply server changes: %w", err)
	}

	return c.saveLastSync(resp.CurrentTime)
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) authContext() context.Context {
	md := metadata.Pairs("authorization", c.token)
	return metadata.NewOutgoingContext(context.Background(), md)
}

func (c *Client) saveConfig() error {
	config := map[string]string{
		"token":  c.token,
		"userID": c.userID,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	configFile := filepath.Join(c.configDir, "config.json")
	return os.WriteFile(configFile, data, 0600)
}

func (c *Client) loadLastSync() (int64, error) {
	configFile := filepath.Join(c.configDir, "sync.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return 0, err
	}

	var config struct {
		LastSync int64 `json:"last_sync"`
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return 0, err
	}
	return config.LastSync, nil
}

func (c *Client) saveLastSync(timestamp int64) error {
	config := struct {
		LastSync int64 `json:"last_sync"`
	}{
		LastSync: timestamp,
	}

	data, _ := json.Marshal(config)
	configFile := filepath.Join(c.configDir, "sync.json")
	return os.WriteFile(configFile, data, 0600)
}

func getConfigDir() (string, error) {
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
