package identity

import (
	"fmt"
	"path/filepath"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	p2pcrypto "github.com/yourorg/p2p-messenger/pkg/crypto"
)

// Service manages the local node's persistent cryptographic identity.
type Service struct {
	privKey libp2pcrypto.PrivKey
	peerID  peer.ID
}

// New loads (or creates) the node identity from dataDir.
func New(dataDir string) (*Service, error) {
	keyDir := filepath.Join(dataDir, "identity")

	privKey, err := p2pcrypto.LoadOrGenerateKey(keyDir)
	if err != nil {
		return nil, fmt.Errorf("identity service: %w", err)
	}

	pid, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("identity service: derive peer ID: %w", err)
	}

	return &Service{privKey: privKey, peerID: pid}, nil
}

// PrivKey returns the node's private key (used when creating the libp2p host).
func (s *Service) PrivKey() libp2pcrypto.PrivKey { return s.privKey }

// PeerID returns the stable peer ID derived from the private key.
func (s *Service) PeerID() peer.ID { return s.peerID }

// PeerIDString returns the base58-encoded string form of the peer ID.
func (s *Service) PeerIDString() string { return s.peerID.String() }
