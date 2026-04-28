package frontdoor

import (
	"encoding/json"
	"fmt"
)

type RawAd map[string]any

func DecodeStoredAd(raw json.RawMessage) (*AdResponse, RawAd, error) {
	ad, err := DecodeAd(raw)
	if err != nil {
		return nil, nil, err
	}
	rawAd, err := DecodeAdRaw(raw)
	if err != nil {
		return nil, nil, err
	}
	return ad, rawAd, nil
}

func DecodeAd(data []byte) (*AdResponse, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("frontdoor ad payload is empty")
	}
	var ad AdResponse
	if err := json.Unmarshal(data, &ad); err != nil {
		return nil, fmt.Errorf("decode frontdoor ad response: %w", err)
	}
	return &ad, nil
}

func DecodeAdRaw(data []byte) (RawAd, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("frontdoor ad payload is empty")
	}
	var ad RawAd
	if err := json.Unmarshal(data, &ad); err != nil {
		return nil, fmt.Errorf("decode raw frontdoor ad response: %w", err)
	}
	return ad, nil
}
