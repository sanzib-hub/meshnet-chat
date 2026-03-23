package discovery

import (
	"log"

	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	peersvc "github.com/yourorg/p2p-messenger/internal/service/peer"
)

// Service listens for peer connect/disconnect events from the libp2p host
// and keeps the peer registry up to date.
type Service struct {
	node  *p2p.Node
	peers *peersvc.Service
}

func New(node *p2p.Node, peers *peersvc.Service) *Service {
	svc := &Service{node: node, peers: peers}
	node.Network().Notify(svc)
	return svc
}

// Bootstrap marks all currently-connected peers as online (call once on startup).
func (s *Service) Bootstrap() {
	for _, pid := range s.node.GetConnectedPeers() {
		s.peers.OnPeerConnected(pid)
	}
}

// -- network.Notifiee interface --

func (s *Service) Listen(network.Network, ma.Multiaddr)      {}
func (s *Service) ListenClose(network.Network, ma.Multiaddr) {}

func (s *Service) Connected(_ network.Network, conn network.Conn) {
	pid := conn.RemotePeer()
	log.Printf("discovery: peer connected %s", pid)
	s.peers.OnPeerConnected(pid)
}

func (s *Service) Disconnected(_ network.Network, conn network.Conn) {
	pid := conn.RemotePeer()
	log.Printf("discovery: peer disconnected %s", pid)
	s.peers.OnPeerDisconnected(pid)
}

func (s *Service) OpenedStream(network.Network, network.Stream) {}
func (s *Service) ClosedStream(network.Network, network.Stream) {}
