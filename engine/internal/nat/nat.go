// Package nat discovers the router's external address and maps the daemon's
// QUIC port via UPnP-IGD or NAT-PMP, so peers outside the LAN can connect.
// Everything here is best-effort: when the router offers no mapping the caller
// simply stays LAN-only.
package nat

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/jackpal/gateway"
	natpmp "github.com/jackpal/go-nat-pmp"
)

const (
	mappingName  = "syncy"
	LeaseSeconds = 3600
)

var ErrNoMapping = errors.New("nat: router offers no usable port mapping")

type igd interface {
	GetExternalIPAddressCtx(ctx context.Context) (string, error)
	AddPortMappingCtx(ctx context.Context, remoteHost string, externalPort uint16, protocol string, internalPort uint16, internalClient string, enabled bool, description string, lease uint32) error
	LocalAddr() net.IP
}

type pmp interface {
	GetExternalAddress() (*natpmp.GetExternalAddressResult, error)
	AddPortMapping(protocol string, internalPort, externalPort, lifetime int) (*natpmp.AddPortMappingResult, error)
}

var discoverIGD = func(ctx context.Context) []igd {
	var out []igd
	if cs, _, err := internetgateway2.NewWANIPConnection2ClientsCtx(ctx); err == nil {
		out = appendClients(out, cs)
	}
	if cs, _, err := internetgateway2.NewWANIPConnection1ClientsCtx(ctx); err == nil {
		out = appendClients(out, cs)
	}
	if cs, _, err := internetgateway1.NewWANIPConnection1ClientsCtx(ctx); err == nil {
		out = appendClients(out, cs)
	}
	if cs, _, err := internetgateway1.NewWANPPPConnection1ClientsCtx(ctx); err == nil {
		out = appendClients(out, cs)
	}
	return out
}

var discoverPMP = func() (pmp, error) {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, err
	}
	return natpmp.NewClientWithTimeout(gw, 2*time.Second), nil
}

func appendClients[T igd](out []igd, clients []T) []igd {
	for _, c := range clients {
		out = append(out, c)
	}
	return out
}

// ExternalEndpoint maps localPort (UDP) on the gateway and returns the
// externally reachable host:port. The mapping leases LeaseSeconds and must be
// refreshed by the caller.
func ExternalEndpoint(ctx context.Context, localPort int) (string, error) {
	if ep, err := upnpEndpoint(ctx, localPort); err == nil {
		return ep, nil
	}
	return pmpEndpoint(localPort)
}

func upnpEndpoint(ctx context.Context, localPort int) (string, error) {
	for _, c := range discoverIGD(ctx) {
		ext, err := c.GetExternalIPAddressCtx(ctx)
		if err != nil {
			continue
		}
		ip := net.ParseIP(ext)
		if !publicIP(ip) {
			continue
		}
		local := c.LocalAddr()
		if local == nil {
			continue
		}
		err = c.AddPortMappingCtx(ctx, "", uint16(localPort), "UDP", uint16(localPort), local.String(), true, mappingName, LeaseSeconds)
		if err != nil {
			continue
		}
		return Endpoint(ip.String(), localPort), nil
	}
	return "", ErrNoMapping
}

func pmpEndpoint(localPort int) (string, error) {
	c, err := discoverPMP()
	if err != nil {
		return "", ErrNoMapping
	}
	addr, err := c.GetExternalAddress()
	if err != nil {
		return "", ErrNoMapping
	}
	ip := net.IP(addr.ExternalIPAddress[:])
	if !publicIP(ip) {
		return "", ErrNoMapping
	}
	res, err := c.AddPortMapping("udp", localPort, localPort, LeaseSeconds)
	if err != nil {
		return "", ErrNoMapping
	}
	return Endpoint(ip.String(), int(res.MappedExternalPort)), nil
}

func Endpoint(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// LANIPs lists the machine's non-loopback addresses peers on the same network
// can reach.
func LANIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	return filterLANIPs(addrs), nil
}

func filterLANIPs(addrs []net.Addr) []string {
	var out []string
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipn.IP
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			out = append(out, v4.String())
			continue
		}
		if ip.IsGlobalUnicast() {
			out = append(out, ip.String())
		}
	}
	return out
}

func publicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}
	return true
}
