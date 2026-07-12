package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

func TestParseDeviceID(t *testing.T) {
	cases := []struct {
		text []string
		want string
	}{
		{[]string{"id=ABC123"}, "ABC123"},
		{[]string{"other=x", "id=DEV"}, "DEV"},
		{[]string{"nope"}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := parseDeviceID(c.text); got != c.want {
			t.Errorf("parseDeviceID(%v) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestFirstAddr(t *testing.T) {
	e := &zeroconf.ServiceEntry{AddrIPv4: []net.IP{net.ParseIP("192.168.1.5")}}
	if got := firstAddr(e); got != "192.168.1.5" {
		t.Errorf("firstAddr = %q, want 192.168.1.5", got)
	}
	if got := firstAddr(&zeroconf.ServiceEntry{}); got != "" {
		t.Errorf("firstAddr(empty) = %q, want empty", got)
	}
}

func TestBrowseCancelClosesChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	peers, err := Browse(ctx)
	if err != nil {
		t.Skipf("mDNS resolver unavailable in this environment: %v", err)
	}
	cancel()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-peers:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("peers channel was not closed after context cancel")
		}
	}
}

func TestAnnounceRoundTripBestEffort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping mDNS round-trip in short mode")
	}
	deviceID := "TESTDEVICEABCDEFGHIJ"
	ann, err := Announce(deviceID, 22067)
	if err != nil {
		t.Skipf("cannot announce in this environment: %v", err)
	}
	defer ann.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	peers, err := Browse(ctx)
	if err != nil {
		t.Skipf("cannot browse in this environment: %v", err)
	}
	for {
		select {
		case p, ok := <-peers:
			if !ok {
				t.Skip("no device discovered; environment may not support multicast")
				return
			}
			if p.DeviceID == deviceID {
				return
			}
		case <-ctx.Done():
			t.Skip("mDNS round-trip timed out; environment may not support multicast")
			return
		}
	}
}
