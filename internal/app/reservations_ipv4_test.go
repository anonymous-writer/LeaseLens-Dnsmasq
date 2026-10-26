package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadIPv4ReservationMACs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.dhcphosts")
	contents := `# sample hosts file
AA:BB:CC:DD:EE:01,workstation,10.16.12.150
AA:BB:CC:DD:EE:02,phone,10.16.12.151 # comment
AA:BB:CC:DD:EE:01,duplicate,10.16.12.152
AA:BB:CC:DD:EE:03,ipv6-only,[fdf3::3]
AA:BB:CC:DD:EE:04,ignored,10.16.12.153,ignore
invalid-mac,host,10.16.12.154
AA:BB:CC:DD:EE:05,10.16.12.155
`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readIPv4ReservationMACs(path)
	if err != nil {
		t.Fatalf("readIPv4ReservationMACs() error = %v", err)
	}
	want := []string{"AA:BB:CC:DD:EE:01", "AA:BB:CC:DD:EE:02", "AA:BB:CC:DD:EE:05"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIPv4ReservationMACs() = %#v, want %#v", got, want)
	}
}

func TestReadIPv4ReservationMACsMissingFile(t *testing.T) {
	_, err := readIPv4ReservationMACs(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected an error for a missing DHCP hosts file")
	}
}
