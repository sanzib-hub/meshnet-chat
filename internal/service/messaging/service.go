package messaging

import (
	"fmt"
	"log"

	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
)

// Service handles sending and receiving messages, backed by local storage.
type Service struct {
	node    *p2p.Node
	storage *local.MessageStorage
	// onReceive is called when a message arrives from the P2P network.
	onReceive func(*message.Message)
}

func New(node *p2p.Node, storage *local.MessageStorage) *Service {
	svc := &Service{
		node:    node,
		storage: storage,
	}
	node.SetMessageHandler(svc.handleIncoming)
	return svc
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

	if err := s.storage.StoreMessage(msg); err != nil {
		log.Printf("messaging: failed to store outbound message: %v", err)
	}

	if err := s.node.SendMessage(peerID, msg); err != nil {
		return nil, fmt.Errorf("messaging: send failed: %w", err)
	}

	return msg, nil
}

// GetHistory returns up to limit messages for the conversation with peerID.
func (s *Service) GetHistory(peerID string, limit int) ([]*message.Message, error) {
	return s.storage.GetMessages(s.node.ID().String(), peerID, limit)
}

// GetConversations returns all conversation keys.
func (s *Service) GetConversations() ([]string, error) {
	return s.storage.GetConversations()
}

func (s *Service) handleIncoming(msg *message.Message) {
	if err := s.storage.StoreMessage(msg); err != nil {
		log.Printf("messaging: failed to store inbound message: %v", err)
	}
	if s.onReceive != nil {
		s.onReceive(msg)
	}
}
