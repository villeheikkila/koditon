package apple

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidNonce    = errors.New("invalid nonce")
	ErrTokenExchange   = errors.New("apple token exchange failed")
	ErrInvalidIssuer   = errors.New("invalid issuer")
	ErrInvalidAudience = errors.New("invalid audience")
)

type TokenExchangeError struct {
	StatusCode       int
	ErrorCode        string
	ErrorDescription string
}

func (e *TokenExchangeError) Error() string {
	if e == nil {
		return ErrTokenExchange.Error()
	}
	if e.ErrorCode != "" && e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s (%s)", ErrTokenExchange, e.ErrorCode, e.ErrorDescription)
	}
	if e.ErrorCode != "" {
		return fmt.Sprintf("%s: %s", ErrTokenExchange, e.ErrorCode)
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("%s: status %d", ErrTokenExchange, e.StatusCode)
	}
	return ErrTokenExchange.Error()
}

func (e *TokenExchangeError) Unwrap() error {
	return ErrTokenExchange
}
