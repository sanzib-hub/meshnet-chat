// internal/app/websocket/handler.go
package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, implement proper origin checking
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	id       string
	lastSeen time.Time
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	p2pNode    *p2p.Node
	mu         sync.RWMutex
}

func NewHub(p2pNode *p2p.Node) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		p2pNode:    p2pNode,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			log.Printf("Client %s connected", client.id)

			// Send connection status to new client
			h.sendConnectionStatus(client)

			// Send peer list to new client
			h.sendPeerList(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s disconnected", client.id)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) handleP2PMessage(msg *message.Message) {
	h.BroadcastMessage(msg)
}

// BroadcastMessage serializes msg and pushes it to all connected WebSocket clients.
// Safe to call from any goroutine.
func (h *Hub) BroadcastMessage(msg *message.Message) {
	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypeMessage, msg)
	data, err := wsMsg.Serialize()
	if err != nil {
		log.Printf("Error serializing message for WebSocket: %v", err)
		return
	}
	h.broadcast <- data
}

func (h *Hub) sendConnectionStatus(client *Client) {
	peers := h.p2pNode.GetConnectedPeers()
	status := protocol.ConnectionStatus{
		Connected: true,
		PeerCount: len(peers),
		NodeID:    h.p2pNode.ID().String(),
	}

	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypeStatus, status)
	data, err := wsMsg.Serialize()
	if err != nil {
		log.Printf("Error serializing status message: %v", err)
		return
	}

	select {
	case client.send <- data:
	default:
		close(client.send)
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
	}
}

func (h *Hub) sendPeerList(client *Client) {
	peers := h.p2pNode.GetConnectedPeers()
	peerInfos := make([]protocol.PeerInfo, len(peers))
	for i, pid := range peers {
		peerInfos[i] = protocol.PeerInfo{ID: pid.String()}
	}
	peerList := protocol.PeerList{Peers: peerInfos}

	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypePeerList, peerList)
	data, err := wsMsg.Serialize()
	if err != nil {
		log.Printf("Error serializing peer list: %v", err)
		return
	}

	select {
	case client.send <- data:
	default:
		close(client.send)
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
	}
}

func (h *Hub) broadcastPeerList() {
	peers := h.p2pNode.GetConnectedPeers()
	peerInfos := make([]protocol.PeerInfo, len(peers))
	for i, pid := range peers {
		peerInfos[i] = protocol.PeerInfo{ID: pid.String()}
	}
	peerList := protocol.PeerList{Peers: peerInfos}

	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypePeerList, peerList)
	data, err := wsMsg.Serialize()
	if err != nil {
		log.Printf("Error serializing peer list: %v", err)
		return
	}

	h.broadcast <- data
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		hub:      h,
		id:       generateClientID(),
		lastSeen: time.Now(),
	}

	client.hub.register <- client

	// Start goroutines for handling client
	go client.writePump()
	go client.readPump()
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming WebSocket message
		var wsMsg protocol.WebSocketMessage
		if err := json.Unmarshal(messageData, &wsMsg); err != nil {
			log.Printf("Invalid WebSocket message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case protocol.WSMsgTypeMessage, protocol.WSMsgTypeSendMsg:
			// Forward message to P2P network
			var req protocol.SendMessageRequest
			payloadBytes, err := json.Marshal(wsMsg.Payload)
			if err != nil {
				log.Printf("Error marshaling payload: %v", err)
				continue
			}
			if err := json.Unmarshal(payloadBytes, &req); err != nil {
				log.Printf("Invalid send_message payload: %v", err)
				continue
			}
			peerID, err := libp2ppeer.Decode(req.PeerID)
			if err != nil {
				log.Printf("Invalid peer ID %q: %v", req.PeerID, err)
				continue
			}
			msg := message.NewMessage(c.hub.p2pNode.ID().String(), req.PeerID, req.Content)
			if err := c.hub.p2pNode.SendMessage(peerID, msg); err != nil {
				log.Printf("Failed to send P2P message: %v", err)
			}
		case protocol.WSMsgTypeGetPeers:
			c.hub.sendPeerList(c)
		default:
			log.Printf("Unknown WebSocket message type: %s", wsMsg.Type)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Flush any queued messages into the same write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generateClientID() string {
	return fmt.Sprintf("client-%d", rand.Int63())
}
