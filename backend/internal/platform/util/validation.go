package util

import (
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func TrimAndValidate(s string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", huma.Error400BadRequest(fmt.Sprintf("%s cannot be empty or whitespace", fieldName))
	}
	return trimmed, nil
}

func TrimOrNull(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func HasRole(userRoles []string, allowedRoles ...string) bool {
	if len(allowedRoles) == 0 {
		return true
	}
	roleMap := make(map[string]struct{}, len(userRoles))
	for _, role := range userRoles {
		roleMap[role] = struct{}{}
	}
	for _, allowed := range allowedRoles {
		if _, ok := roleMap[allowed]; ok {
			return true
		}
	}
	return false
}

const (
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleVerifier  = "verifier"
	RoleUser      = "user"
)

func CanVerify(roles []string) bool {
	return HasRole(roles, RoleAdmin, RoleModerator, RoleVerifier)
}
