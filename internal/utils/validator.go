package utils

import (
	"fmt"
	"strings"

	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
)

// ValidatePeerID returns an error if s is not a valid libp2p peer ID.
func ValidatePeerID(s string) error {
	if s == "" {
		return fmt.Errorf("peer ID must not be empty")
	}
	if _, err := libp2ppeer.Decode(s); err != nil {
		return fmt.Errorf("invalid peer ID %q: %w", s, err)
	}
	return nil
}

// ValidateMessageContent returns an error if content is empty or too long.
func ValidateMessageContent(content string, maxLen int) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("message content must not be empty")
	}
	if maxLen > 0 && len(content) > maxLen {
		return fmt.Errorf("message content exceeds maximum length of %d", maxLen)
	}
	return nil
}

// ValidatePort returns an error if port is outside the valid range.
func ValidatePort(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("port %d out of range [0, 65535]", port)
	}
	return nil
}
