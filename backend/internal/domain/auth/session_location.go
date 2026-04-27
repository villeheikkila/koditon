package auth

import (
	"context"
	"log/slog"
	"net/netip"
	"strings"

	"koditon/internal/platform/logging"

	"golang.org/x/text/language"
)

const (
	sessionLocationSourceIPGeo    = "ip_geo"
	sessionLocationSourceTimeZone = "time_zone"
	sessionLocationSourceLocale   = "locale"
	sessionLocationSourceUnknown  = "unknown"
)

type sessionLocationSnapshot struct {
	City        string
	Region      string
	CountryCode string
	Source      string
}

func (s *Service) resolveSessionLocation(ctx context.Context, params createSessionParams) sessionLocationSnapshot {
	if ip, ok := parseGlobalSessionIP(params.IP); ok && s.geoResolver != nil {
		if snapshot, resolved, err := s.geoResolver.Resolve(ctx, ip); err == nil && resolved {
			return snapshot
		} else if err != nil {
			logging.With(s.logger, logging.Op("auth.session_location.resolve"), slog.String("ip", ip.String())).WarnContext(ctx, "geoip lookup failed", "error", err, "outcome", logging.OutcomeError)
		}
	}
	return resolveFallbackSessionLocation(params.DeviceTimeZone, params.DeviceLocale)
}

func resolveFallbackSessionLocation(timeZoneIdentifier, localeIdentifier string) sessionLocationSnapshot {
	timeZoneIdentifier = strings.TrimSpace(timeZoneIdentifier)
	localeIdentifier = strings.TrimSpace(localeIdentifier)

	snapshot := sessionLocationSnapshot{
		Source: sessionLocationSourceUnknown,
	}

	if timeZoneIdentifier != "" {
		parts := strings.Split(timeZoneIdentifier, "/")
		if len(parts) > 0 {
			snapshot.Region = normalizeLocationPart(parts[0])
		}
		if len(parts) > 1 {
			snapshot.City = normalizeLocationPart(parts[len(parts)-1])
		}
		if snapshot.City != "" || snapshot.Region != "" {
			snapshot.Source = sessionLocationSourceTimeZone
		}
	}

	if localeIdentifier != "" {
		if base, err := language.Parse(localeIdentifier); err == nil {
			if region, confidence := base.Region(); confidence != language.No {
				snapshot.CountryCode = strings.ToUpper(region.String())
				if snapshot.Source == sessionLocationSourceUnknown {
					snapshot.Source = sessionLocationSourceLocale
				}
			}
		}
	}

	return snapshot
}

func normalizeLocationPart(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return ""
	}
	return value
}

func parseGlobalSessionIP(raw string) (netip.Addr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(raw)
	if err != nil || !shouldResolveGeoIP(ip) {
		return netip.Addr{}, false
	}
	return ip, true
}
