package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/yourorg/p2p-messenger/internal/domain/message"
)

type MessageStorage struct {
	dataDir string
	mu      sync.RWMutex
	cache   map[string][]*message.Message
}

func NewMessageStorage(dataDir string) (*MessageStorage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	storage := &MessageStorage{
		dataDir: dataDir,
		cache:   make(map[string][]*message.Message),
	}

	if err := storage.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load message from disk: %w", err)
	}

	return storage, nil
}

func (s *MessageStorage) StoreMessage(msg *message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getConversationKey(msg.SenderID, msg.ReceiverID)

	s.cache[key] = append(s.cache[key], msg)

	sort.Slice(s.cache[key], func(i, j int) bool {
		return s.cache[key][i].Timestamp.Before(s.cache[key][j].Timestamp)

	})

	return s.saveToDisk(key)
}

func (s *MessageStorage) GetMessages(peerID1, peerID2 string, limit int) ([]*message.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.getConversationKey(peerID1, peerID2)

	messages := s.cache[key]

	if limit > 0 && len(messages) > limit {
		start := len(messages) - limit
		return messages[start:], nil
	}

	return messages, nil
}

func (s *MessageStorage) GetRecentMessages(limit int) ([]*message.Message, error) {
	s.mu.Lock()
	defer s.mu.RUnlock()

	var allMessages []*message.Message
	for _, messages := range s.cache {
		allMessages = append(allMessages, messages...)
	}

	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp.After(allMessages[j].Timestamp)

	})

	if limit > 0 && len(allMessages) > limit {
		return allMessages[:limit], nil
	}

	return allMessages, nil
}

func (s *MessageStorage) GetConversations() ([]string, error) {
	s.mu.Lock()
	defer s.mu.RUnlock()

	conversations := make([]string, 0, len(s.cache))
	for key := range s.cache {
		conversations = append(conversations, key)
	}
	return conversations, nil
}

func (s *MessageStorage) getConversationKey(peerID1, peerID2 string) string {
	if peerID1 < peerID2 {
		return fmt.Sprintf("%s_%s", peerID1, peerID2)
	}
	return fmt.Sprintf("%s_%s", peerID2, peerID1)
}

func (s *MessageStorage) saveToDisk(conversationKey string) error {
	messages := s.cache[conversationKey]
	if len(messages) == 0 {
		return nil
	}

	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", conversationKey))
	data, err := json.MarshalIndent(messages, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func (s *MessageStorage) loadFromDisk() error {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			conversationKey := file.Name()[:len(file.Name())-5]
			if err := s.loadConversation(conversationKey); err != nil {
				return fmt.Errorf("failed to load conversation %s: %w", conversationKey, err)
			}
		}
	}
	return nil
}

// CleanupOldMessages removes messages older than the specified duration
func (s *MessageStorage) loadConversation(conversationKey string) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", conversationKey))

	data, err := os.ReadFile(filename)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)

	}

	var messages []*message.Message

	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	s.cache[conversationKey] = messages
	return nil
}

func (s *MessageStorage) CleanupOldMessages(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for key, messages := range s.cache {
		var keepMessages []*message.Message
		for _, msg := range messages {
			if msg.Timestamp.After(cutoff) {
				keepMessages = append(keepMessages, msg)
			}
		}

		if len(keepMessages) != len(messages) {
			s.cache[key] = keepMessages
			if err := s.saveToDisk(key); err != nil {
				return fmt.Errorf("failed to save conversation %s after cleanup: %w", key, err)

			}
		}
	}
	return nil
}
