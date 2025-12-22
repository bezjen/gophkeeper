package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pb "github.com/bezjen/gophkeeper/pkg/proto"
)

type Protocol struct{}

func NewProtocol() *Protocol {
	return &Protocol{}
}

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

func (p *Protocol) EncryptData(crypto *Crypto, content interface{}) ([]byte, error) {
	var jsonData []byte
	var err error

	switch dt := content.(type) {
	case *LoginPassword:
		jsonData, err = json.Marshal(dt)
	case *BankCard:
		jsonData, err = json.Marshal(dt)
	case *TextData:
		jsonData, err = json.Marshal(dt)
	case *BinaryData:
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

func (p *Protocol) DecryptData(crypto *Crypto, dataType pb.DataType, encrypted []byte, target interface{}) error {
	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	switch dataType {
	case pb.DataType_LOGIN_PASSWORD:
		var lp LoginPassword
		if err := json.Unmarshal(decrypted, &lp); err != nil {
			return fmt.Errorf("failed to unmarshal login data: %w", err)
		}
		*target.(*LoginPassword) = lp
	case pb.DataType_BANK_CARD:
		var bc BankCard
		if err := json.Unmarshal(decrypted, &bc); err != nil {
			return fmt.Errorf("failed to unmarshal card data: %w", err)
		}
		*target.(*BankCard) = bc
	case pb.DataType_TEXT_DATA:
		var td TextData
		if err := json.Unmarshal(decrypted, &td); err != nil {
			td.Text = string(decrypted)
		}
		*target.(*TextData) = td
	case pb.DataType_BINARY_DATA:
		var bd BinaryData
		if err := json.Unmarshal(decrypted, &bd); err != nil {
			bd.Data = decrypted
			bd.Size = int64(len(decrypted))
		}
		*target.(*BinaryData) = bd
	default:
		return fmt.Errorf("unknown data type: %v", dataType)
	}

	return nil
}

func (p *Protocol) CreateDataRecord(
	id string,
	dataType pb.DataType,
	name string,
	content interface{},
	metadata map[string]string,
	crypto *Crypto,
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
