package auth

import (
	"context"
	"io"
	"net/netip"
)

type GeoResolver interface {
	Resolve(context.Context, netip.Addr) (sessionLocationSnapshot, bool, error)
}

type noopGeoResolver struct{}

func (noopGeoResolver) Resolve(context.Context, netip.Addr) (sessionLocationSnapshot, bool, error) {
	return sessionLocationSnapshot{}, false, nil
}

func shouldResolveGeoIP(ip netip.Addr) bool {
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback()
}

func closeGeoResolver(resolver GeoResolver) error {
	if closer, ok := resolver.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
