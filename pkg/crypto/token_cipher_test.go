package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCipher_Roundtrip(t *testing.T) {
	cipher, err := NewTokenCipher("dev-secret-not-for-production-use")
	require.NoError(t, err)

	plain := []byte("ya29.a0AfH6SMB-test-access-token")
	enc, err := cipher.Encrypt(plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, enc)

	dec, err := cipher.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, plain, dec)
}

func TestTokenCipher_HexKey(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	cipher, err := NewTokenCipher(hexKey)
	require.NoError(t, err)

	enc, err := cipher.Encrypt([]byte("refresh-token"))
	require.NoError(t, err)
	dec, err := cipher.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, []byte("refresh-token"), dec)
}

func TestTokenCipher_Raw32ByteSecret(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef" // 32 bytes
	cipher, err := NewTokenCipher(secret)
	require.NoError(t, err)

	enc, err := cipher.Encrypt([]byte("x"))
	require.NoError(t, err)
	dec, err := cipher.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, []byte("x"), dec)
}

func TestTokenCipher_EmptySecret(t *testing.T) {
	_, err := NewTokenCipher("")
	assert.Error(t, err)
}

func TestTokenCipher_DecryptTooShort(t *testing.T) {
	cipher, err := NewTokenCipher("another-secret")
	require.NoError(t, err)
	_, err = cipher.Decrypt([]byte("short"))
	assert.Error(t, err)
}
