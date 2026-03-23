package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// RandomHex returns a random hex string of n bytes (2n characters).
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// SHA256Hex returns the hex-encoded SHA-256 hash of data.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// DeriveKey derives a 32-byte key using HKDF (HMAC-SHA256) from secret and info.
// secret is the key material (e.g. private key bytes), info is a context label.
func DeriveKey(secret, info []byte) []byte {
	// HKDF-Extract: PRK = HMAC-SHA256(salt="", IKM=secret)
	mac := hmac.New(sha256.New, []byte("p2p-messenger-v1"))
	mac.Write(secret)
	prk := mac.Sum(nil)

	// HKDF-Expand: OKM = HMAC-SHA256(PRK, info || 0x01)
	mac2 := hmac.New(sha256.New, prk)
	mac2.Write(info)
	mac2.Write([]byte{0x01})
	return mac2.Sum(nil) // 32 bytes = AES-256 key
}

// EncryptAESGCM encrypts plaintext with AES-256-GCM using key.
// The returned ciphertext is: nonce (12 bytes) || encrypted data.
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptAESGCM decrypts ciphertext produced by EncryptAESGCM.
func DecryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}
