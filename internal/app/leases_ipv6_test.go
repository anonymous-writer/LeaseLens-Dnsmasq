package app

import "testing"

func TestParseIPv4LeaseKeepsMACAddress(t *testing.T) {
	lease, ok := parseLeaseLine(
		"1792400459 aa:bb:cc:dd:ee:ff 192.0.2.10 test-device 01:aa:bb:cc:dd:ee:ff",
		newReservations(),
	)
	if !ok {
		t.Fatal("expected IPv4 lease to parse")
	}
	if lease.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("MACAddress = %q, want uppercase MAC", lease.MACAddress)
	}
	if lease.IAID != "" || lease.DUID != "" {
		t.Fatalf("IPv4 lease should not have IAID/DUID, got IAID=%q DUID=%q", lease.IAID, lease.DUID)
	}
}

func TestParseIPv6LeaseKeepsIAIDAndDUIDSeparateFromMAC(t *testing.T) {
	lease, ok := parseLeaseLine(
		"1791753729 3055685611 2001:db8::41 test-device 00:04:12:34:56:78:9a:bc",
		newReservations(),
	)
	if !ok {
		t.Fatal("expected IPv6 lease to parse")
	}
	if lease.MACAddress != "" {
		t.Fatalf("IPv6 IAID must not be returned as MACAddress; got %q", lease.MACAddress)
	}
	if lease.IAID != "3055685611" {
		t.Fatalf("IAID = %q, want 3055685611", lease.IAID)
	}
	if lease.DUID != "00:04:12:34:56:78:9A:BC" {
		t.Fatalf("DUID = %q, want uppercase DUID", lease.DUID)
	}
}

func TestParseIPv6LeaseWithZeroIAID(t *testing.T) {
	lease, ok := parseLeaseLine(
		"0 0 fdf3::10:16:12:86 homepod 00:01:00:01:2b:c1:9d:2b:f4:34:f0:4e:65:e1",
		newReservations(),
	)
	if !ok {
		t.Fatal("expected IPv6 lease with IAID zero to parse")
	}
	if lease.MACAddress != "" || lease.IAID != "0" {
		t.Fatalf("unexpected IPv6 identity fields: MACAddress=%q IAID=%q", lease.MACAddress, lease.IAID)
	}
}
