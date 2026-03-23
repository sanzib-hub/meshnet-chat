package identity

// Identity holds the local node's human-readable profile.
// The cryptographic identity (private key + peer ID) lives in
// internal/service/identity — this domain object holds display metadata only.
type Identity struct {
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

func New(peerID, displayName string) *Identity {
	return &Identity{
		PeerID:      peerID,
		DisplayName: displayName,
	}
}
