
package p2p

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
	protocolpkg "github.com/yourorg/p2p-messenger/pkg/protocol"
)

const (
	ProtocolID  = "/p2pmessenger/1.0.0"
	ServiceName = "p2p-messenger"
)

type MessageHandler func(*message.Message)

type Node struct {
	host.Host
	ctx            context.Context
	cancel         context.CancelFunc
	mdns           mdns.Service
	storage        *local.MessageStorage
	messageHandler MessageHandler
	peers          map[peer.ID]time.Time 
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("Discovered peer: %s", pi.ID.String())
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := n.h.Connect(ctx, pi); err != nil {
		log.Printf("Failed to connect to peer %s: %v", pi.ID.String(), err)
		return
	}
	
	log.Printf("Successfully connected to peer: %s", pi.ID.String())
}

func NewNode(ctx context.Context, port int, dataDir string) (*Node, error) {
	nodeCtx, cancel := context.WithCancel(ctx)
	
	
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Ping(false),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	
	storage, err := local.NewMessageStorage(dataDir)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create message storage: %w", err)
	}

	node := &Node{
		Host:    h,
		ctx:     nodeCtx,
		cancel:  cancel,
		storage: storage,
		peers:   make(map[peer.ID]time.Time),
	}

	
	h.SetStreamHandler(protocol.ID(ProtocolID), node.handleStream)

	
	if err := node.setupDiscovery(); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to setup discovery: %w", err)
	}

	
	go node.monitorPeers()

	log.Printf("Node started with ID: %s", h.ID().String())
	log.Printf("Listening on addresses:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().String())
	}

	return node, nil
}

func (n *Node) setupDiscovery() error {
	disc := mdns.NewMdnsService(n.Host, ServiceName, &discoveryNotifee{h: n.Host})
	
	if err := disc.Start(); err != nil {
		return fmt.Errorf("failed to start mDNS discovery: %w", err)
	}

	n.mdns = disc
	log.Println("mDNS discovery service started")
	return nil
}

func (n *Node) handleStream(s network.Stream) {
	defer s.Close()
	
	log.Printf("Received stream from peer: %s", s.Conn().RemotePeer().String())
	
	
	n.peers[s.Conn().RemotePeer()] = time.Now()
	
	
	data, err := io.ReadAll(s)
	if err != nil {
		log.Printf("Error reading from stream: %v", err)
		return
	}

	
	envelope, err := protocolpkg.DeserializeEnvelope(data)
	if err != nil {
		log.Printf("Error deserializing message: %v", err)
		return
	}

	log.Printf("Received message from %s: %s", envelope.Message.SenderID, envelope.Message.Content)

	
	if err := n.storage.StoreMessage(envelope.Message); err != nil {
		log.Printf("Error storing message: %v", err)
	}

	
	if n.messageHandler != nil {
		n.messageHandler(envelope.Message)
	}
}

func (n *Node) SendMessage(peerID peer.ID, content string) error {
	
	msg := message.NewMessage(n.Host.ID().String(), peerID.String(), content)
	
	
	envelope := protocolpkg.CreateEnvelop(msg)
	
	
	data, err := envelope.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	
	s, err := n.Host.NewStream(n.ctx, peerID, protocol.ID(ProtocolID))
	if err != nil {
		return fmt.Errorf("failed to open stream to peer %s: %w", peerID.String(), err)
	}
	defer s.Close()

	if _, err := s.Write(data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	
	if err := n.storage.StoreMessage(msg); err != nil {
		log.Printf("Error storing sent message: %v", err)
	}

	log.Printf("Sent message to %s: %s", peerID.String(), content)
	return nil
}

func (n *Node) SetMessageHandler(handler MessageHandler) {
	n.messageHandler = handler
}

func (n *Node) GetMessages(peerID string, limit int) ([]*message.Message, error) {
	return n.storage.GetMessages(n.Host.ID().String(), peerID, limit)
}

func (n *Node) GetRecentMessages(limit int) ([]*message.Message, error) {
	return n.storage.GetRecentMessages(limit)
}

func (n *Node) GetConnectedPeers() []protocolpkg.PeerInfo {
	connectedPeers := n.Host.Network().Peers()
	peerInfos := make([]protocolpkg.PeerInfo, len(connectedPeers))
	
	for i, peerID := range connectedPeers {
		lastSeen := time.Now()
		if t, exists := n.peers[peerID]; exists {
			lastSeen = t
		}
		
		peerInfos[i] = protocolpkg.PeerInfo{
			ID:       peerID.String(),
			Name:     peerID.ShortString(), 
			Online:   true,
			LastSeen: lastSeen,
		}
	}
	
	return peerInfos
}

func (n *Node) monitorPeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			
			cutoff := time.Now().Add(-5 * time.Minute)
			for peerID, lastSeen := range n.peers {
				if lastSeen.Before(cutoff) {
					delete(n.peers, peerID)
				}
			}
		}
	}
}

func (n *Node) Close() error {
	log.Println("Shutting down P2P node...")
	
	if n.mdns != nil {
		if err := n.mdns.Close(); err != nil {
			log.Printf("Error closing mDNS service: %v", err)
		}
	}
	
	n.cancel()
	return n.Host.Close()
}