package common

import (
	"net"
	"testing"
)

// ── QuoteIdentifier ───────────────────────────────────────────────────────────

func TestQuoteIdentifier(t *testing.T) {
	cases := []struct{ in, want string }{
		{"users", `"users"`},
		{"order_items", `"order_items"`},
		{"", `""`},
	}
	for _, c := range cases {
		got := QuoteIdentifier(c.in)
		if got != c.want {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── FindAvailablePort ─────────────────────────────────────────────────────────

func TestFindAvailablePort_ReturnsUsablePort(t *testing.T) {
	port := FindAvailablePort(19000)
	if port < 19000 || port > 19100 {
		t.Errorf("port = %d, want in range [19000, 19100]", port)
	}
	// Verify the returned port is actually bindable.
	ln, err := net.Listen("tcp", ":"+itoa(port))
	if err != nil {
		t.Errorf("returned port %d is not bindable: %v", port, err)
	} else {
		ln.Close()
	}
}

func TestFindAvailablePort_SkipsOccupied(t *testing.T) {
	// Occupy a port, then ask FindAvailablePort to start from it.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot bind test port")
	}
	defer ln.Close()
	occupied := ln.Addr().(*net.TCPAddr).Port

	// FindAvailablePort should return a different port.
	got := FindAvailablePort(occupied)
	if got == occupied {
		t.Errorf("FindAvailablePort returned occupied port %d", occupied)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
