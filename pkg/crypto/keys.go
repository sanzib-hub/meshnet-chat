package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

const keyFileName = "identity.key"

// LoadOrGenerateKey loads a persistent private key from dataDir, or generates
// and saves a new one if none exists. This gives the node a stable peer ID
// across restarts.
func LoadOrGenerateKey(dataDir string) (libp2pcrypto.PrivKey, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	keyPath := filepath.Join(dataDir, keyFileName)

	if data, err := os.ReadFile(keyPath); err == nil {
		return decodeKey(data)
	}

	// Generate a new Ed25519 key pair.
	priv, _, err := libp2pcrypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	if err := saveKey(keyPath, priv); err != nil {
		return nil, err
	}

	return priv, nil
}

func saveKey(path string, priv libp2pcrypto.PrivKey) error {
	raw, err := libp2pcrypto.MarshalPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	encoded := []byte(base64.StdEncoding.EncodeToString(raw))
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

func decodeKey(data []byte) (libp2pcrypto.PrivKey, error) {
	raw, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode key file: %w", err)
	}

	priv, err := libp2pcrypto.UnmarshalPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	return priv, nil
}
