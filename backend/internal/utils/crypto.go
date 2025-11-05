package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeSHA256 returns the hex-encoded SHA256 of a string
func ComputeSHA256(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}