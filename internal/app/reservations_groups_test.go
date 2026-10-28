package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadReservationGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dhcphosts")
	content := `# LeaseEditor-Gruppe: Unterhaltung / Smart Home / Sonstiges
c0:95:6d:9c:c9:bd,10.16.12.10,[::10:16:12:10],apple-tv-micha,infinite
30:c9:ab:c9:a6:6e,10.16.12.20,[::10:16:12:20],brother

# LeaseEditor-Gruppe: Geräte von Monika / Alfred
1c:d6:be:e6:c6:96,10.16.12.100,[::10:16:12:100],sony-fernseher-kd-43x80j
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	groups := readReservationGroups(path)
	if got := groups["C0:95:6D:9C:C9:BD"]; got != "Unterhaltung / Smart Home / Sonstiges" {
		t.Fatalf("first group = %q", got)
	}
	if got := groups["30:C9:AB:C9:A6:6E"]; got != "Unterhaltung / Smart Home / Sonstiges" {
		t.Fatalf("second group = %q", got)
	}
	if got := groups["1C:D6:BE:E6:C6:96"]; got != "Geräte von Monika / Alfred" {
		t.Fatalf("third group = %q", got)
	}
	if _, ok := groups["AA:BB:CC:DD:EE:FF"]; ok {
		t.Fatal("unexpected group for unknown MAC")
	}
}

func TestReadReservationGroupsWithoutMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dhcphosts")
	if err := os.WriteFile(path, []byte("AA:BB:CC:DD:EE:FF,192.168.1.10,device\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if groups := readReservationGroups(path); len(groups) != 0 {
		t.Fatalf("expected no groups, got %#v", groups)
	}
}
