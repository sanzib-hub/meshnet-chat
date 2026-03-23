package message

import (
	"time"

	"github.com/google/uuid"
)


type MessageType int

const (
	TextMessage MessageType = iota
	MediaMessage
	SyetemMessage
)

type Message struct {
	ID         string            `json:"id"`
	SenderID   string            `json:"sender_id"`
	ReceiverID string            `json:"receiver_id"`
	Type       MessageType       `json:"type"`
	Content    string            `json:"content"`
	Encrypted  bool              `json:"encrypted,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	RoomID     string            `json:"room_id,omitempty"`
	MetaData   map[string]string `json:"metadata,omitempty"`
}

func NewMessage(senderID, receiverID, content string) *Message {
	return &Message{
		ID:         uuid.New().String(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       TextMessage,
		Content:    content,
		Timestamp:  time.Now(),
		MetaData:   make(map[string]string),
	}
}

// Clone returns a shallow copy of the message with a duplicated MetaData map.
func (m *Message) Clone() *Message {
	c := *m
	c.MetaData = make(map[string]string, len(m.MetaData))
	for k, v := range m.MetaData {
		c.MetaData[k] = v
	}
	return &c
}

func NewSystemMessage(content string) *Message {
	return &Message{
		ID: uuid.New().String(),
		Type: SyetemMessage,
		Content: content,
		Timestamp: time.Now(),
		MetaData: make(map[string]string),
	}
}