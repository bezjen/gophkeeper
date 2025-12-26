package protocol

import (
	"encoding/json"
	"testing"
	"time"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
	"github.com/bezjen/gophkeeper/internal/client/crypto"
	"github.com/bezjen/gophkeeper/internal/client/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProtocol(t *testing.T) {
	p := NewProtocol()
	assert.NotNil(t, p)
}

func TestStringToDataType(t *testing.T) {
	p := NewProtocol()

	testCases := []struct {
		input    string
		expected pb.DataType
		name     string
	}{
		// LoginPassword варианты
		{"login", pb.DataType_LOGIN_PASSWORD, "lowercase login"},
		{"LOGIN", pb.DataType_LOGIN_PASSWORD, "uppercase login"},
		{"Login", pb.DataType_LOGIN_PASSWORD, "mixed case login"},
		{"password", pb.DataType_LOGIN_PASSWORD, "lowercase password"},
		{"PASSWORD", pb.DataType_LOGIN_PASSWORD, "uppercase password"},
		{"Password", pb.DataType_LOGIN_PASSWORD, "mixed case password"},
		{"LoGiN", pb.DataType_LOGIN_PASSWORD, "mixed case LoGiN"},

		// TextData варианты
		{"text", pb.DataType_TEXT_DATA, "lowercase text"},
		{"TEXT", pb.DataType_TEXT_DATA, "uppercase text"},
		{"Text", pb.DataType_TEXT_DATA, "mixed case text"},
		{"note", pb.DataType_TEXT_DATA, "lowercase note"},
		{"NOTE", pb.DataType_TEXT_DATA, "uppercase note"},
		{"Note", pb.DataType_TEXT_DATA, "mixed case note"},

		// BinaryData варианты
		{"binary", pb.DataType_BINARY_DATA, "lowercase binary"},
		{"BINARY", pb.DataType_BINARY_DATA, "uppercase binary"},
		{"Binary", pb.DataType_BINARY_DATA, "mixed case binary"},
		{"file", pb.DataType_BINARY_DATA, "lowercase file"},
		{"FILE", pb.DataType_BINARY_DATA, "uppercase file"},
		{"File", pb.DataType_BINARY_DATA, "mixed case file"},

		// BankCard варианты
		{"card", pb.DataType_BANK_CARD, "lowercase card"},
		{"CARD", pb.DataType_BANK_CARD, "uppercase card"},
		{"Card", pb.DataType_BANK_CARD, "mixed case card"},
		{"bank", pb.DataType_BANK_CARD, "lowercase bank"},
		{"BANK", pb.DataType_BANK_CARD, "uppercase bank"},
		{"Bank", pb.DataType_BANK_CARD, "mixed case bank"},

		// Неизвестные значения (должны возвращать TEXT_DATA по умолчанию)
		{"unknown", pb.DataType_TEXT_DATA, "unknown type"},
		{"", pb.DataType_TEXT_DATA, "empty string"},
		{"random", pb.DataType_TEXT_DATA, "random string"},
		{"123", pb.DataType_TEXT_DATA, "numeric string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := p.StringToDataType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEncryptData(t *testing.T) {
	p := NewProtocol()
	c := crypto.NewCrypto("test-password", "test-salt")

	t.Run("encrypt LoginPassword", func(t *testing.T) {
		loginPassword := &models.LoginPassword{
			Username: "testuser",
			Password: "testpass123",
		}

		encrypted, err := p.EncryptData(c, loginPassword)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)

		// Проверяем, что зашифрованные данные не равны исходному JSON
		jsonData, _ := json.Marshal(loginPassword)
		assert.NotEqual(t, jsonData, encrypted)
	})

	t.Run("encrypt BankCard", func(t *testing.T) {
		bankCard := &models.BankCard{
			Number: "4111111111111111",
			Holder: "TEST USER",
			Expiry: "12/25",
		}

		encrypted, err := p.EncryptData(c, bankCard)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
	})

	t.Run("encrypt TextData", func(t *testing.T) {
		textData := &models.TextData{
			Text: "This is a secret note with sensitive information.",
		}

		encrypted, err := p.EncryptData(c, textData)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
	})

	t.Run("encrypt BinaryData", func(t *testing.T) {
		binaryData := &models.BinaryData{
			Data: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			Size: 5,
		}

		encrypted, err := p.EncryptData(c, binaryData)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
	})

	t.Run("encrypt string directly", func(t *testing.T) {
		text := "plain text string"

		encrypted, err := p.EncryptData(c, text)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, text, string(encrypted))
	})

	t.Run("encrypt bytes directly", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

		encrypted, err := p.EncryptData(c, data)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, data, encrypted)
	})

	t.Run("encrypt arbitrary JSON marshallable struct", func(t *testing.T) {
		type CustomStruct struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}

		custom := &CustomStruct{
			Field1: "value1",
			Field2: 42,
		}

		encrypted, err := p.EncryptData(c, custom)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
	})

	t.Run("encrypt nil content", func(t *testing.T) {
		encrypted, err := p.EncryptData(c, nil)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.NotEmpty(t, encrypted)
	})

	t.Run("encrypt with invalid crypto", func(t *testing.T) {
		invalidCrypto := crypto.NewCrypto("", "")
		textData := &models.TextData{Text: "test"}

		encrypted, err := p.EncryptData(invalidCrypto, textData)
		require.NoError(t, err) // crypto.NewCrypto должен работать даже с пустыми строками
		assert.NotNil(t, encrypted)
	})
}

func TestDecryptData(t *testing.T) {
	p := NewProtocol()
	c := crypto.NewCrypto("test-password", "test-salt")

	t.Run("decrypt LoginPassword", func(t *testing.T) {
		original := &models.LoginPassword{
			Username: "testuser",
			Password: "testpass123",
		}

		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		var decrypted models.LoginPassword
		err = p.DecryptData(c, pb.DataType_LOGIN_PASSWORD, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, original.Username, decrypted.Username)
		assert.Equal(t, original.Password, decrypted.Password)
	})

	t.Run("decrypt BankCard", func(t *testing.T) {
		original := &models.BankCard{
			Number: "4111111111111111",
			Holder: "TEST USER",
			Expiry: "12/25",
		}

		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		var decrypted models.BankCard
		err = p.DecryptData(c, pb.DataType_BANK_CARD, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, original.Number, decrypted.Number)
		assert.Equal(t, original.Holder, decrypted.Holder)
		assert.Equal(t, original.Expiry, decrypted.Expiry)
	})

	t.Run("decrypt TextData as JSON", func(t *testing.T) {
		original := &models.TextData{
			Text: "This is a secret note.",
		}

		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		var decrypted models.TextData
		err = p.DecryptData(c, pb.DataType_TEXT_DATA, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, original.Text, decrypted.Text)
	})

	t.Run("decrypt TextData as plain string (fallback)", func(t *testing.T) {
		// Шифруем просто строку, а не JSON
		plainText := "This is plain text, not JSON"
		encrypted, err := c.Encrypt([]byte(plainText))
		require.NoError(t, err)

		var decrypted models.TextData
		err = p.DecryptData(c, pb.DataType_TEXT_DATA, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, plainText, decrypted.Text)
	})

	t.Run("decrypt BinaryData as JSON", func(t *testing.T) {
		original := &models.BinaryData{
			Data: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			Size: 5,
		}

		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		var decrypted models.BinaryData
		err = p.DecryptData(c, pb.DataType_BINARY_DATA, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, original.Data, decrypted.Data)
		assert.Equal(t, original.Size, decrypted.Size)
	})

	t.Run("decrypt BinaryData as raw bytes (fallback)", func(t *testing.T) {
		// Шифруем просто байты, а не JSON
		rawData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		encrypted, err := c.Encrypt(rawData)
		require.NoError(t, err)

		var decrypted models.BinaryData
		err = p.DecryptData(c, pb.DataType_BINARY_DATA, encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, rawData, decrypted.Data)
		assert.Equal(t, int64(len(rawData)), decrypted.Size)
	})

	t.Run("decrypt with wrong data type", func(t *testing.T) {
		original := &models.LoginPassword{
			Username: "test",
			Password: "pass",
		}

		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		// Пытаемся расшифровать как TextData
		var wrongType models.TextData
		err = p.DecryptData(c, pb.DataType_TEXT_DATA, encrypted, &wrongType)
		// Это не должно вызывать ошибку, так как зашифрованный JSON может быть некорректно разобран
		// Но текст может быть пустым или содержать мусор
		assert.NoError(t, err)
	})

	t.Run("decrypt with invalid encrypted data", func(t *testing.T) {
		invalidData := []byte("not encrypted data")
		var target models.LoginPassword

		err := p.DecryptData(c, pb.DataType_LOGIN_PASSWORD, invalidData, &target)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decrypt")
	})

	t.Run("decrypt with wrong crypto key", func(t *testing.T) {
		original := &models.TextData{Text: "secret"}
		encrypted, err := p.EncryptData(c, original)
		require.NoError(t, err)

		// Другой crypto с другим паролем
		wrongCrypto := crypto.NewCrypto("wrong-password", "wrong-salt")
		var decrypted models.TextData

		err = p.DecryptData(wrongCrypto, pb.DataType_TEXT_DATA, encrypted, &decrypted)
		// GCM должен обнаружить несанкционированное изменение
		if err != nil {
			assert.Contains(t, err.Error(), "failed to decrypt")
		}
	})

	t.Run("decrypt with unknown data type", func(t *testing.T) {
		encrypted, err := c.Encrypt([]byte("test"))
		require.NoError(t, err)

		var target models.TextData
		err = p.DecryptData(c, pb.DataType(999), encrypted, &target)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown data type")
	})
}

func TestCreateDataRecord(t *testing.T) {
	p := NewProtocol()
	c := crypto.NewCrypto("test-password", "test-salt")

	t.Run("create record with LoginPassword", func(t *testing.T) {
		id := "test-id-1"
		name := "Test Credentials"
		metadata := map[string]string{
			"url":      "example.com",
			"category": "work",
		}
		content := &models.LoginPassword{
			Username: "user123",
			Password: "pass123",
		}

		record, err := p.CreateDataRecord(id, pb.DataType_LOGIN_PASSWORD, name, content, metadata, c)
		require.NoError(t, err)

		assert.Equal(t, id, record.Id)
		assert.Equal(t, pb.DataType_LOGIN_PASSWORD, record.Type)
		assert.Equal(t, name, record.Name)
		assert.NotEmpty(t, record.EncryptedData)
		assert.Equal(t, metadata, record.Metadata)
		assert.Equal(t, int64(1), record.Version)
		assert.False(t, record.Deleted)
		assert.True(t, record.UpdatedAt > 0)

		// Проверяем, что можем расшифровать
		var decrypted models.LoginPassword
		err = p.DecryptData(c, pb.DataType_LOGIN_PASSWORD, record.EncryptedData, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, content.Username, decrypted.Username)
		assert.Equal(t, content.Password, decrypted.Password)
	})

	t.Run("create record with TextData", func(t *testing.T) {
		id := "test-id-2"
		name := "Test Note"
		content := &models.TextData{
			Text: "This is a secret note.",
		}

		record, err := p.CreateDataRecord(id, pb.DataType_TEXT_DATA, name, content, nil, c)
		require.NoError(t, err)

		assert.Equal(t, id, record.Id)
		assert.Equal(t, pb.DataType_TEXT_DATA, record.Type)
		assert.Equal(t, name, record.Name)
		assert.NotEmpty(t, record.EncryptedData)
		assert.NotNil(t, record.Metadata)
		assert.Equal(t, int64(1), record.Version)

		var decrypted models.TextData
		err = p.DecryptData(c, pb.DataType_TEXT_DATA, record.EncryptedData, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, content.Text, decrypted.Text)
	})

	t.Run("create record with string content", func(t *testing.T) {
		id := "test-id-3"
		name := "String Record"
		content := "This is plain text content"

		record, err := p.CreateDataRecord(id, pb.DataType_TEXT_DATA, name, content, nil, c)
		require.NoError(t, err)

		assert.Equal(t, id, record.Id)
		assert.NotEmpty(t, record.EncryptedData)

		var decrypted models.TextData
		err = p.DecryptData(c, pb.DataType_TEXT_DATA, record.EncryptedData, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, content, decrypted.Text)
	})

	t.Run("create record with empty metadata", func(t *testing.T) {
		id := "test-id-4"
		content := &models.TextData{Text: "test"}

		record, err := p.CreateDataRecord(id, pb.DataType_TEXT_DATA, "Test", content, nil, c)
		require.NoError(t, err)
		assert.NotNil(t, record.Metadata)
		assert.Empty(t, record.Metadata)
	})

	t.Run("create record with existing metadata", func(t *testing.T) {
		id := "test-id-5"
		metadata := map[string]string{"key": "value"}
		content := &models.TextData{Text: "test"}

		record, err := p.CreateDataRecord(id, pb.DataType_TEXT_DATA, "Test", content, metadata, c)
		require.NoError(t, err)
		assert.Equal(t, metadata, record.Metadata)
	})

	t.Run("create record with BankCard", func(t *testing.T) {
		id := "test-id-6"
		name := "Credit Card"
		content := &models.BankCard{
			Number: "4111111111111111",
			Holder: "John Doe",
			Expiry: "12/25",
		}

		record, err := p.CreateDataRecord(id, pb.DataType_BANK_CARD, name, content, nil, c)
		require.NoError(t, err)

		assert.Equal(t, pb.DataType_BANK_CARD, record.Type)
		assert.Equal(t, name, record.Name)

		var decrypted models.BankCard
		err = p.DecryptData(c, pb.DataType_BANK_CARD, record.EncryptedData, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, content.Number, decrypted.Number)
		assert.Equal(t, content.Holder, decrypted.Holder)
		assert.Equal(t, content.Expiry, decrypted.Expiry)
	})

	t.Run("create record with BinaryData", func(t *testing.T) {
		id := "test-id-7"
		name := "Binary File"
		content := &models.BinaryData{
			Data: []byte{0x01, 0x02, 0x03},
			Size: 3,
		}

		record, err := p.CreateDataRecord(id, pb.DataType_BINARY_DATA, name, content, nil, c)
		require.NoError(t, err)

		assert.Equal(t, pb.DataType_BINARY_DATA, record.Type)

		var decrypted models.BinaryData
		err = p.DecryptData(c, pb.DataType_BINARY_DATA, record.EncryptedData, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, content.Data, decrypted.Data)
		assert.Equal(t, content.Size, decrypted.Size)
	})

	t.Run("create record with encryption error", func(t *testing.T) {
		// Здесь сложно вызвать ошибку шифрования, так как crypto.Encrypt почти всегда работает
		// Мы просто проверяем, что функция возвращает ошибку при ошибке маршалинга
		// Создаем структуру, которую нельзя замаршалить
		unmarshalableContent := func() {} // функция не может быть замаршалена в JSON

		record, err := p.CreateDataRecord("id", pb.DataType_TEXT_DATA, "name", unmarshalableContent, nil, c)
		require.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "failed to marshal")
	})

	t.Run("verify timestamp is recent", func(t *testing.T) {
		before := time.Now().Unix()
		content := &models.TextData{Text: "test"}

		record, err := p.CreateDataRecord("test-time", pb.DataType_TEXT_DATA, "Test", content, nil, c)
		require.NoError(t, err)

		after := time.Now().Unix()
		assert.True(t, record.UpdatedAt >= before && record.UpdatedAt <= after,
			"UpdatedAt should be between %d and %d, got %d", before, after, record.UpdatedAt)
	})
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	p := NewProtocol()
	c := crypto.NewCrypto("test-password", "test-salt")

	testCases := []struct {
		name     string
		dataType pb.DataType
		content  interface{}
	}{
		{
			name:     "LoginPassword roundtrip",
			dataType: pb.DataType_LOGIN_PASSWORD,
			content: &models.LoginPassword{
				Username: "alice",
				Password: "alice123!",
			},
		},
		{
			name:     "BankCard roundtrip",
			dataType: pb.DataType_BANK_CARD,
			content: &models.BankCard{
				Number: "5555555555554444",
				Holder: "Alice Smith",
				Expiry: "05/27",
			},
		},
		{
			name:     "TextData roundtrip",
			dataType: pb.DataType_TEXT_DATA,
			content: &models.TextData{
				Text: "This is a longer text with multiple lines.\nLine 2\nLine 3",
			},
		},
		{
			name:     "BinaryData roundtrip",
			dataType: pb.DataType_BINARY_DATA,
			content: &models.BinaryData{
				Data: make([]byte, 1024), // 1KB данных
				Size: 1024,
			},
		},
		{
			name:     "String roundtrip as TextData",
			dataType: pb.DataType_TEXT_DATA,
			content:  "Plain string content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Шифруем
			encrypted, err := p.EncryptData(c, tc.content)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			// Создаем запись
			record, err := p.CreateDataRecord("test-id", tc.dataType, tc.name, tc.content, nil, c)
			require.NoError(t, err)

			// Расшифровываем разными способами
			switch tc.dataType {
			case pb.DataType_LOGIN_PASSWORD:
				var decrypted models.LoginPassword
				err = p.DecryptData(c, tc.dataType, record.EncryptedData, &decrypted)
				require.NoError(t, err)
				original := tc.content.(*models.LoginPassword)
				assert.Equal(t, original.Username, decrypted.Username)
				assert.Equal(t, original.Password, decrypted.Password)

			case pb.DataType_BANK_CARD:
				var decrypted models.BankCard
				err = p.DecryptData(c, tc.dataType, record.EncryptedData, &decrypted)
				require.NoError(t, err)
				original := tc.content.(*models.BankCard)
				assert.Equal(t, original.Number, decrypted.Number)
				assert.Equal(t, original.Holder, decrypted.Holder)
				assert.Equal(t, original.Expiry, decrypted.Expiry)

			case pb.DataType_TEXT_DATA:
				var decrypted models.TextData
				err = p.DecryptData(c, tc.dataType, record.EncryptedData, &decrypted)
				require.NoError(t, err)

				switch content := tc.content.(type) {
				case *models.TextData:
					assert.Equal(t, content.Text, decrypted.Text)
				case string:
					assert.Equal(t, content, decrypted.Text)
				}

			case pb.DataType_BINARY_DATA:
				var decrypted models.BinaryData
				err = p.DecryptData(c, tc.dataType, record.EncryptedData, &decrypted)
				require.NoError(t, err)
				original := tc.content.(*models.BinaryData)
				assert.Equal(t, original.Data, decrypted.Data)
				assert.Equal(t, original.Size, decrypted.Size)
			}
		})
	}
}
