package sourcejson

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

// HashAlgorithmSHA256 identifies the source JSON hash algorithm.
const HashAlgorithmSHA256 = "sha256"

// CanonicalizeAndHash returns stable JSON bytes and their SHA-256 hash.
func CanonicalizeAndHash(data []byte) ([]byte, string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, "", nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, "", fmt.Errorf("decode json payload: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, "", fmt.Errorf("decode json payload: trailing data")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, "", fmt.Errorf("canonicalize json payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return canonical, hex.EncodeToString(sum[:]), nil
}
