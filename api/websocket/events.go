package websocket

// Event type constants used by the WebSocket protocol.
// These mirror pkg/protocol.WebSocketMessageType and serve as the canonical
// reference for frontend clients.
const (
	// Server → Client
	EventMessage    = "message"     // inbound P2P message
	EventPeerList   = "peer_list"   // current peer list
	EventStatus     = "status"      // node connection status
	EventError      = "error"       // error notification

	// Client → Server
	EventSendMsg  = "send_message" // send a message to a peer
	EventGetPeers = "get_peers"    // request the current peer list
)

// Payloads for Client → Server events.

// SendMessagePayload is the JSON body for EventSendMsg.
type SendMessagePayload struct {
	PeerID  string `json:"peer_id"`
	Content string `json:"content"`
}

// GetPeersPayload has no fields; the client sends an empty object.
type GetPeersPayload struct{}
