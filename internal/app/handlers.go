package app

// HTTP handlers for the web interface and JSON API.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// newHTTPHandler wires the page, static assets, and API endpoints together.
func newHTTPHandler(cfg Config, known *KnownMACs, tmpl *template.Template, staticFS fs.FS, dnsmasqVersion string, nativeBuild bool) http.Handler {
	mux := http.NewServeMux()
	enableLeaseRelease := nativeBuild && validInterfaceName(cfg.DHCPInterface)
	mux.HandleFunc("/", homeHandler(cfg, tmpl, dnsmasqVersion, enableLeaseRelease))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/known-macs", knownMACsHandler(cfg, known))
	mux.HandleFunc("/leases", leasesHandler(cfg, known))
	if enableLeaseRelease {
		mux.HandleFunc("/release-lease", releaseLeaseHandler(cfg))
	}
	return mux
}

func homeHandler(cfg Config, tmpl *template.Template, dnsmasqVersion string, enableLeaseRelease bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := PageData{Version: cfg.AppVersion, ReleaseDate: cfg.ReleaseDate, RepoURL: cfg.RepoURL, SystemName: cfg.SystemName, DnsmasqVersion: dnsmasqVersion, EnableLeaseRelease: enableLeaseRelease}
		if err := tmpl.Execute(w, data); err != nil {
			LogErrorf("cannot render home page: %v", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// knownMACsHandler supports individual mark/unmark, reset, and bulk-marking
// currently detected and configured IPv4 DHCP reservations.
func knownMACsHandler(cfg Config, known *KnownMACs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			LogWarnf("rejected non-POST request to /known-macs method=%s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			MAC    string `json:"mac"`
			Action string `json:"action"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil {
			LogWarnf("rejected invalid JSON request to /known-macs")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		action := body.Action
		if action == "" {
			action = "mark"
		}
		var err error
		markedCount := 0
		var markedMACs []string
		switch action {
		case "mark":
			err = known.mark(body.MAC)
		case "unmark":
			err = known.unmark(body.MAC)
		case "reset":
			err = known.reset()
		case "mark-reservations":
			var leases []LeaseEntry
			leases, err = readLeaseEntries(cfg.LeasePath, cfg.HostsPath)
			if err == nil {
				macs := make([]string, 0, len(leases))
				for _, lease := range leases {
					mac := strings.TrimSpace(lease.MACAddress)
					if lease.StaticIP && macRE.MatchString(mac) {
						macs = append(macs, mac)
					}
				}
				markedCount, err = known.markMany(macs)
			}
		case "mark-all-ipv4-reservations":
			markedMACs, err = readIPv4ReservationMACs(cfg.HostsPath)
			if err != nil {
				LogErrorf("cannot read DHCP hosts file path=%s: %v", cfg.HostsPath, err)
				http.Error(w, "cannot read DHCP hosts file (check DNSMASQ_HOSTS_FILE and file permissions)", http.StatusInternalServerError)
				return
			}
			markedCount, err = known.markMany(markedMACs)
		default:
			LogWarnf("rejected unknown known-MAC action action=%q", action)
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
		if err != nil {
			LogErrorf("known-MAC action failed action=%s: %v", action, err)
			http.Error(w, "cannot save known MACs (check KNOWN_MACS_FILE permissions)", http.StatusInternalServerError)
			return
		}

		switch action {
		case "mark":
			LogInfof("device marked as known")
		case "unmark":
			LogInfof("device marked as new again")
		case "reset":
			LogInfof("remembered-device list reset")
		case "mark-reservations":
			LogInfof("currently detected DHCP reservations processed for remembered-device list count=%d", markedCount)
		case "mark-all-ipv4-reservations":
			LogInfof("configured IPv4 DHCP reservations processed for remembered-device list count=%d", markedCount)
		}

		w.Header().Set("Content-Type", "application/json")
		if action == "mark-reservations" {
			if err := json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": markedCount}); err != nil {
				LogDebugf("client disconnected while writing reservation action response: %v", err)
			}
			return
		}
		if action == "mark-all-ipv4-reservations" {
			if err := json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": markedCount, "macs": markedMACs}); err != nil {
				LogDebugf("client disconnected while writing all-IPv4-reservations response: %v", err)
			}
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

func leasesHandler(cfg Config, known *KnownMACs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leases, err := readLeases(cfg.LeasePath, cfg.HostsPath)
		if err != nil {
			LogErrorf("cannot read DHCP leases file path=%s: %v", cfg.LeasePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"leases file unavailable"}`))
			return
		}
		for i := range leases {
			mac := strings.TrimSpace(leases[i].MACAddress)
			ip := net.ParseIP(leases[i].IPAddress)
			// IPv6 entries without a valid MAC stay visible but are not labelled new.
			if ip != nil && ip.To4() == nil && !macRE.MatchString(mac) {
				leases[i].IsNew = false
				continue
			}
			leases[i].IsNew = !known.has(mac)
		}
		w.Header().Set("Content-Type", "application/json")
		LogDebugf("served lease list count=%d", len(leases))
		if err := json.NewEncoder(w).Encode(map[string]any{"leases": leases}); err != nil {
			LogDebugf("client disconnected while writing leases response: %v", err)
		}
	}
}

// runDHCPRelease is a seam for tests; production always calls the fixed
// dhcp_release utility through non-interactive sudo.
var runDHCPRelease = func(ctx context.Context, iface, ip, mac, clientID string) ([]byte, error) {
	return exec.CommandContext(ctx, "sudo", "-n", "/usr/bin/dhcp_release", iface, ip, mac, clientID).CombinedOutput()
}

// releaseLeaseHandler uses dnsmasq-utils' dhcp_release utility rather than
// editing the lease file directly. The endpoint only exists in native builds
// with an explicitly configured DHCP interface.
func releaseLeaseHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			IP  string `json:"ip"`
			MAC string `json:"mac"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ip := net.ParseIP(strings.TrimSpace(body.IP))
		mac := strings.ToUpper(strings.TrimSpace(body.MAC))
		if ip == nil || ip.To4() == nil || !macRE.MatchString(mac) {
			http.Error(w, "a valid IPv4 address and MAC address are required", http.StatusBadRequest)
			return
		}
		ipText := ip.To4().String()
		leases, err := readLeaseEntries(cfg.LeasePath, cfg.HostsPath)
		if err != nil {
			LogErrorf("cannot read DHCP leases before release: %v", err)
			http.Error(w, "cannot read DHCP lease file", http.StatusServiceUnavailable)
			return
		}
		var target *LeaseEntry
		for i := range leases {
			entry := &leases[i]
			parsed := net.ParseIP(entry.IPAddress)
			if parsed != nil && parsed.To4() != nil && parsed.To4().String() == ipText && strings.EqualFold(entry.MACAddress, mac) {
				target = entry
				break
			}
		}
		if target == nil {
			http.Error(w, "no matching IPv4 lease found; refresh the page and try again", http.StatusConflict)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		clientID := strings.TrimSpace(target.ClientID)
		if clientID == "" {
			clientID = "*"
		}
		output, err := runDHCPRelease(ctx, cfg.DHCPInterface, ipText, mac, clientID)
		if err != nil {
			LogErrorf("native IPv4 lease release failed interface=%s ip=%s mac=%s: %v output=%q", cfg.DHCPInterface, ipText, mac, err, strings.TrimSpace(string(output)))
			message := "could not release DHCP lease; install dnsmasq-utils and configure sudo permission for /usr/bin/dhcp_release"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				message = "DHCP lease release timed out"
			}
			http.Error(w, message, http.StatusBadGateway)
			return
		}
		LogInfof("native IPv4 DHCP lease release requested interface=%s ip=%s mac=%s", cfg.DHCPInterface, ipText, mac)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"ok":true}`)
	}
}

func validInterfaceName(value string) bool {
	if value == "" || len(value) > 15 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == ':' || r == '-') {
			return false
		}
	}
	return true
}
