package room

import (
	"time"

	"github.com/google/uuid"
)

// Room represents a named group chat channel.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Members   []string  `json:"members"` // peer ID strings
	CreatedAt time.Time `json:"created_at"`
}

func New(name string, members ...string) *Room {
	return &Room{
		ID:        uuid.New().String(),
		Name:      name,
		Members:   members,
		CreatedAt: time.Now(),
	}
}

func (r *Room) AddMember(peerID string) {
	for _, m := range r.Members {
		if m == peerID {
			return
		}
	}
	r.Members = append(r.Members, peerID)
}

func (r *Room) RemoveMember(peerID string) {
	filtered := r.Members[:0]
	for _, m := range r.Members {
		if m != peerID {
			filtered = append(filtered, m)
		}
	}
	r.Members = filtered
}

func (r *Room) HasMember(peerID string) bool {
	for _, m := range r.Members {
		if m == peerID {
			return true
		}
	}
	return false
}
