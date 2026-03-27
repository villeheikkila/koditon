package auth

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"strings"

	"github.com/oschwald/geoip2-golang/v2"
)

type GeoResolver interface {
	Resolve(context.Context, netip.Addr) (sessionLocationSnapshot, bool, error)
}

type noopGeoResolver struct{}

func (noopGeoResolver) Resolve(context.Context, netip.Addr) (sessionLocationSnapshot, bool, error) {
	return sessionLocationSnapshot{}, false, nil
}

type mmdbGeoResolver struct {
	reader *geoip2.Reader
}

func NewGeoLiteResolver(path string) (GeoResolver, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("geoip mmdb path is required")
	}
	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open geoip mmdb: %w", err)
	}
	return &mmdbGeoResolver{reader: reader}, nil
}

func (r *mmdbGeoResolver) Resolve(_ context.Context, ip netip.Addr) (sessionLocationSnapshot, bool, error) {
	if !shouldResolveGeoIP(ip) {
		return sessionLocationSnapshot{}, false, nil
	}

	record, err := r.reader.City(ip)
	if err != nil {
		return sessionLocationSnapshot{}, false, fmt.Errorf("lookup city record: %w", err)
	}

	snapshot := sessionLocationSnapshot{
		City:        normalizeLocationPart(record.City.Names.English),
		Region:      normalizeLocationPart(firstSubdivisionName(record)),
		CountryCode: strings.ToUpper(strings.TrimSpace(record.Country.ISOCode)),
		Source:      sessionLocationSourceIPGeo,
	}
	if snapshot.City == "" && snapshot.Region == "" && snapshot.CountryCode == "" {
		return sessionLocationSnapshot{}, false, nil
	}
	return snapshot, true, nil
}

func (r *mmdbGeoResolver) Close() error {
	if r == nil || r.reader == nil {
		return nil
	}
	return r.reader.Close()
}

func shouldResolveGeoIP(ip netip.Addr) bool {
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback()
}

func firstSubdivisionName(record *geoip2.City) string {
	for _, subdivision := range record.Subdivisions {
		if name := normalizeLocationPart(subdivision.Names.English); name != "" {
			return name
		}
	}
	return ""
}

func closeGeoResolver(resolver GeoResolver) error {
	if closer, ok := resolver.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
