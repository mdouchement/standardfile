package session_test

import (
	"regexp"
	"testing"

	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/stretchr/testify/assert"
)

func TestSecureToken(t *testing.T) {
	assert.Panics(t, func() { session.SecureToken(-1) })
	assert.Len(t, session.SecureToken(24), 24)
	assert.Regexp(t, regexp.MustCompile(`^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`), session.SecureToken(24))

	n := 8192
	h := make(map[string]bool, 0)
	for i := 0; i < n; i++ {
		h[session.SecureToken(24)] = true
	}
	assert.Len(t, h, n, "tokens must be unique")
}

func TestSecureCompare(t *testing.T) {
	assert.True(t, session.SecureCompare("123456789", "123456789"))
	assert.False(t, session.SecureCompare("123456789", "123456780"))
}
