package util

import (
	"errors"
	"fmt"

	hashids "github.com/speps/go-hashids/v2"
)

type IDHasher struct {
	hash *hashids.HashID
}

func NewIDHasher(salt string) (*IDHasher, error) {
	if salt == "" {
		return nil, errors.New("id hash salt is required")
	}
	data := hashids.NewData()
	data.Salt = salt
	data.MinLength = 10
	h, err := hashids.NewWithData(data)
	if err != nil {
		return nil, fmt.Errorf("create hashids: %w", err)
	}
	return &IDHasher{hash: h}, nil
}

func (h *IDHasher) EncodeInt64(id int64) (string, error) {
	if h == nil || h.hash == nil {
		return "", errors.New("id hasher not configured")
	}
	if id <= 0 {
		return "", errors.New("id must be positive")
	}
	return h.hash.EncodeInt64([]int64{id})
}

func (h *IDHasher) DecodeInt64(encoded string) (int64, error) {
	if h == nil || h.hash == nil {
		return 0, errors.New("id hasher not configured")
	}
	ids, err := h.hash.DecodeInt64WithError(encoded)
	if err != nil {
		return 0, errors.New("invalid id hash")
	}
	if len(ids) != 1 {
		return 0, errors.New("invalid id hash")
	}
	return ids[0], nil
}
