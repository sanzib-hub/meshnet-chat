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
	ID string `json:"id"`
	SenderID string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Type MessageType `json:"type"`
	Content string `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	RoomID string `json:"room_id"`
	MetaData map[string]string `json:"metadata"`
}

func NewMessage(senderID, receiverID, content string) *Message{
	return &Message{
		ID: uuid.New().String(),
		SenderID: senderID,
		ReceiverID: receiverID,
		Type: TextMessage,
		Content: content,
		MetaData: make(map[string]string),
	}
}

func NewSystemMessage(content string) *Message{
	return &Message{
		ID: uuid.New().String(),
		Type: SyetemMessage,
		Content: content,
		Timestamp: time.Now(),
		MetaData: make(map[string]string),
	}
}