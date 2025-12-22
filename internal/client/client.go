package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bezjen/gophkeeper/internal/client/config"
	"github.com/bezjen/gophkeeper/internal/client/crypto"
	"github.com/bezjen/gophkeeper/internal/client/filestore"
	"github.com/bezjen/gophkeeper/internal/client/models"
	"github.com/bezjen/gophkeeper/internal/client/protocol"
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
	configMgr  config.ManagerInterface
	localStore filestore.StoreInterface
	crypto     *crypto.Crypto
	Token      string
	UserID     string
	ServerAddr string
}

func NewDefaultClient(serverAddr string) (*Client, error) {
	configMgr, err := config.NewDefaultManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	storeDir := configMgr.GetDataDir()
	localStore, err := filestore.NewFileStore(storeDir)
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

	client := pb.NewGophKeeperClient(conn)
	return NewClient(serverAddr, configMgr, localStore, conn, client)
}

func NewClient(
	serverAddr string,
	configMgr config.ManagerInterface,
	store filestore.StoreInterface,
	conn *grpc.ClientConn,
	client pb.GophKeeperClient,
) (*Client, error) {
	c := &Client{
		conn:       conn,
		client:     client,
		configMgr:  configMgr,
		localStore: store,
		ServerAddr: serverAddr,
	}

	c.LoadConfig()

	return c, nil
}

func (c *Client) LoadConfig() error {
	config, err := c.configMgr.LoadClientConfig()
	if err != nil {
		// Если файла конфигурации нет, это нормально (первый запуск)
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load client config: %w", err)
	}

	c.Token = config.Token
	c.UserID = config.UserID
	return nil
}

func (c *Client) InitSession(password string) error {
	if c.UserID == "" {
		return fmt.Errorf("user ID not found in config")
	}
	c.crypto = crypto.NewCrypto(password, c.UserID)
	return nil
}

func (c *Client) IsAuthenticated() bool {
	return c.Token != "" && c.UserID != ""
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

	c.Token = resp.Token
	c.UserID = resp.UserId
	c.crypto = crypto.NewCrypto(password, c.UserID)

	if err := c.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
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

	c.Token = resp.Token
	c.UserID = resp.UserId
	c.crypto = crypto.NewCrypto(password, c.UserID)

	// Save config after successful login
	if err := c.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Perform initial sync
	return c.Sync()
}

func (c *Client) StoreLoginPassword(name, username, password string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := protocol.NewProtocol()
	loginData := &models.LoginPassword{
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

	// Save locally
	if err := c.localStore.Save(record); err != nil {
		return "", fmt.Errorf("failed to save locally: %w", err)
	}

	// Sync to server
	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Data: record,
	})
	if err != nil {
		return "", fmt.Errorf("failed to store on server: %w", err)
	}

	record.Id = resp.Id
	record.Version = resp.Version
	c.localStore.Save(record)

	return record.Id, nil
}

func (c *Client) StoreText(name, text string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := protocol.NewProtocol()
	textData := &models.TextData{
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
		return "", fmt.Errorf("failed to save locally: %w", err)
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Data: record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	c.localStore.Save(record)

	return record.Id, nil
}

func (c *Client) StoreBinary(name string, data []byte, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := protocol.NewProtocol()
	binaryData := &models.BinaryData{
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
		return "", fmt.Errorf("failed to save locally: %w", err)
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Data: record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	c.localStore.Save(record)

	return record.Id, nil
}

func (c *Client) StoreCard(name, number, holder, expiry string, metadata map[string]string) (string, error) {
	if c.crypto == nil {
		return "", fmt.Errorf("not authenticated")
	}

	proto := protocol.NewProtocol()
	cardData := &models.BankCard{
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
		return "", fmt.Errorf("failed to save locally: %w", err)
	}

	ctx := c.authContext()
	resp, err := c.client.StoreData(ctx, &pb.StoreRequest{
		Data: record,
	})
	if err != nil {
		return "", err
	}

	record.Id = resp.Id
	record.Version = resp.Version
	c.localStore.Save(record)

	return record.Id, nil
}

func (c *Client) GetData(id string) (*models.DataItem, error) {
	// Try local storage first
	record, err := c.localStore.Get(id)
	if err != nil {
		// Fallback to server
		ctx := c.authContext()
		resp, err := c.client.RetrieveData(ctx, &pb.RetrieveRequest{
			Id: id,
		})
		if err != nil {
			return nil, fmt.Errorf("data not found: %w", err)
		}
		record = resp.Data
		c.localStore.Save(record)
	}

	if record.Deleted {
		return nil, fmt.Errorf("record was deleted")
	}

	// Decrypt data
	var content interface{}
	proto := protocol.NewProtocol()

	switch record.Type {
	case pb.DataType_LOGIN_PASSWORD:
		var lp models.LoginPassword
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &lp); err != nil {
			return nil, err
		}
		content = lp
	case pb.DataType_BANK_CARD:
		var bc models.BankCard
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &bc); err != nil {
			return nil, err
		}
		content = bc
	case pb.DataType_TEXT_DATA:
		var td models.TextData
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &td); err != nil {
			return nil, err
		}
		content = td
	case pb.DataType_BINARY_DATA:
		var bd models.BinaryData
		if err := proto.DecryptData(c.crypto, record.Type, record.EncryptedData, &bd); err != nil {
			return nil, err
		}
		content = bd
	default:
		return nil, fmt.Errorf("unknown data type: %v", record.Type)
	}

	return &models.DataItem{
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

func (c *Client) ListData(filterType pb.DataType) ([]*models.DataItem, error) {
	records, err := c.localStore.List(filterType)
	if err != nil {
		return nil, err
	}

	var items []*models.DataItem
	for _, record := range records {
		if record.Deleted {
			continue
		}

		items = append(items, &models.DataItem{
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
	// Mark as deleted locally
	record, err := c.localStore.Get(id)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	record.Deleted = true
	record.UpdatedAt = time.Now().Unix()
	c.localStore.Save(record)

	// Delete on server
	ctx := c.authContext()
	_, err = c.client.DeleteData(ctx, &pb.DeleteRequest{
		Id: id,
	})
	return err
}

func (c *Client) Sync() error {
	// Get local changes
	lastSync := c.LoadLastSync()
	localChanges, err := c.localStore.GetChangedSince(lastSync)
	if err != nil {
		return fmt.Errorf("failed to get local changes: %w", err)
	}

	// Send to server
	ctx := c.authContext()
	resp, err := c.client.Sync(ctx, &pb.SyncRequest{
		LocalChanges: localChanges,
		LastSync:     lastSync,
	})
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Apply server changes
	if err := c.localStore.Sync(resp.ServerData); err != nil {
		return fmt.Errorf("failed to apply server changes: %w", err)
	}

	// Update last sync time
	if err := c.SaveLastSync(resp.CurrentTime); err != nil {
		return fmt.Errorf("failed to save sync config: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) authContext() context.Context {
	md := metadata.Pairs("authorization", c.Token)
	return metadata.NewOutgoingContext(context.Background(), md)
}

func (c *Client) SaveConfig() error {
	config := &config.ClientConfig{
		Token:  c.Token,
		UserID: c.UserID,
	}
	return c.configMgr.SaveClientConfig(config)
}

func (c *Client) LoadLastSync() int64 {
	syncConfig, err := c.configMgr.LoadSyncConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		return 0
	}
	return syncConfig.LastSync
}

func (c *Client) SaveLastSync(timestamp int64) error {
	config := &config.SyncConfig{
		LastSync: timestamp,
	}
	return c.configMgr.SaveSyncConfig(config)
}
