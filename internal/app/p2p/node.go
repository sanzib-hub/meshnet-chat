package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pprotocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/pkg/protocol"
)

const (
	// Protocol ID for our messaging service
	ProtocolID = "/p2pmessenger/1.0.0"
	// Service name for mDNS discovery
	ServiceName = "p2p-messenger"
)

type Node struct {
	host.Host
	ctx            context.Context
	cancel         context.CancelFunc
	mdns           mdns.Service
	messageHandler func(msg *message.Message)
}

func (n *Node) SetMessageHandler(handler func(msg *message.Message)) {
	n.messageHandler = handler
}

type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound is called when a new peer is discovered via mDNS
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("Discovery Peer: %s", pi.ID.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := n.h.Connect(ctx, pi); err != nil {
		log.Printf("Failed to connect to peer %s: %v", pi.ID.String(), err)
		return
	}

	log.Printf("Successfully connected to peer: %s", pi.ID.String())
}

// NewNode creates a new P2P node with mDNS discovery.
// privKey is optional: pass nil to generate a fresh (non-persistent) key.
// Use pkg/crypto.LoadOrGenerateKey to obtain a persistent key.
func NewNode(ctx context.Context, port int, privKey libp2pcrypto.PrivKey) (*Node, error) {
	nodeCtx, cancel := context.WithCancel(ctx)

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Ping(false),
	}
	if privKey != nil {
		opts = append(opts, libp2p.Identity(privKey))
	}

	h, err := libp2p.New(opts...)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)

	}
	node := &Node{
		Host:   h,
		ctx:    nodeCtx,
		cancel: cancel,
	}

	// Set up stream handler for incoming messages
	h.SetStreamHandler(libp2pprotocol.ID(ProtocolID), node.handleStream)

	if err := node.setupDiscover(); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to setup discovery: %w", err)
	}

	log.Printf("Node started with ID: %s", h.ID().String())
	log.Printf("Listening on addresses:")

	for _, addr := range h.Addrs() {
		log.Printf(" %s/p2p/%s", addr, h.ID().String())
	}

	return node, nil
}

func (n *Node) setupDiscover() error {
	// Create mDNS discovery service
	disc := mdns.NewMdnsService(n.Host, ServiceName, &discoveryNotifee{h: n.Host})

	//Start the discovery service
	if err := disc.Start(); err != nil {
		return fmt.Errorf("failed to start a mDNS doscovery: %w", err)

	}
	n.mdns = disc
	log.Println("mDNS discovery service started")
	return nil
}

func (n *Node) handleStream(s network.Stream) {
	log.Printf("Received stream from peer: %s", s.Conn().RemotePeer().String())

	defer s.Close()

	envelope, err := protocol.DeserializeEnvelope(s)
	if err != nil {
		log.Printf("Error reading from stream: %v", err)
		return
	}

	if envelope.Message != nil && n.messageHandler != nil {
		n.messageHandler(envelope.Message)
	}
}

// SendMessage sends a message to a peer
func (n *Node) SendMessage(peerID peer.ID, msg *message.Message) error {
	s, err := n.Host.NewStream(n.ctx, peerID, libp2pprotocol.ID(ProtocolID))
	if err != nil {
		return fmt.Errorf("failed to open stream to peer %s: %w", peerID.String(), err)
	}
	defer s.Close()

	envelope := protocol.CreateEnvelop(msg)
	data, err := envelope.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	if err := json.NewEncoder(s).Encode(data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Sent message to %s: %s", peerID.String(), msg.Content)
	return nil
}

func (n *Node) GetConnectedPeers() []peer.ID {
	return n.Host.Network().Peers()
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
