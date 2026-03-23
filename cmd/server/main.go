package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	wshandler "github.com/yourorg/p2p-messenger/internal/app/websocket"
	"github.com/yourorg/p2p-messenger/internal/config"
	httpdelivery "github.com/yourorg/p2p-messenger/internal/delivery/http"
	"github.com/yourorg/p2p-messenger/internal/middleware"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
	"github.com/yourorg/p2p-messenger/internal/service/identity"
	discsvc "github.com/yourorg/p2p-messenger/internal/service/discovery"
	msgsvc  "github.com/yourorg/p2p-messenger/internal/service/messaging"
	peersvc "github.com/yourorg/p2p-messenger/internal/service/peer"
)

func main() {
	var (
		httpPort = flag.Int("http-port", 0, "HTTP port (0 → HTTP_PORT env or 8080)")
		p2pPort  = flag.Int("p2p-port", 0, "P2P TCP port (0 for random)")
		dataDir  = flag.String("data-dir", defaultDataDir(), "Directory for keys and message storage")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *httpPort != 0 {
		cfg.HTTP.Port = *httpPort
	}
	if *p2pPort != 0 {
		cfg.P2P.Port = *p2pPort
	}

	// --- Identity (stable peer ID across restarts) ---
	idSvc, err := identity.New(*dataDir)
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	log.Printf("Node identity: %s", idSvc.PeerIDString())

	// --- P2P node ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := p2p.NewNode(ctx, cfg.P2P.Port, idSvc.PrivKey())
	if err != nil {
		log.Fatalf("p2p: %v", err)
	}
	defer node.Close()

	// --- Storage ---
	storage, err := local.NewMessageStorage(filepath.Join(*dataDir, "messages"))
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	// --- Services ---
	peerService := peersvc.New()
	discService := discsvc.New(node, peerService)
	discService.Bootstrap()

	msgService := msgsvc.New(node, storage)
	msgService.EnableEncryption(idSvc.PrivKey()) // at-rest AES-256-GCM encryption

	// --- WebSocket hub ---
	hub := wshandler.NewHub(node)
	// Forward incoming P2P messages to WebSocket clients.
	msgService.SetOnReceive(hub.BroadcastMessage)
	go hub.Run()

	// --- HTTP routes ---
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","node_id":%q,"peers":%d}`,
			node.ID().String(), len(node.GetConnectedPeers()))
	})

	restHandler := httpdelivery.NewHandler(peerService, msgService)
	restHandler.RegisterRoutes(mux)

	// Apply middleware (CORS → rate-limit → mux).
	handler := middleware.CORS(middleware.RateLimit(100, time.Minute)(mux))

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Printf("HTTP server listening on %s", addr)
	log.Printf("WebSocket: ws://localhost%s/ws", addr)
	log.Printf("REST API:  http://localhost%s/api/", addr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatalf("http: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down...")
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".p2p-messenger"
	}
	return filepath.Join(home, ".p2p-messenger")
}
