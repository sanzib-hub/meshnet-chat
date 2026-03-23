package integration_test

import (
	"os"
	"testing"
	"time"

	"github.com/yourorg/p2p-messenger/internal/domain/message"
	"github.com/yourorg/p2p-messenger/internal/repository/local"
)

func tempStorage(t *testing.T) *local.MessageStorage {
	t.Helper()
	dir, err := os.MkdirTemp("", "p2p-test-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := local.NewMessageStorage(dir)
	if err != nil {
		t.Fatalf("NewMessageStorage: %v", err)
	}
	return s
}

func TestStoreAndRetrieve(t *testing.T) {
	s := tempStorage(t)

	msg := message.NewMessage("alice", "bob", "hello")
	if err := s.StoreMessage(msg); err != nil {
		t.Fatalf("StoreMessage: %v", err)
	}

	msgs, err := s.GetMessages("alice", "bob", 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Errorf("content: got %q, want %q", msgs[0].Content, "hello")
	}
}

func TestConversationKeySymmetry(t *testing.T) {
	s := tempStorage(t)

	// "alice_bob" and "bob_alice" should map to the same conversation.
	s.StoreMessage(message.NewMessage("alice", "bob", "msg1"))
	s.StoreMessage(message.NewMessage("bob", "alice", "msg2"))

	msgs, err := s.GetMessages("alice", "bob", 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestGetMessagesLimit(t *testing.T) {
	s := tempStorage(t)
	for i := 0; i < 10; i++ {
		s.StoreMessage(message.NewMessage("a", "b", "msg"))
	}

	msgs, err := s.GetMessages("a", "b", 3)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages (limit), got %d", len(msgs))
	}
}

func TestTimestampOrdering(t *testing.T) {
	s := tempStorage(t)

	m1 := message.NewMessage("a", "b", "first")
	m1.Timestamp = time.Now().Add(-2 * time.Second)
	m2 := message.NewMessage("a", "b", "second")
	m2.Timestamp = time.Now().Add(-1 * time.Second)
	m3 := message.NewMessage("a", "b", "third")
	m3.Timestamp = time.Now()

	// Store out of order.
	s.StoreMessage(m3)
	s.StoreMessage(m1)
	s.StoreMessage(m2)

	msgs, _ := s.GetMessages("a", "b", 10)
	want := []string{"first", "second", "third"}
	for i, m := range msgs {
		if m.Content != want[i] {
			t.Errorf("position %d: got %q, want %q", i, m.Content, want[i])
		}
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	dir, err := os.MkdirTemp("", "p2p-persist-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Write a message.
	s1, _ := local.NewMessageStorage(dir)
	s1.StoreMessage(message.NewMessage("x", "y", "persistent"))

	// Reload storage from the same directory.
	s2, err := local.NewMessageStorage(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	msgs, _ := s2.GetMessages("x", "y", 10)
	if len(msgs) != 1 || msgs[0].Content != "persistent" {
		t.Errorf("persistence failed: %+v", msgs)
	}
}

func TestGetConversations(t *testing.T) {
	s := tempStorage(t)
	s.StoreMessage(message.NewMessage("a", "b", "hi"))
	s.StoreMessage(message.NewMessage("a", "c", "hey"))

	convs, err := s.GetConversations()
	if err != nil {
		t.Fatalf("GetConversations: %v", err)
	}
	if len(convs) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convs))
	}
}

func TestGetRecentMessages(t *testing.T) {
	s := tempStorage(t)
	for i := 0; i < 5; i++ {
		s.StoreMessage(message.NewMessage("a", "b", "msg"))
	}
	for i := 0; i < 5; i++ {
		s.StoreMessage(message.NewMessage("a", "c", "msg"))
	}

	recent, err := s.GetRecentMessages(3)
	if err != nil {
		t.Fatalf("GetRecentMessages: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("expected 3 recent messages, got %d", len(recent))
	}
}
