package app

import "testing"

func TestParseDnsmasqVersion(t *testing.T) {
	got := parseDnsmasqVersion("Dnsmasq version 2.90  Copyright (c) 2000-2024 Simon Kelley")
	if got != "2.90" {
		t.Fatalf("parseDnsmasqVersion() = %q, want %q", got, "2.90")
	}
}

func TestParseDnsmasqVersionUnrecognized(t *testing.T) {
	if got := parseDnsmasqVersion("no version information available"); got != "" {
		t.Fatalf("parseDnsmasqVersion() = %q, want empty string", got)
	}
}
