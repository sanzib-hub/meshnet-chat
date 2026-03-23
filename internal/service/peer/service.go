package peer

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	peerdomain "github.com/yourorg/p2p-messenger/internal/domain/peer"
)

// Service tracks known peers and their online status.
type Service struct {
	mu    sync.RWMutex
	peers map[string]*peerdomain.Peer
}

func New() *Service {
	return &Service{
		peers: make(map[string]*peerdomain.Peer),
	}
}

// OnPeerConnected registers or marks a peer online.
func (s *Service) OnPeerConnected(id peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idStr := id.String()
	if p, ok := s.peers[idStr]; ok {
		p.Online = true
		p.LastSeen = time.Now()
	} else {
		s.peers[idStr] = peerdomain.New(idStr)
	}
}

// OnPeerDisconnected marks a peer offline.
func (s *Service) OnPeerDisconnected(id peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p, ok := s.peers[id.String()]; ok {
		p.Online = false
		p.LastSeen = time.Now()
	}
}

// SetName sets a display name for a peer.
func (s *Service) SetName(id peer.ID, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idStr := id.String()
	if _, ok := s.peers[idStr]; !ok {
		s.peers[idStr] = peerdomain.New(idStr)
	}
	s.peers[idStr].Name = name
}

// GetPeer returns the peer record for id, or nil if unknown.
func (s *Service) GetPeer(id peer.ID) *peerdomain.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.peers[id.String()]
	return p
}

// ListOnline returns all currently online peers.
func (s *Service) ListOnline() []*peerdomain.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*peerdomain.Peer, 0)
	for _, p := range s.peers {
		if p.Online {
			result = append(result, p)
		}
	}
	return result
}

// ListAll returns every known peer (online and offline).
func (s *Service) ListAll() []*peerdomain.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*peerdomain.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		result = append(result, p)
	}
	return result
}
