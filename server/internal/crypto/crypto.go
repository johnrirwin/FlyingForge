// Package crypto provides encryption/decryption utilities for sensitive data at rest.
// Uses AES-256-GCM for authenticated encryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("encryption key must be exactly 32 bytes for AES-256")
	// ErrCiphertextTooShort is returned when ciphertext is shorter than the nonce
	ErrCiphertextTooShort = errors.New("ciphertext too short")
	// ErrDecryptionFailed is returned when decryption fails (tampered or wrong key)
	ErrDecryptionFailed = errors.New("decryption failed: data may be tampered or wrong key")
)

// Encryptor handles encryption and decryption of sensitive data
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new Encryptor with the given 32-byte key.
// The key should be loaded from a secure source (environment variable, secrets manager).
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext.
// Returns empty string if plaintext is empty (nothing to encrypt).
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Generate a random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 encode for safe storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
// Returns empty string if ciphertext is empty (nothing to decrypt).
func (e *Encryptor) Decrypt(ciphertextB64 string) (string, error) {
	if ciphertextB64 == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// EncryptIfNotEmpty encrypts only if the value is non-empty, otherwise returns empty string
func (e *Encryptor) EncryptIfNotEmpty(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	encrypted, err := e.Encrypt(plaintext)
	if err != nil {
		// In case of encryption failure, return empty to avoid storing plaintext
		return ""
	}
	return encrypted
}

// DecryptIfNotEmpty decrypts only if the value is non-empty, otherwise returns empty string
func (e *Encryptor) DecryptIfNotEmpty(ciphertext string) string {
	if ciphertext == "" {
		return ""
	}
	decrypted, err := e.Decrypt(ciphertext)
	if err != nil {
		// If decryption fails (wrong key, corrupted data), return empty
		return ""
	}
	return decrypted
}
