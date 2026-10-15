package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadIPv4ReservationMACs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dnsmasq.dhcphosts")
	content := `# Comment line
AA:BB:CC:DD:EE:01,192.0.2.10,device-one
AA:BB:CC:DD:EE:02,10.16.12.2,[::10:16:12:2],device-two
AA:BB:CC:DD:EE:03,[2001:db8::3],ipv6-only
id:client-no-mac,192.0.2.11,no-mac
AA:BB:CC:DD:EE:04,192.0.2.12,ignored,ignore
AA:BB:CC:DD:EE:01,192.0.2.10,duplicate # repeated reservation
AA:BB:CC:DD:EE:05,hostname-only
not-a-mac,192.0.2.13,invalid
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readIPv4ReservationMACs(path)
	if err != nil {
		t.Fatalf("readIPv4ReservationMACs returned error: %v", err)
	}
	want := []string{"AA:BB:CC:DD:EE:01", "AA:BB:CC:DD:EE:02"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIPv4ReservationMACs() = %#v, want %#v", got, want)
	}
}

func TestReadIPv4ReservationMACsMissingFile(t *testing.T) {
	_, err := readIPv4ReservationMACs(filepath.Join(t.TempDir(), "missing.dhcphosts"))
	if err == nil {
		t.Fatal("expected error for missing DHCP hosts file")
	}
}

func TestMarkAllIPv4ReservationsHandlerMarksOfflineReservations(t *testing.T) {
	dir := t.TempDir()
	hostsPath := filepath.Join(dir, "dnsmasq.dhcphosts")
	knownPath := filepath.Join(dir, "known-macs.json")
	content := "AA:BB:CC:DD:EE:10,192.0.2.10,offline-device\n" +
		"AA:BB:CC:DD:EE:11,[2001:db8::11],ipv6-only\n" +
		"AA:BB:CC:DD:EE:12,192.0.2.12,ignored,ignore\n"
	if err := os.WriteFile(hostsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	known := newKnownMACs(knownPath)
	handler := knownMACsHandler(Config{HostsPath: hostsPath}, known)
	request := httptest.NewRequest(http.MethodPost, "/known-macs", strings.NewReader(`{"action":"mark-all-ipv4-reservations"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		OK    bool     `json:"ok"`
		Count int      `json:"count"`
		MACs  []string `json:"macs"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.OK || body.Count != 1 || !reflect.DeepEqual(body.MACs, []string{"AA:BB:CC:DD:EE:10"}) {
		t.Fatalf("unexpected response: %#v", body)
	}
	if !known.has("AA:BB:CC:DD:EE:10") {
		t.Fatal("IPv4 reservation MAC was not stored as known")
	}
	if known.has("AA:BB:CC:DD:EE:11") || known.has("AA:BB:CC:DD:EE:12") {
		t.Fatal("IPv6-only or ignored reservation was incorrectly stored as known")
	}
}
