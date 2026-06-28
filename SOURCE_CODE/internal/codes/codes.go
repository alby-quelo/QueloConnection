package codes

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Alphabet excludes ambiguous characters: 0/O, 1/I/L.
const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// Generate returns a human-readable agent code like KITE-7X4M.
func Generate() (string, error) {
	left, err := randomPart(4)
	if err != nil {
		return "", err
	}
	right, err := randomPart(4)
	if err != nil {
		return "", err
	}
	return left + "-" + right, nil
}

func randomPart(n int) (string, error) {
	var b strings.Builder
	max := big.NewInt(int64(len(alphabet)))
	for range n {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate code: %w", err)
		}
		b.WriteByte(alphabet[idx.Int64()])
	}
	return b.String(), nil
}

// Normalize uppercases and strips spaces for user input.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
