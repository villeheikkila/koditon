package auth

import (
	"context"
	"net/netip"
	"testing"
)

type fakeGeoResolver struct {
	snapshot sessionLocationSnapshot
	ok       bool
	err      error
}

func (f fakeGeoResolver) Resolve(context.Context, netip.Addr) (sessionLocationSnapshot, bool, error) {
	return f.snapshot, f.ok, f.err
}

func TestResolveSessionLocation_PrefersIPGeo(t *testing.T) {
	t.Parallel()

	service := &Service{
		geoResolver: fakeGeoResolver{
			snapshot: sessionLocationSnapshot{
				City:        "Helsinki",
				Region:      "Uusimaa",
				CountryCode: "FI",
				Source:      sessionLocationSourceIPGeo,
			},
			ok: true,
		},
	}

	location := service.resolveSessionLocation(context.Background(), createSessionParams{
		IP:             "91.198.174.192",
		DeviceTimeZone: "Europe/Stockholm",
		DeviceLocale:   "sv_SE",
	})

	if location.Source != sessionLocationSourceIPGeo || location.City != "Helsinki" || location.CountryCode != "FI" {
		t.Fatalf("expected ip geo location, got %+v", location)
	}
}

func TestResolveSessionLocation_FallsBackToTimeZoneAndLocale(t *testing.T) {
	t.Parallel()

	service := &Service{
		geoResolver: fakeGeoResolver{},
	}

	location := service.resolveSessionLocation(context.Background(), createSessionParams{
		IP:             "127.0.0.1",
		DeviceTimeZone: "Europe/Helsinki",
		DeviceLocale:   "fi_FI",
	})

	if location.Source != sessionLocationSourceTimeZone {
		t.Fatalf("expected time zone fallback, got %+v", location)
	}
	if location.City != "Helsinki" || location.Region != "Europe" || location.CountryCode != "FI" {
		t.Fatalf("unexpected fallback location: %+v", location)
	}
}

func TestShouldResolveGeoIPRejectsPrivateAndLoopback(t *testing.T) {
	t.Parallel()

	cases := []string{"127.0.0.1", "10.0.0.5", "192.168.1.10"}
	for _, raw := range cases {
		ip := netip.MustParseAddr(raw)
		if shouldResolveGeoIP(ip) {
			t.Fatalf("expected ip %s to be ignored", raw)
		}
	}
}
