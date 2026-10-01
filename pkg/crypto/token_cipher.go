package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// TokenCipher encrypts and decrypts OAuth tokens at rest using AES-GCM.
type TokenCipher struct {
	gcm cipher.AEAD
}

// NewTokenCipher derives a 32-byte AES key from secret.
// If secret is a 64-char hex string it is decoded; otherwise SHA-256(secret) is used.
func NewTokenCipher(secret string) (*TokenCipher, error) {
	if secret == "" {
		return nil, fmt.Errorf("token encryption key is empty")
	}

	key, err := deriveKey(secret)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &TokenCipher{gcm: gcm}, nil
}

func deriveKey(secret string) ([]byte, error) {
	if len(secret) == 64 {
		if decoded, err := hex.DecodeString(secret); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(secret) == 32 {
		return []byte(secret), nil
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}

// Encrypt encrypts plaintext and returns nonce||ciphertext.
func (c *TokenCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return c.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts nonce||ciphertext produced by Encrypt.
func (c *TokenCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := c.gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plain, nil
}
