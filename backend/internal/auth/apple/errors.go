package apple

import "errors"

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidNonce    = errors.New("invalid nonce")
	ErrTokenExchange   = errors.New("apple token exchange failed")
	ErrInvalidIssuer   = errors.New("invalid issuer")
	ErrInvalidAudience = errors.New("invalid audience")
)
