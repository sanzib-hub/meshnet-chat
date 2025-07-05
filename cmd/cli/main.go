package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/config"
)

func main() {
	var port = flag.Int("port", 0, "Port to listen on (0 for random)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *port != 0 {
		cfg.P2P.Port = *port
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := p2p.NewNode(ctx, cfg.P2P.Port)

	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	defer node.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go runCLI(node)

	<-sigChan
	log.Println("Received shutdown signal")
	cancel()

}

func runCLI(node *p2p.Node) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== P2P Messenger CLI ===")
	fmt.Println("Commands:")
	fmt.Println("  peers          - List connected peers")
	fmt.Println("  send <id> <msg> - Send message to peer")
	fmt.Println("  help           - Show this help")
	fmt.Println("  quit           - Exit")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "peers":
			listPeers(node)
		case "send":
			if len(parts) < 3 {
				fmt.Println("Usage: send <peer_index> <message>")
				continue
			}
			sendMessage(node, parts[1], strings.Join(parts[2:], " "))
		case "help":
			showHelp()
		case "quit", "exit":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

func showHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  peers           - List all connected peers with indices")
	fmt.Println("  send <idx> <msg> - Send message to peer by index")
	fmt.Println("  help            - Show this help message")
	fmt.Println("  quit/exit       - Exit the application")
	fmt.Println()
}

func sendMessage(node *p2p.Node, s1, s2 string) {
	peers := node.GetConnectedPeers()

	if len(peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	peerIndex, err := strconv.Atoi(s1)
	if err != nil || peerIndex < 0 || peerIndex >= len(peers) {
		fmt.Printf("Invalid peer index. Use 0-%d\n", len(peers)-1)
		return
	}

	peerID := peers[peerIndex]
	if err := node.SendMessage(peerID, s2); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}
	fmt.Printf("Message sent to peer [%d]\n", peerIndex)

}

func listPeers(node *p2p.Node) {
	peers := node.GetConnectedPeers()

	if len(peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	fmt.Printf("Connected peers (%d):\n", len(peers))
	for i, peerID := range peers {
		fmt.Printf("  [%d] %s\n", i, peerID.String())
	}
}
