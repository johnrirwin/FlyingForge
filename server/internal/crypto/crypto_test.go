package crypto

import (
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"valid 32-byte key", 32, false},
		{"too short key", 16, true},
		{"too long key", 64, true},
		{"empty key", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewEncryptor(key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("this-is-a-32-byte-test-key-12345")
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple phrase", "my-secret-bind-phrase"},
		{"empty string", ""},
		{"unicode", "秘密のフレーズ"},
		{"special chars", "p@$$w0rd!#$%^&*()"},
		{"long phrase", "this-is-a-very-long-bind-phrase-that-someone-might-use-for-their-receiver-settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty plaintext should return empty ciphertext
			if tt.plaintext == "" {
				if ciphertext != "" {
					t.Errorf("Encrypt() of empty string should be empty, got %v", ciphertext)
				}
				return
			}

			// Ciphertext should be different from plaintext
			if ciphertext == tt.plaintext {
				t.Error("Encrypt() ciphertext should not equal plaintext")
			}

			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte("this-is-a-32-byte-test-key-12345")
	key2 := []byte("different-32-byte-test-key-56789")

	enc1, _ := NewEncryptor(key1)
	enc2, _ := NewEncryptor(key2)

	ciphertext, _ := enc1.Encrypt("secret-phrase")

	_, err := enc2.Decrypt(ciphertext)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong key should return ErrDecryptionFailed, got %v", err)
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := []byte("this-is-a-32-byte-test-key-12345")
	enc, _ := NewEncryptor(key)

	plaintext := "same-bind-phrase"

	// Encrypt the same plaintext twice
	ct1, _ := enc.Encrypt(plaintext)
	ct2, _ := enc.Encrypt(plaintext)

	// Due to random nonce, ciphertexts should be different
	if ct1 == ct2 {
		t.Error("Encrypting same plaintext twice should produce different ciphertexts (random nonce)")
	}

	// But both should decrypt to the same plaintext
	pt1, _ := enc.Decrypt(ct1)
	pt2, _ := enc.Decrypt(ct2)

	if pt1 != plaintext || pt2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	key := []byte("this-is-a-32-byte-test-key-12345")
	enc, _ := NewEncryptor(key)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    bool
	}{
		{"empty string", "", false},
		{"invalid base64", "not-valid-base64!!!", true},
		{"too short", "YWJj", true}, // "abc" in base64
		{"tampered", "dGFtcGVyZWQtZGF0YS10aGF0LWlzLWxvbmctZW5vdWdo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptIfNotEmpty(t *testing.T) {
	key := []byte("this-is-a-32-byte-test-key-12345")
	enc, _ := NewEncryptor(key)

	// Empty should return empty
	if got := enc.EncryptIfNotEmpty(""); got != "" {
		t.Errorf("EncryptIfNotEmpty(\"\") = %v, want \"\"", got)
	}

	// Non-empty should return encrypted value
	if got := enc.EncryptIfNotEmpty("secret"); got == "" || got == "secret" {
		t.Errorf("EncryptIfNotEmpty(\"secret\") should return encrypted value, got %v", got)
	}
}

func TestDecryptIfNotEmpty(t *testing.T) {
	key := []byte("this-is-a-32-byte-test-key-12345")
	enc, _ := NewEncryptor(key)

	// Empty should return empty
	if got := enc.DecryptIfNotEmpty(""); got != "" {
		t.Errorf("DecryptIfNotEmpty(\"\") = %v, want \"\"", got)
	}

	// Valid encrypted value should decrypt
	encrypted, _ := enc.Encrypt("secret")
	if got := enc.DecryptIfNotEmpty(encrypted); got != "secret" {
		t.Errorf("DecryptIfNotEmpty() = %v, want \"secret\"", got)
	}

	// Invalid ciphertext should return empty (fail silently)
	if got := enc.DecryptIfNotEmpty("invalid-ciphertext"); got != "" {
		t.Errorf("DecryptIfNotEmpty(invalid) = %v, want \"\"", got)
	}
}
