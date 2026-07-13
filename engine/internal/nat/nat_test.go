package nat

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	natpmp "github.com/jackpal/go-nat-pmp"
)

func TestEndpoint(t *testing.T) {
	if got := Endpoint("203.0.113.9", 22067); got != "203.0.113.9:22067" {
		t.Errorf("Endpoint = %q", got)
	}
	if got := Endpoint("2001:db8::1", 22067); got != "[2001:db8::1]:22067" {
		t.Errorf("IPv6 Endpoint = %q, want bracketed host", got)
	}
}

func ipNet(s string) *net.IPNet {
	ip, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	n.IP = ip
	return n
}

func TestFilterLANIPs(t *testing.T) {
	addrs := []net.Addr{
		ipNet("127.0.0.1/8"),
		ipNet("169.254.10.20/16"),
		ipNet("192.168.1.42/24"),
		ipNet("10.1.2.3/8"),
		ipNet("::1/128"),
		ipNet("fe80::abcd/64"),
		ipNet("2001:db8::7/64"),
		&net.TCPAddr{IP: net.ParseIP("192.168.9.9"), Port: 80},
	}
	got := filterLANIPs(addrs)
	want := []string{"192.168.1.42", "10.1.2.3", "2001:db8::7"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filterLANIPs = %v, want %v", got, want)
	}
}

func TestPublicIP(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":       true,
		"203.0.113.10":  true,
		"2001:4860::1":  true,
		"10.0.0.1":      false,
		"172.16.5.5":    false,
		"192.168.1.1":   false,
		"100.64.0.1":    false,
		"100.127.255.1": false,
		"100.128.0.1":   true,
		"127.0.0.1":     false,
		"169.254.1.1":   false,
		"0.0.0.0":       false,
	}
	for s, want := range cases {
		if got := publicIP(net.ParseIP(s)); got != want {
			t.Errorf("publicIP(%s) = %v, want %v", s, got, want)
		}
	}
	if publicIP(nil) {
		t.Error("publicIP(nil) should be false")
	}
}

type fakeIGD struct {
	external string
	extErr   error
	mapErr   error
	local    net.IP

	mappedExternal uint16
	mappedInternal uint16
	mappedProto    string
	mappedClient   string
}

func (f *fakeIGD) GetExternalIPAddressCtx(context.Context) (string, error) {
	return f.external, f.extErr
}

func (f *fakeIGD) AddPortMappingCtx(_ context.Context, _ string, ext uint16, proto string, internal uint16, client string, _ bool, _ string, _ uint32) error {
	if f.mapErr != nil {
		return f.mapErr
	}
	f.mappedExternal, f.mappedInternal, f.mappedProto, f.mappedClient = ext, internal, proto, client
	return nil
}

func (f *fakeIGD) LocalAddr() net.IP { return f.local }

type fakePMP struct {
	external [4]byte
	extErr   error
	mapErr   error
	extPort  uint16
}

func (f *fakePMP) GetExternalAddress() (*natpmp.GetExternalAddressResult, error) {
	if f.extErr != nil {
		return nil, f.extErr
	}
	return &natpmp.GetExternalAddressResult{ExternalIPAddress: f.external}, nil
}

func (f *fakePMP) AddPortMapping(string, int, int, int) (*natpmp.AddPortMappingResult, error) {
	if f.mapErr != nil {
		return nil, f.mapErr
	}
	return &natpmp.AddPortMappingResult{MappedExternalPort: f.extPort}, nil
}

func stubDiscovery(t *testing.T, igds []igd, pmpClient pmp, pmpErr error) {
	t.Helper()
	oldIGD, oldPMP := discoverIGD, discoverPMP
	discoverIGD = func(context.Context) []igd { return igds }
	discoverPMP = func() (pmp, error) { return pmpClient, pmpErr }
	t.Cleanup(func() { discoverIGD, discoverPMP = oldIGD, oldPMP })
}

func TestExternalEndpointNoGateway(t *testing.T) {
	stubDiscovery(t, nil, nil, errors.New("no gateway"))
	if _, err := ExternalEndpoint(context.Background(), 22067); !errors.Is(err, ErrNoMapping) {
		t.Errorf("error = %v, want ErrNoMapping", err)
	}
}

func TestExternalEndpointUPnP(t *testing.T) {
	dev := &fakeIGD{external: "203.0.113.5", local: net.ParseIP("192.168.1.20")}
	stubDiscovery(t, []igd{dev}, nil, errors.New("unused"))

	ep, err := ExternalEndpoint(context.Background(), 22067)
	if err != nil {
		t.Fatalf("ExternalEndpoint: %v", err)
	}
	if ep != "203.0.113.5:22067" {
		t.Errorf("endpoint = %q", ep)
	}
	if dev.mappedProto != "UDP" || dev.mappedExternal != 22067 || dev.mappedInternal != 22067 {
		t.Errorf("mapping = %s %d->%d, want UDP 22067->22067", dev.mappedProto, dev.mappedExternal, dev.mappedInternal)
	}
	if dev.mappedClient != "192.168.1.20" {
		t.Errorf("internal client = %q", dev.mappedClient)
	}
}

func TestExternalEndpointSkipsBrokenIGDs(t *testing.T) {
	good := &fakeIGD{external: "203.0.113.6", local: net.ParseIP("192.168.1.20")}
	stubDiscovery(t, []igd{
		&fakeIGD{extErr: errors.New("down")},
		&fakeIGD{external: "10.0.0.2", local: net.ParseIP("192.168.1.20")},
		&fakeIGD{external: "203.0.113.5", local: net.ParseIP("192.168.1.20"), mapErr: errors.New("refused")},
		good,
	}, nil, errors.New("unused"))

	ep, err := ExternalEndpoint(context.Background(), 41000)
	if err != nil {
		t.Fatalf("ExternalEndpoint: %v", err)
	}
	if ep != "203.0.113.6:41000" {
		t.Errorf("endpoint = %q", ep)
	}
}

func TestExternalEndpointRejectsPrivateExternalIP(t *testing.T) {
	stubDiscovery(t, []igd{&fakeIGD{external: "192.168.0.2", local: net.ParseIP("192.168.1.20")}}, nil, errors.New("no gateway"))
	if _, err := ExternalEndpoint(context.Background(), 22067); !errors.Is(err, ErrNoMapping) {
		t.Errorf("error = %v, want ErrNoMapping for double-NAT external IP", err)
	}
}

func TestExternalEndpointNATPMPFallback(t *testing.T) {
	stubDiscovery(t, nil, &fakePMP{external: [4]byte{203, 0, 113, 7}, extPort: 40123}, nil)
	ep, err := ExternalEndpoint(context.Background(), 22067)
	if err != nil {
		t.Fatalf("ExternalEndpoint: %v", err)
	}
	if ep != "203.0.113.7:40123" {
		t.Errorf("endpoint = %q", ep)
	}
}

func TestExternalEndpointNATPMPPrivate(t *testing.T) {
	stubDiscovery(t, nil, &fakePMP{external: [4]byte{100, 64, 1, 1}, extPort: 40123}, nil)
	if _, err := ExternalEndpoint(context.Background(), 22067); !errors.Is(err, ErrNoMapping) {
		t.Errorf("error = %v, want ErrNoMapping for CGNAT external IP", err)
	}
}
