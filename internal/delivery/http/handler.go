package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	peersvc "github.com/yourorg/p2p-messenger/internal/service/peer"
	msgsvc "github.com/yourorg/p2p-messenger/internal/service/messaging"
)

// Handler exposes REST endpoints for peers and message history.
type Handler struct {
	peers     *peersvc.Service
	messaging *msgsvc.Service
}

func NewHandler(peers *peersvc.Service, messaging *msgsvc.Service) *Handler {
	return &Handler{peers: peers, messaging: messaging}
}

// RegisterRoutes mounts all REST routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/peers", h.listPeers)
	mux.HandleFunc("/api/messages", h.sendMessage)
	mux.HandleFunc("/api/messages/", h.getHistory) // /api/messages/{peerID}
	mux.HandleFunc("/api/conversations", h.listConversations)
}

// GET /api/peers
func (h *Handler) listPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, h.peers.ListAll())
}

// POST /api/messages  body: {"peer_id":"...", "content":"..."}
func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID  string `json:"peer_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.PeerID == "" || req.Content == "" {
		http.Error(w, "peer_id and content are required", http.StatusBadRequest)
		return
	}

	msg, err := h.messaging.Send(req.PeerID, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, msg)
}

// GET /api/messages/{peerID}?limit=50
func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peerID := r.URL.Path[len("/api/messages/"):]
	if peerID == "" {
		http.Error(w, "peer ID required", http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	msgs, err := h.messaging.GetHistory(peerID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, msgs)
}

// GET /api/conversations
func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	convs, err := h.messaging.GetConversations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, convs)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
