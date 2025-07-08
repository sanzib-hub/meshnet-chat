package protocol

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourorg/p2p-messenger/internal/domain/message"
)

const (
	ProtocolVersion = "1.0.0"
)

type MessageEnvelope struct{
	ProtocolVersion string `json:"protocol_version"`
	Message *message.Message `json:"message"`
	Signature string `json:"signature,omitempty"`
}

type WebSocketMessageType string

const (
	WSMsgTypeMessage WebSocketMessageType = "message"
	WSMsgTypePeerList WebSocketMessageType = "peer_list"
	WSMsgTypeStatus WebSocketMessageType = "status"
	WSMsgTypeError WebSocketMessageType = "error"
	WSMsgTypeSendMsg WebSocketMessageType = "send_message"
	WSMsgTypeGetPeers WebSocketMessageType = "get_peers"
)

type WebSocketMessage struct{
	Type WebSocketMessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type PeerInfo struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Online bool `json:"online"`
	LastSeen time.Time `json:"last_seen"`
}

type PeerList struct {
	Peers []PeerInfo `json:"peers"`
}

type ConnectionStatus struct{
	Connected bool `json:"connected"`
	PeerCount int `json:"peer_count"`
	NodeID string `json:"node_id"`
}

type ErrorMessage struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

type SendMessageRequest struct {
	PeerID string `json:"peer_id"`
	Content string `json:"content"`
}

func(e *MessageEnvelope) Serialize() ([]byte, error){
	return json.Marshal(e)
} 

func DeserializeEnvelope(data []byte) (*MessageEnvelope, error){
	var envelope MessageEnvelope

	err:= json.Unmarshal(data, &envelope)
	if err != nil{
		return nil,fmt.Errorf("failed to deserialize message envelope: %w", err)
	}
	return &envelope, nil
}

func CreateEnvelop(msg *message.Message) *MessageEnvelope{
	return &MessageEnvelope{
		ProtocolVersion: ProtocolVersion,
		Message: msg,
	}
}

func NewWebSocketMessage(msgType WebSocketMessageType, payload interface{}) *WebSocketMessage{
	return  &WebSocketMessage{
		Type: msgType,
		Payload: payload,
	}
}

func (w *WebSocketMessage) Serialize() ([]byte, error){
	return json.Marshal(w)
}