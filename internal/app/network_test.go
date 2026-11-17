package app

import (
	"net"
	"testing"
	"time"
)

func TestFormatWebURLHandlesIPv4AndIPv6(t *testing.T) {
	if got := formatWebURL("http", "192.0.2.10"); got != "http://192.0.2.10" {
		t.Fatalf("IPv4 URL = %q", got)
	}
	if got := formatWebURL("https", "2001:db8::10"); got != "https://[2001:db8::10]" {
		t.Fatalf("IPv6 URL = %q", got)
	}
	if got := formatWebURL("http", "not-an-ip"); got != "" {
		t.Fatalf("invalid IP URL = %q, want empty", got)
	}
}

func TestResolvesToIPRejectsInvalidInputs(t *testing.T) {
	if resolvesToIP("localhost", "not-an-ip", time.Second) {
		t.Fatal("invalid IP must not resolve")
	}
	if resolvesToIP("", "127.0.0.1", time.Second) {
		t.Fatal("empty hostname must not resolve")
	}
}

func TestPingReachableRejectsInvalidIP(t *testing.T) {
	if pingReachable("not-an-ip", time.Second) {
		t.Fatal("invalid IP must not be reachable")
	}
}

func TestCheckWebUIRejectsInvalidIP(t *testing.T) {
	if got := checkWebUI("not-an-ip", time.Second); got != "" {
		t.Fatalf("checkWebUI() = %q, want empty", got)
	}
}

func TestResolvesToIPLocalhost(t *testing.T) {
	// localhost resolution is expected to be available without relying on
	// external DNS; skip if the host environment does not provide it.
	ips, err := net.LookupIP("localhost")
	if err != nil || len(ips) == 0 {
		t.Skip("localhost is not resolvable in this test environment")
	}
	want := ips[0].String()
	if !resolvesToIP("localhost", want, 2*time.Second) {
		t.Fatalf("localhost did not resolve to %s", want)
	}
}
