package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


type Client struct{
	conn *websocket.Conn
	send chan []byte
	hub *Hub
	id string
	lastSeen time.Time
}

type Hub struct{
	clients map[*Client]bool
	broadcast chan []byte
	register chan *Client
	unregister chan *Client
	p2pNode *p2p.Node
	mu sync.RWMutex
}

func NewHub(p2pNode *p2p.Node) *Hub{
	hub :=&Hub{
		clients: make(map[*Client]bool),
		broadcast: make(chan []byte),
		register: make(chan *Client),
		unregister: make(chan *Client),
		p2pNode: p2pNode,
	}
	p2pNode.SetMessageHandler(hub.handleP2PMessage)
	return hub
}

func (h *Hub) Run(){
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			log.Printf("Client %s connected", client.id)

			h.sendConnectionStatus(client)

			h.sendPeerList(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok{
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s disconnected", client.id)
			} 
			h.mu.Unlock()

			case message := <- h.broadcast:
				h.mu.RLock()
				for client := range h.clients{
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
				h.mu.RUnlock()
		}
	}
}

func (h *Hub) handleP2PMessage(msg *message.Message){
	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypeMessage, msg)

	data,err := wsMsg.Serialize()
	if err != nil{
		log.Printf("Error serializing message for WebSocket: %v", err)
		return
	}

	h.broadcast <- data
}

func (h *Hub) sendConnectionStatus(client *Client){
	peers := h.p2pNode.GetConnectedPeers()

	status := protocol.ConnectionStatus{
		Connected: true,
		PeerCount: len(peers),
		NodeID: h.p2pNode.ID().String(),
	}

	wsMsg := protocol.NewWebSocketMessage(protocol.WSMsgTypeStatus, status)
	data , err := wsMsg.Serialize()
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

