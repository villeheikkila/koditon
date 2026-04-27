package util

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

const PublicIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var publicIDBase = big.NewInt(62)

func EncodeUUIDBase62(u uuid.UUID) string {
	return encodeBase62(u[:])
}

func DecodeUUIDBase62(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, errors.New("public id is empty")
	}
	n := big.NewInt(0)
	for i := 0; i < len(s); i++ {
		idx := strings.IndexByte(PublicIDAlphabet, s[i])
		if idx < 0 {
			return uuid.Nil, fmt.Errorf("invalid public id character: %q", s[i])
		}
		n.Mul(n, publicIDBase)
		n.Add(n, big.NewInt(int64(idx)))
	}
	bytes := n.Bytes()
	if len(bytes) > 16 {
		return uuid.Nil, errors.New("public id out of range")
	}
	var out uuid.UUID
	copy(out[16-len(bytes):], bytes)
	return out, nil
}

func encodeBase62(input []byte) string {
	n := new(big.Int).SetBytes(input)
	if n.Sign() == 0 {
		return "0"
	}
	var encoded []byte
	for n.Sign() > 0 {
		mod := new(big.Int)
		n.DivMod(n, publicIDBase, mod)
		encoded = append(encoded, PublicIDAlphabet[mod.Int64()])
	}
	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}
	return string(encoded)
}
