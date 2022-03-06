package session

import (
	"crypto/rand"
	"crypto/subtle"
	"math/big"
	mrand "math/rand"
	"time"
)

// SecureToken generates a unique random token.
// Length should be 24 to match ActiveRecord::SecureToken used by the reference implementation.
func SecureToken(length int) string {
	const base58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	pass := make([]byte, length)
	chars := []byte(base58)
	mrand.New(mrand.NewSource(time.Now().UnixNano())).Shuffle(len(chars), func(i, j int) {
		chars[i], chars[j] = chars[j], chars[i]
	})
	max := big.NewInt(int64(len(chars)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err) // should never occured because max >= 0
		}
		pass[i] = chars[int(n.Int64())]
	}

	return string(pass)
}

// SecureCompare compares the givens strings in a constant time.
// So length info is not leaked via timing attacks.
func SecureCompare(s1, s2 string) bool {
	return subtle.ConstantTimeCompare([]byte(s1), []byte(s2)) == 1
}
