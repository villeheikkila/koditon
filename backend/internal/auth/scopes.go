package auth

import (
	"fmt"
	"slices"
)

const (
	ScopeCoreRead     = "core:read"
	ScopeMCPCoreRead  = "mcp:core:read"
	ScopeProfileRead  = "profile:read"
	ScopeProfileWrite = "profile:write"
)

var userScopes = []string{
	ScopeCoreRead,
	ScopeMCPCoreRead,
	ScopeProfileRead,
	ScopeProfileWrite,
}

var adminScopes = []string{}

func ScopesForRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(userScopes)+len(adminScopes))
	scopes := make([]string, 0, len(userScopes)+len(adminScopes))
	addScopes := func(values []string) {
		for _, scope := range values {
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			scopes = append(scopes, scope)
		}
	}
	addScopes(userScopes)
	if slices.Contains(roles, "admin") {
		addScopes(adminScopes)
	}
	return scopes
}

func HasScopes(available []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(available) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(available))
	for _, scope := range available {
		set[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := set[scope]; !ok {
			return false
		}
	}
	return true
}

func LimitScopes(available []string, limit []string) []string {
	if len(limit) == 0 {
		return available
	}
	if len(available) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(available))
	for _, scope := range available {
		set[scope] = struct{}{}
	}
	limited := make([]string, 0, len(limit))
	for _, scope := range limit {
		if _, ok := set[scope]; ok {
			limited = append(limited, scope)
		}
	}
	return limited
}

func HasScopeForRoles(roles []string, required string) bool {
	return HasScopes(ScopesForRoles(roles), []string{required})
}

func ValidateRequestedScopes(requested, clientAllowed, userAllowed []string) error {
	if len(requested) == 0 {
		return nil
	}
	if len(clientAllowed) > 0 && !HasScopes(clientAllowed, requested) {
		return fmt.Errorf("%w: requested scope is not allowed for this client", ErrInvalidScope)
	}
	if len(userAllowed) > 0 && !HasScopes(userAllowed, requested) {
		return fmt.Errorf("%w: requested scope exceeds user grants", ErrInvalidScope)
	}
	return nil
}

func ClampRequestedScopesToUserGrants(requested, clientAllowed, userAllowed []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	if len(clientAllowed) > 0 && !HasScopes(clientAllowed, requested) {
		return nil, fmt.Errorf("%w: requested scope is not allowed for this client", ErrInvalidScope)
	}
	if userAllowed == nil {
		return append([]string(nil), requested...), nil
	}
	granted := LimitScopes(userAllowed, requested)
	if len(granted) == 0 {
		return nil, fmt.Errorf("%w: requested scope exceeds user grants", ErrInvalidScope)
	}
	return granted, nil
}
