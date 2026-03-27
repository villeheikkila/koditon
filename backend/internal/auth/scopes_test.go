package auth

import (
	"errors"
	"slices"
	"testing"
)

func TestClampRequestedScopesToUserGrantsKeepsRequestedScopesWhenAllowed(t *testing.T) {
	t.Parallel()

	got, err := ClampRequestedScopesToUserGrants(
		[]string{ScopeCoreRead, ScopeProfileRead},
		[]string{ScopeCoreRead, ScopeProfileRead, ScopeProfileWrite},
		[]string{ScopeCoreRead, ScopeProfileRead, ScopeProfileWrite},
	)
	if err != nil {
		t.Fatalf("ClampRequestedScopesToUserGrants returned error: %v", err)
	}

	want := []string{ScopeCoreRead, ScopeProfileRead}
	if !slices.Equal(got, want) {
		t.Fatalf("ClampRequestedScopesToUserGrants = %v, want %v", got, want)
	}
}

func TestClampRequestedScopesToUserGrantsDropsScopesMissingFromUserGrants(t *testing.T) {
	t.Parallel()

	got, err := ClampRequestedScopesToUserGrants(
		[]string{ScopeCoreRead, ScopeAdminProductsRead, ScopeProfileRead},
		[]string{ScopeCoreRead, ScopeProfileRead, ScopeAdminProductsRead},
		[]string{ScopeCoreRead, ScopeProfileRead},
	)
	if err != nil {
		t.Fatalf("ClampRequestedScopesToUserGrants returned error: %v", err)
	}

	want := []string{ScopeCoreRead, ScopeProfileRead}
	if !slices.Equal(got, want) {
		t.Fatalf("ClampRequestedScopesToUserGrants = %v, want %v", got, want)
	}
}

func TestClampRequestedScopesToUserGrantsRejectsClientDisallowedScopes(t *testing.T) {
	t.Parallel()

	_, err := ClampRequestedScopesToUserGrants(
		[]string{ScopeCoreRead, ScopeAdminProductsRead},
		[]string{ScopeCoreRead, ScopeProfileRead},
		[]string{ScopeCoreRead, ScopeAdminProductsRead},
	)
	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("ClampRequestedScopesToUserGrants error = %v, want ErrInvalidScope", err)
	}
}

func TestClampRequestedScopesToUserGrantsRejectsEmptyGrantedScopes(t *testing.T) {
	t.Parallel()

	_, err := ClampRequestedScopesToUserGrants(
		[]string{ScopeAdminProductsRead},
		[]string{ScopeAdminProductsRead},
		[]string{ScopeCoreRead},
	)
	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("ClampRequestedScopesToUserGrants error = %v, want ErrInvalidScope", err)
	}
}
