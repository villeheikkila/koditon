package frontdoor

import (
	"testing"

	"github.com/google/uuid"

	frontdoorpayload "koditon/internal/providers/frontdoor"
)

func TestAnnouncementIdentityIgnoresMutablePriceFields(t *testing.T) {
	t.Parallel()
	first := frontdoorpayload.Announcement{ID: ptr(123), SearchPrice: ptr(250000.0), PricePerSquare: ptr(5000.0)}
	updated := frontdoorpayload.Announcement{ID: ptr(123), SearchPrice: ptr(240000.0), PricePerSquare: ptr(4800.0)}
	if announcementIdentityKey(first, uuid.New()) != announcementIdentityKey(updated, uuid.New()) {
		t.Fatal("price change should preserve announcement identity")
	}
}

func TestAnnouncementIdentityPrefersExternalID(t *testing.T) {
	t.Parallel()
	first := frontdoorpayload.Announcement{ID: ptr(123), FriendlyID: ptr("old-friendly-id")}
	updated := frontdoorpayload.Announcement{ID: ptr(123), FriendlyID: ptr("new-friendly-id")}
	if got, want := announcementIdentityKey(first, uuid.New()), announcementIdentityKey(updated, uuid.New()); got != want {
		t.Fatalf("identity changed from %q to %q", got, want)
	}
}

func TestFilterUniqueAnnouncementsKeepsPublishedUpdate(t *testing.T) {
	t.Parallel()
	announcements := []frontdoorpayload.Announcement{
		{FriendlyID: ptr("listing-123"), SearchPrice: ptr(250000.0), Published: ptr(false)},
		{FriendlyID: ptr("listing-123"), SearchPrice: ptr(240000.0), Published: ptr(true)},
	}
	got := filterUniqueAnnouncements(announcements)
	if len(got) != 1 {
		t.Fatalf("got %d announcements, want 1", len(got))
	}
	if got[0].SearchPrice == nil || *got[0].SearchPrice != 240000 {
		t.Fatalf("got price %v, want 240000", got[0].SearchPrice)
	}
}
