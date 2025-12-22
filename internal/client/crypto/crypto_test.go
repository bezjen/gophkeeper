package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"io"
	"testing"
)

const (
	testPassword = "testPassword123"
	testSalt     = "testSalt"
)

func TestNewCrypto(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	if c == nil {
		t.Fatal("NewCrypto returned nil")
	}

	key := c.GetKey()
	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestEncryptDecrypt(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	testData := []byte("This is a test plaintext.")

	// Test encryption
	ciphertext, err := c.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify ciphertext is different from plaintext
	if bytes.Equal(ciphertext, testData) {
		t.Fatal("Ciphertext should not equal plaintext")
	}

	// Verify ciphertext is longer than plaintext (has nonce prepended)
	if len(ciphertext) <= len(testData) {
		t.Fatal("Ciphertext should be longer than plaintext due to nonce")
	}

	// Test decryption
	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, testData) {
		t.Fatal("Decrypted text doesn't match original")
	}
}

func TestEncryptDecrypt_EmptyData(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	emptyData := []byte{}

	ciphertext, err := c.Encrypt(emptyData)
	if err != nil {
		t.Fatalf("Encrypt empty data failed: %v", err)
	}

	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty data failed: %v", err)
	}

	if !bytes.Equal(plaintext, emptyData) {
		t.Fatal("Decrypted empty data doesn't match original")
	}
}

func TestEncryptDecrypt_LargeData(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	// Create 10KB of test data
	largeData := make([]byte, 10*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	ciphertext, err := c.Encrypt(largeData)
	if err != nil {
		t.Fatalf("Encrypt large data failed: %v", err)
	}

	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt large data failed: %v", err)
	}

	if !bytes.Equal(plaintext, largeData) {
		t.Fatal("Decrypted large data doesn't match original")
	}
}

func TestEncryptDecrypt_MultipleMessages(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	messages := [][]byte{
		[]byte("Message 1"),
		[]byte("Message 2"),
		[]byte("Message 3"),
	}

	for i, msg := range messages {
		ciphertext, err := c.Encrypt(msg)
		if err != nil {
			t.Fatalf("Encrypt message %d failed: %v", i+1, err)
		}

		plaintext, err := c.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt message %d failed: %v", i+1, err)
		}

		if !bytes.Equal(plaintext, msg) {
			t.Fatalf("Decrypted message %d doesn't match original", i+1)
		}
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	// Test with ciphertext too short
	shortCiphertext := make([]byte, 5)
	_, err := c.Decrypt(shortCiphertext)
	if err == nil {
		t.Fatal("Expected error for short ciphertext")
	}

	// Test with random bytes
	randomBytes := make([]byte, 100)
	io.ReadFull(rand.Reader, randomBytes)
	_, err = c.Decrypt(randomBytes)
	if err == nil {
		t.Fatal("Expected error for random ciphertext")
	}
}

func TestEncryptWithIV_DecryptWithIV(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	testData := []byte("Test data for IV mode")

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)

	// Test encryption with IV
	ciphertext, err := c.EncryptWithIV(testData, iv)
	if err != nil {
		t.Fatalf("EncryptWithIV failed: %v", err)
	}

	// Verify IV is prepended to ciphertext
	if !bytes.Equal(ciphertext[:aes.BlockSize], iv) {
		t.Fatal("IV should be prepended to ciphertext")
	}

	// Test decryption with IV
	plaintext, err := c.DecryptWithIV(ciphertext)
	if err != nil {
		t.Fatalf("DecryptWithIV failed: %v", err)
	}

	if !bytes.Equal(plaintext, testData) {
		t.Fatal("Decrypted text doesn't match original")
	}
}

func TestEncryptWithIV_DecryptWithIV_EmptyData(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	emptyData := []byte{}
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)

	ciphertext, err := c.EncryptWithIV(emptyData, iv)
	if err != nil {
		t.Fatalf("EncryptWithIV empty data failed: %v", err)
	}

	plaintext, err := c.DecryptWithIV(ciphertext)
	if err != nil {
		t.Fatalf("DecryptWithIV empty data failed: %v", err)
	}

	if !bytes.Equal(plaintext, emptyData) {
		t.Fatal("Decrypted empty data doesn't match original")
	}
}

func TestDecryptWithIV_InvalidCiphertext(t *testing.T) {
	c := NewCrypto(testPassword, testSalt)

	// Test with ciphertext shorter than block size
	shortCiphertext := make([]byte, aes.BlockSize-1)
	_, err := c.DecryptWithIV(shortCiphertext)
	if err == nil {
		t.Fatal("Expected error for ciphertext shorter than block size")
	}

	// Test with exactly block size (only IV, no data)
	ivOnly := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, ivOnly)
	plaintext, err := c.DecryptWithIV(ivOnly)
	if err != nil {
		t.Fatalf("DecryptWithIV with only IV failed: %v", err)
	}

	if len(plaintext) != 0 {
		t.Fatal("Expected empty plaintext when ciphertext contains only IV")
	}
}

func TestDifferentInstancesSameKey(t *testing.T) {
	// Create two instances with same password and salt
	c1 := NewCrypto(testPassword, testSalt)
	c2 := NewCrypto(testPassword, testSalt)

	testData := []byte("Test data")

	// Encrypt with first instance
	ciphertext, err := c1.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt with first instance failed: %v", err)
	}

	// Decrypt with second instance (should work with same key)
	plaintext, err := c2.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt with second instance failed: %v", err)
	}

	if !bytes.Equal(plaintext, testData) {
		t.Fatal("Decrypted text doesn't match original")
	}
}

func TestDifferentInstancesDifferentKey(t *testing.T) {
	// Create two instances with different salts
	c1 := NewCrypto(testPassword, "salt1")
	c2 := NewCrypto(testPassword, "salt2")

	testData := []byte("Test data")

	// Encrypt with first instance
	ciphertext, err := c1.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt with first instance failed: %v", err)
	}

	// Try to decrypt with second instance (should fail or give wrong result)
	plaintext, err := c2.Decrypt(ciphertext)
	// Note: GCM will detect tampering and return an error
	if err == nil {
		// If no error, the plaintext should be different
		if bytes.Equal(plaintext, testData) {
			t.Fatal("Different keys should not decrypt correctly")
		}
	}
}

func TestEncrypt_DeterministicKey(t *testing.T) {
	// Test that same password and salt always produce same key
	c1 := NewCrypto("password", "salt")
	c2 := NewCrypto("password", "salt")

	key1 := c1.GetKey()
	key2 := c2.GetKey()

	if !bytes.Equal(key1, key2) {
		t.Fatal("Same password and salt should produce same key")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	c := NewCrypto(testPassword, testSalt)
	data := make([]byte, 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Encrypt(data)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	c := NewCrypto(testPassword, testSalt)
	data := make([]byte, 1024)
	ciphertext, err := c.Encrypt(data)
	if err != nil {
		b.Fatalf("Setup encrypt failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Decrypt(ciphertext)
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}
	}
}
