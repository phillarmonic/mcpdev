package canonical

import (
	"crypto/sha256"
	"encoding/hex"
)

// Digest is SHA-256 of already-canonical bytes.
func Digest(canonical []byte) [32]byte {
	return sha256.Sum256(canonical)
}

// Hex formats a digest as lowercase hexadecimal.
func Hex(sum [32]byte) string {
	return hex.EncodeToString(sum[:])
}

// SchemaDigest canonicalizes raw schema JSON and returns its hex digest.
func SchemaDigest(raw []byte) (string, error) {
	body, err := Schema(raw)
	if err != nil {
		return "", err
	}
	return Hex(Digest(body)), nil
}
