package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseLeaseHandlerUsesMatchedLeaseAndClientID(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "dnsmasq.leases")
	content := "1792400459 aa:bb:cc:dd:ee:ff 192.0.2.10 test-device 01:aa:bb:cc:dd:ee:ff\n"
	if err := os.WriteFile(leasePath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	called := false
	oldRunner := runDHCPRelease
	runDHCPRelease = func(_ context.Context, iface, ip, mac, clientID string) ([]byte, error) {
		called = true
		if iface != "eth1" || ip != "192.0.2.10" || mac != "AA:BB:CC:DD:EE:FF" || clientID != "01:aa:bb:cc:dd:ee:ff" {
			t.Fatalf("unexpected dhcp_release args: %q %q %q %q", iface, ip, mac, clientID)
		}
		return nil, nil
	}
	defer func() { runDHCPRelease = oldRunner }()

	cfg := Config{LeasePath: leasePath, HostsPath: filepath.Join(t.TempDir(), "missing-hosts"), DHCPInterface: "eth1"}
	req := httptest.NewRequest(http.MethodPost, "/release-lease", strings.NewReader(`{"ip":"192.0.2.10","mac":"aa:bb:cc:dd:ee:ff"}`))
	resp := httptest.NewRecorder()
	releaseLeaseHandler(cfg)(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%q; want 200", resp.Code, resp.Body.String())
	}
	if !called {
		t.Fatal("expected dhcp_release runner to be called")
	}
}

func TestReleaseLeaseHandlerRejectsIPv6AndUnknownLease(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "dnsmasq.leases")
	if err := os.WriteFile(leasePath, []byte("1792400459 aa:bb:cc:dd:ee:ff 192.0.2.10 test-device *\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{LeasePath: leasePath, HostsPath: filepath.Join(t.TempDir(), "missing-hosts"), DHCPInterface: "eth1"}

	for _, payload := range []string{
		`{"ip":"2001:db8::1","mac":"aa:bb:cc:dd:ee:ff"}`,
		`{"ip":"192.0.2.99","mac":"aa:bb:cc:dd:ee:ff"}`,
		`{"ip":"192.0.2.10","mac":"not-a-mac"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/release-lease", strings.NewReader(payload))
		resp := httptest.NewRecorder()
		releaseLeaseHandler(cfg)(resp, req)
		if resp.Code < 400 {
			t.Fatalf("payload %s: status = %d, want error", payload, resp.Code)
		}
	}
}
