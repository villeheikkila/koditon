package emailaddr

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

var ErrInvalidEmail = errors.New("invalid email")

type Address string

func Parse(raw string) (Address, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidEmail
	}
	if strings.Count(trimmed, "@") != 1 {
		return "", ErrInvalidEmail
	}
	for _, r := range trimmed {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", ErrInvalidEmail
		}
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmail
	}
	if parsed.Address != trimmed {
		return "", ErrInvalidEmail
	}

	localPart, domain, ok := strings.Cut(trimmed, "@")
	if !ok || localPart == "" || domain == "" {
		return "", ErrInvalidEmail
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return "", ErrInvalidEmail
	}
	if !strings.Contains(domain, ".") {
		return "", ErrInvalidEmail
	}

	return Address(strings.ToLower(localPart + "@" + domain)), nil
}

func (a Address) String() string {
	return string(a)
}
