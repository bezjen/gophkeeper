// Package protocol provides data serialization and encryption protocol for GophKeeper.
// It handles conversion between data types, encryption/decryption, and record creation.
package protocol

import (
	"encoding/json"
	"fmt"
	"github.com/bezjen/gophkeeper/internal/client/crypto"
	"github.com/bezjen/gophkeeper/internal/client/models"
	"strings"
	"time"

	pb "github.com/bezjen/gophkeeper/pkg/proto"
)

// Protocol provides methods for data type conversion, encryption, and record creation.
type Protocol struct{}

// NewProtocol creates a new Protocol instance.
func NewProtocol() *Protocol {
	return &Protocol{}
}

// StringToDataType converts a string representation to a DataType enum value.
func (p *Protocol) StringToDataType(s string) pb.DataType {
	switch strings.ToLower(s) {
	case "login", "password":
		return pb.DataType_LOGIN_PASSWORD
	case "text", "note":
		return pb.DataType_TEXT_DATA
	case "binary", "file":
		return pb.DataType_BINARY_DATA
	case "card", "bank":
		return pb.DataType_BANK_CARD
	default:
		return pb.DataType_TEXT_DATA
	}
}

// EncryptData encrypts structured data using the provided crypto instance.
func (p *Protocol) EncryptData(crypto *crypto.Crypto, content interface{}) ([]byte, error) {
	var jsonData []byte
	var err error

	switch dt := content.(type) {
	case *models.LoginPassword:
		jsonData, err = json.Marshal(dt)
	case *models.BankCard:
		jsonData, err = json.Marshal(dt)
	case *models.TextData:
		jsonData, err = json.Marshal(dt)
	case *models.BinaryData:
		jsonData, err = json.Marshal(dt)
	case string:
		jsonData = []byte(dt)
	case []byte:
		jsonData = dt
	default:
		jsonData, err = json.Marshal(content)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return crypto.Encrypt(jsonData)
}

// DecryptData decrypts and unmarshals data into the appropriate structure based on type.
func (p *Protocol) DecryptData(crypto *crypto.Crypto, dataType pb.DataType, encrypted []byte, target interface{}) error {
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	switch dataType {
	case pb.DataType_LOGIN_PASSWORD:
		var lp models.LoginPassword
		if err := json.Unmarshal(decrypted, &lp); err != nil {
			return fmt.Errorf("failed to unmarshal login data: %w", err)
		}
		*target.(*models.LoginPassword) = lp
	case pb.DataType_BANK_CARD:
		var bc models.BankCard
		if err := json.Unmarshal(decrypted, &bc); err != nil {
			return fmt.Errorf("failed to unmarshal card data: %w", err)
		}
		*target.(*models.BankCard) = bc
	case pb.DataType_TEXT_DATA:
		var td models.TextData
		if err := json.Unmarshal(decrypted, &td); err != nil {
			td.Text = string(decrypted)
		}
		*target.(*models.TextData) = td
	case pb.DataType_BINARY_DATA:
		var bd models.BinaryData
		if err := json.Unmarshal(decrypted, &bd); err != nil {
			bd.Data = decrypted
			bd.Size = int64(len(decrypted))
		}
		*target.(*models.BinaryData) = bd
	default:
		return fmt.Errorf("unknown data type: %v", dataType)
	}

	return nil
}

// CreateDataRecord creates a new encrypted DataRecord from the provided data.
func (p *Protocol) CreateDataRecord(
	id string,
	dataType pb.DataType,
	name string,
	content interface{},
	metadata map[string]string,
	crypto *crypto.Crypto,
) (*pb.DataRecord, error) {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	encrypted, err := p.EncryptData(crypto, content)
	if err != nil {
		return nil, err
	}

	return &pb.DataRecord{
		Id:            id,
		Type:          dataType,
		Name:          name,
		EncryptedData: encrypted,
		Metadata:      metadata,
		Version:       1,
		UpdatedAt:     time.Now().Unix(),
		Deleted:       false,
	}, nil
}
