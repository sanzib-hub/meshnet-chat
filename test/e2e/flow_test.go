package e2e_test

import (
	"context"
	"os"
	"testing"
	"time"

	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/yourorg/p2p-messenger/internal/app/p2p"
	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
	msgsvc "github.com/yourorg/p2p-messenger/internal/service/messaging"
)

func newTestNode(t *testing.T, ctx context.Context) (*p2p.Node, *msgsvc.Service) {
	t.Helper()
	dir, err := os.MkdirTemp("", "p2p-e2e-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	node, err := p2p.NewNode(ctx, 0, nil) // random port, ephemeral key
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	t.Cleanup(func() { node.Close() })

	storage, err := local.NewMessageStorage(dir)
	if err != nil {
		t.Fatalf("NewMessageStorage: %v", err)
	}

	svc := msgsvc.New(node, storage)
	return node, svc
}

func TestTwoNodeMessageExchange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node1, svc1 := newTestNode(t, ctx)
	node2, svc2 := newTestNode(t, ctx)

	// Connect node1 → node2 directly (mDNS won't fire in unit tests).
	peerInfo2 := libp2ppeer.AddrInfo{ID: node2.ID(), Addrs: node2.Addrs()}
	if err := node1.Connect(ctx, peerInfo2); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// node2 receives messages via svc2.
	received := make(chan *message.Message, 1)
	svc2.SetOnReceive(func(m *message.Message) {
		received <- m
	})

	// node1 sends a message to node2.
	want := "hello from e2e test"
	if _, err := svc1.Send(node2.ID().String(), want); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case msg := <-received:
		if msg.Content != want {
			t.Errorf("got %q, want %q", msg.Content, want)
		}
		if msg.SenderID != node1.ID().String() {
			t.Errorf("sender: got %q, want %q", msg.SenderID, node1.ID().String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestMessageStoredOnBothSides(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node1, svc1 := newTestNode(t, ctx)
	node2, svc2 := newTestNode(t, ctx)

	peerInfo2 := libp2ppeer.AddrInfo{ID: node2.ID(), Addrs: node2.Addrs()}
	if err := node1.Connect(ctx, peerInfo2); err != nil {
		t.Fatalf("connect: %v", err)
	}

	done := make(chan struct{})
	svc2.SetOnReceive(func(_ *message.Message) { close(done) })

	if _, err := svc1.Send(node2.ID().String(), "stored?"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Both sides should have the message in history.
	msgs1, _ := svc1.GetHistory(node2.ID().String(), 10)
	if len(msgs1) != 1 {
		t.Errorf("node1 history: expected 1, got %d", len(msgs1))
	}

	msgs2, _ := svc2.GetHistory(node1.ID().String(), 10)
	if len(msgs2) != 1 {
		t.Errorf("node2 history: expected 1, got %d", len(msgs2))
	}
}
