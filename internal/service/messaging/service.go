package messaging

import (
	"encoding/base64"
	"fmt"
	"log"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
	"github.com/yourorg/p2p-messenger/internal/utils"
)

// Service handles sending and receiving messages, backed by local storage.
//
// Transport security: libp2p's Noise protocol provides E2E encryption on the
// wire — messages cannot be read by a third party in transit.
//
// At-rest encryption: call EnableEncryption to additionally encrypt message
// content before writing to disk. Each node derives an independent storage key
// from its private key + the peer ID, so stored files are unreadable without
// the local private key.
type Service struct {
	node      *p2p.Node
	storage   *local.MessageStorage
	onReceive func(*message.Message)
	privKey   libp2pcrypto.PrivKey // nil = no at-rest encryption
}

func New(node *p2p.Node, storage *local.MessageStorage) *Service {
	svc := &Service{node: node, storage: storage}
	node.SetMessageHandler(svc.handleIncoming)
	return svc
}

// EnableEncryption turns on AES-256-GCM at-rest encryption for stored messages.
// privKey must be the same key used by the local P2P node.
func (s *Service) EnableEncryption(privKey libp2pcrypto.PrivKey) {
	s.privKey = privKey
}

// SetOnReceive registers a callback invoked for every inbound message.
func (s *Service) SetOnReceive(fn func(*message.Message)) {
	s.onReceive = fn
}

// Send creates, stores, and delivers a message to peerID.
func (s *Service) Send(peerIDStr, content string) (*message.Message, error) {
	peerID, err := libp2ppeer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	msg := message.NewMessage(s.node.ID().String(), peerIDStr, content)

	// Encrypt content for local storage (wire copy stays plaintext;
	// Noise already handles wire-level E2E encryption).
	stored := msg.Clone()
	if s.privKey != nil {
		if err := s.encryptContent(stored, peerIDStr); err != nil {
			log.Printf("messaging: encrypt failed, storing plaintext: %v", err)
		}
	}
	if err := s.storage.StoreMessage(stored); err != nil {
		log.Printf("messaging: failed to store outbound message: %v", err)
	}

	if err := s.node.SendMessage(peerID, msg); err != nil {
		return nil, fmt.Errorf("messaging: send failed: %w", err)
	}

	return msg, nil
}

// GetHistory returns up to limit messages for the conversation with peerID.
// If encryption is enabled, message content is decrypted before returning.
func (s *Service) GetHistory(peerID string, limit int) ([]*message.Message, error) {
	msgs, err := s.storage.GetMessages(s.node.ID().String(), peerID, limit)
	if err != nil {
		return nil, err
	}
	if s.privKey != nil {
		for _, m := range msgs {
			if m.Encrypted {
				if err := s.decryptContent(m, peerID); err != nil {
					log.Printf("messaging: decrypt failed for message %s: %v", m.ID, err)
				}
			}
		}
	}
	return msgs, nil
}

// GetConversations returns all conversation keys.
func (s *Service) GetConversations() ([]string, error) {
	return s.storage.GetConversations()
}

func (s *Service) handleIncoming(msg *message.Message) {
	stored := msg.Clone()
	if s.privKey != nil {
		if err := s.encryptContent(stored, msg.SenderID); err != nil {
			log.Printf("messaging: encrypt failed, storing plaintext: %v", err)
		}
	}
	if err := s.storage.StoreMessage(stored); err != nil {
		log.Printf("messaging: failed to store inbound message: %v", err)
	}
	if s.onReceive != nil {
		s.onReceive(msg) // deliver original plaintext to listeners
	}
}

// storageKey derives a 32-byte AES key for the conversation with peerID.
// Key = HKDF(localPrivKey.Raw(), "storage:" + peerID)
func (s *Service) storageKey(peerID string) ([]byte, error) {
	raw, err := s.privKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("raw private key: %w", err)
	}
	return utils.DeriveKey(raw, []byte("storage:"+peerID)), nil
}

func (s *Service) encryptContent(msg *message.Message, peerID string) error {
	key, err := s.storageKey(peerID)
	if err != nil {
		return err
	}
	ciphertext, err := utils.EncryptAESGCM(key, []byte(msg.Content))
	if err != nil {
		return err
	}
	msg.Content = base64.StdEncoding.EncodeToString(ciphertext)
	msg.Encrypted = true
	return nil
}

func (s *Service) decryptContent(msg *message.Message, peerID string) error {
	key, err := s.storageKey(peerID)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(msg.Content)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}
	plaintext, err := utils.DecryptAESGCM(key, raw)
	if err != nil {
		return err
	}
	msg.Content = string(plaintext)
	msg.Encrypted = false
	return nil
}
