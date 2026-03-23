package peer

import "time"

// Peer represents a known network peer.
type Peer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Addresses []string  `json:"addresses,omitempty"`
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"last_seen"`
}

func New(id string) *Peer {
	return &Peer{
		ID:       id,
		Name:     shortID(id),
		Online:   true,
		LastSeen: time.Now(),
	}
}

// shortID returns the first 8 chars of the peer ID as a display name.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
