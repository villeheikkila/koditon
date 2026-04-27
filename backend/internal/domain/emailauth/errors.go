package emailauth

import "errors"

var (
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidToken   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenConsumed  = errors.New("token consumed")
	ErrInvalidTicket  = errors.New("invalid auth ticket")
	ErrTicketExpired  = errors.New("auth ticket expired")
	ErrTicketConsumed = errors.New("auth ticket consumed")
)
