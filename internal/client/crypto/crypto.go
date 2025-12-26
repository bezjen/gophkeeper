// Package crypto provides cryptographic operations for data encryption and decryption in GophKeeper.
// It uses AES-GCM for authenticated encryption and Argon2 for key derivation.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	iterations  = uint32(2)
	memory      = uint32(19 * 1024)
	parallelism = uint8(1)
	keyLength   = uint32(32)
)

// Crypto provides methods for encrypting and decrypting data using a derived key.
type Crypto struct {
	key []byte
}

// NewCrypto creates a new Crypto instance with a key derived from the provided password and salt.
func NewCrypto(password, salt string) *Crypto {
	key := argon2.IDKey([]byte(password), []byte(salt), iterations, memory, parallelism, keyLength)
	return &Crypto{key: key}
}

// Encrypt encrypts plaintext data using AES-GCM encryption.
func (c *Crypto) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext data encrypted with Encrypt method.
func (c *Crypto) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptWithIV encrypts plaintext data using AES-CFB mode with a provided initialization vector.
func (c *Crypto) EncryptWithIV(plaintext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)

	result := make([]byte, len(iv)+len(ciphertext))
	copy(result[:len(iv)], iv)
	copy(result[len(iv):], ciphertext)

	return result, nil
}

// DecryptWithIV decrypts ciphertext data encrypted with EncryptWithIV method.
func (c *Crypto) DecryptWithIV(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// GetKey returns the raw encryption key (use with caution).
func (c *Crypto) GetKey() []byte {
	return c.key
}
