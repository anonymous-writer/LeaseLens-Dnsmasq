package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveIPv6LeasesForUniqueHostname(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.leases")
	content := strings.Join([]string{
		"1792400459 aa:bb:cc:dd:ee:ff 192.0.2.10 test-device 01:aa:bb:cc:dd:ee:ff",
		"1792400459 00010001 2001:db8::10 test-device 00:01:00:01:aa:bb:cc:dd",
		"1792400459 00010002 2001:db8::11 other-device 00:01:00:02:aa:bb:cc:dd",
		"1792400459 aa:bb:cc:dd:ee:01 192.0.2.11 other-device *",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := removeIPv6LeasesForUniqueHostname(path, "test-device", "192.0.2.10", "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d, want 1", removed)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "2001:db8::10") {
		t.Fatal("target IPv6 lease was not removed")
	}
	if !strings.Contains(string(got), "2001:db8::11") || !strings.Contains(string(got), "192.0.2.11") {
		t.Fatal("unrelated lease was removed")
	}
}

func TestRemoveIPv6LeasesDoesNotGuessSharedHostname(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.leases")
	content := "1792400459 aa:bb:cc:dd:ee:ff 192.0.2.10 same *\n" +
		"1792400459 aa:bb:cc:dd:ee:01 192.0.2.11 same *\n" +
		"1792400459 00010001 2001:db8::10 same 00:01:00:01:aa:bb:cc:dd\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	removed, err := removeIPv6LeasesForUniqueHostname(path, "same", "192.0.2.10", "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("removed=%d, want 0 for ambiguous hostname", removed)
	}
}

func TestRemoveIPv6LeasesAfterIPv4RecordWasAlreadyReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.leases")
	content := "1792151602 0 fdf3::10:16:12:73 apple-watch-theresa 00:03:00:01:2a:5a:ca:4e:2c:2a\n" +
		"1792782639 a4:f9:33:ca:eb:b7 10.16.12.43 lenovo-ideapad-s145-15iil 01:a4:f9:33:ca:eb:b7\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := removeIPv6LeasesForUniqueHostname(path, "apple-watch-theresa", "10.16.12.73", "2A:5A:CA:4E:2C:2A")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d, want 1 after IPv4 record is already gone", removed)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "fdf3::10:16:12:73") {
		t.Fatal("target IPv6 lease was not removed")
	}
	if !strings.Contains(string(got), "lenovo-ideapad-s145-15iil") {
		t.Fatal("unrelated lease was removed")
	}
}
