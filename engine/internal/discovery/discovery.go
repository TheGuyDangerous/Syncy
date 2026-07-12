// Package discovery advertises this device on the local network and discovers
// peers over mDNS / DNS-SD, so devices find each other without configuration.
package discovery

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

const (
	service = "_syncy._udp"
	domain  = "local."
	txtKey  = "id="
)

type Peer struct {
	DeviceID string
	Addr     string
}

type Announcer struct {
	server *zeroconf.Server
}

func Announce(deviceID string, port int) (*Announcer, error) {
	server, err := zeroconf.Register(deviceID, service, domain, port, []string{txtKey + deviceID}, nil)
	if err != nil {
		return nil, err
	}
	return &Announcer{server: server}, nil
}

func (a *Announcer) Close() {
	a.server.Shutdown()
}

func Browse(ctx context.Context) (<-chan Peer, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	peers := make(chan Peer)

	go func() {
		defer close(peers)
		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-entries:
				if !ok {
					return
				}
				id := parseDeviceID(entry.Text)
				addr := firstAddr(entry)
				if id == "" || addr == "" {
					continue
				}
				select {
				case peers <- Peer{DeviceID: id, Addr: net.JoinHostPort(addr, strconv.Itoa(entry.Port))}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	go func() { _ = resolver.Browse(ctx, service, domain, entries) }()
	return peers, nil
}

func parseDeviceID(text []string) string {
	for _, t := range text {
		if strings.HasPrefix(t, txtKey) {
			return strings.TrimPrefix(t, txtKey)
		}
	}
	return ""
}

func firstAddr(entry *zeroconf.ServiceEntry) string {
	if len(entry.AddrIPv4) > 0 {
		return entry.AddrIPv4[0].String()
	}
	if len(entry.AddrIPv6) > 0 {
		return entry.AddrIPv6[0].String()
	}
	return ""
}
