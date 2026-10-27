package app

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// runDHCPNotification sends a notification for a newly observed lease.
// DHCP changes are detected by the in-process lease watcher; no dnsmasq script is required.
func runDHCPNotification(cfg Config) error {
	LogDebugf("ntfy event received action=%q mac=%q ip=%q hostname=%q enabled=%t notify_unknown_ipv4=%t db_type=%q",
		cfg.DHCPEventAction, cfg.DHCPEventMAC, cfg.DHCPEventIP, cfg.DHCPEventHostname,
		cfg.NTFYEnabled, cfg.NTFYNotifyUnknownIPv4, cfg.DatabaseType)
	if !cfg.NTFYEnabled {
		LogDebugf("ntfy skipped: ntfy_enabled=false")
		return nil
	}
	if !cfg.NTFYNotifyUnknownIPv4 {
		LogDebugf("ntfy skipped: ntfy_notify_unknown_ipv4=false")
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(cfg.DHCPEventAction))
	// dnsmasq's "old" action restores an existing lease and is not a new device.
	if action != "add" {
		LogDebugf("ntfy skipped: DHCP action %q is not add", action)
		return nil
	}
	mac := strings.ToUpper(strings.TrimSpace(cfg.DHCPEventMAC))
	if !macRE.MatchString(mac) {
		return fmt.Errorf("invalid or missing MAC address")
	}
	ip := net.ParseIP(strings.TrimSpace(cfg.DHCPEventIP))
	if ip == nil || ip.To4() == nil {
		LogDebugf("ntfy skipped: invalid/non-IPv4 address %q", cfg.DHCPEventIP)
		return nil
	}
	if err := validateNTFYURL(cfg.NTFYURL); err != nil {
		return fmt.Errorf("validate ntfy_url: %w", err)
	}
	LogDebugf("ntfy accepted event mac=%s ip=%s endpoint=%s", mac, ip.To4(), cfg.NTFYURL)
	if cfg.NTFYTimeoutSeconds < 1 || cfg.NTFYTimeoutSeconds > 120 {
		cfg.NTFYTimeoutSeconds = 10
	}

	db, err := openConfiguredDatabase(cfg.DatabaseType, cfg.DatabasePath, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresSSLMode, cfg.KnownMACsPath, cfg.UISettingsPath, cfg.DHCPEventsPath)
	if err != nil {
		return fmt.Errorf("open configured database type=%q: %w", cfg.DatabaseType, err)
	}
	defer checkpointAndCloseDatabase(db)
	LogDebugf("ntfy database connection established type=%q", cfg.DatabaseType)

	var found int
	query := adaptSQL(`SELECT 1 FROM known_devices WHERE mac=? COLLATE NOCASE`)
	LogDebugf("ntfy database query: checking known_devices for mac=%s", mac)
	err = db.QueryRow(query, mac).Scan(&found)
	if err == nil {
		LogInfof("ntfy skipped: device already known mac=%s", mac)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("query known_devices for mac %s: %w", mac, err)
	}
	LogInfof("ntfy device is unknown mac=%s ip=%s; sending push", mac, ip.To4())

	hostname := strings.TrimSpace(cfg.DHCPEventHostname)
	if hostname == "" || hostname == "*" {
		hostname = "(unbekannt)"
	}
	body := fmt.Sprintf("Neues unbekanntes IPv4-Gerät erkannt\nHostname: %s\nIPv4-Adresse: %s\nMAC-Adresse: %s\nDHCP-Ereignis: add", hostname, ip.To4().String(), mac)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.NTFYTimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.NTFYURL, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("create ntfy request: %w", err)
	}
	req.Header.Set("Title", "LeaseLens: neues IPv4-Gerät")
	req.Header.Set("Priority", validNTFYPriority(cfg.NTFYPriority))
	req.Header.Set("Tags", "computer,warning")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.NTFYToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.NTFYToken)
	}
	client := &http.Client{Timeout: time.Duration(cfg.NTFYTimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send ntfy notification to %s: %w", cfg.NTFYURL, err)
	}
	defer resp.Body.Close()
	LogDebugf("ntfy HTTP response status=%d", resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy server returned HTTP %d", resp.StatusCode)
	}
	LogInfof("ntfy notification sent successfully mac=%s ip=%s http_status=%d", mac, ip.To4().String(), resp.StatusCode)
	return nil
}

func validNTFYPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "min", "1":
		return "1"
	case "low", "2":
		return "2"
	case "default", "3":
		return "3"
	case "high", "4":
		return "4"
	case "urgent", "5":
		return "5"
	default:
		return "4"
	}
}

// validateNTFYURL is kept separate so config validation can be tested without network access.
func validateNTFYURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("ntfy_url must be an absolute HTTP(S) URL")
	}
	return nil
}

// startLeaseNotificationWatcher polls dnsmasq's lease file from inside the main Go process.
// It establishes a baseline at startup, then processes only newly appearing MAC/IP leases.
func startLeaseNotificationWatcher(ctx context.Context, cfg Config) {
	if !cfg.NTFYEnabled {
		LogInfof("ntfy watcher disabled (ntfy_enabled=false)")
		return
	}
	LogInfof("ntfy lease watcher starting lease_file=%s interval=3s", cfg.LeasePath)
	go func() {
		known := map[string]bool{}
		leases, err := readLeaseEntries(cfg.LeasePath, cfg.HostsPath)
		if err != nil {
			LogWarnf("ntfy watcher cannot read initial lease file %q: %v", cfg.LeasePath, err)
		} else {
			for _, lease := range leases {
				if lease.MACAddress != "" && net.ParseIP(lease.IPAddress) != nil {
					known[strings.ToUpper(lease.MACAddress)+"|"+lease.IPAddress] = true
				}
			}
			LogInfof("ntfy watcher baseline loaded leases=%d tracked=%d", len(leases), len(known))
		}
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				LogInfof("ntfy lease watcher stopped")
				return
			case <-ticker.C:
				leases, err := readLeaseEntries(cfg.LeasePath, cfg.HostsPath)
				if err != nil {
					LogWarnf("ntfy watcher cannot read lease file %q: %v", cfg.LeasePath, err)
					continue
				}
				for _, lease := range leases {
					mac := strings.ToUpper(strings.TrimSpace(lease.MACAddress))
					ip := strings.TrimSpace(lease.IPAddress)
					parsedIP := net.ParseIP(ip)
					if mac == "" || parsedIP == nil || parsedIP.To4() == nil {
						continue
					}
					key := mac + "|" + ip
					if known[key] {
						continue
					}
					known[key] = true
					// If the same MAC changes IP, treat that as a new lease observation too.
					eventCfg := cfg
					eventCfg.DHCPEventAction = "add"
					eventCfg.DHCPEventMAC = mac
					eventCfg.DHCPEventIP = ip
					eventCfg.DHCPEventHostname = lease.Name
					LogInfof("ntfy watcher detected newly observed DHCP lease mac=%s ip=%s hostname=%q", mac, ip, lease.Name)
					if err := runDHCPNotification(eventCfg); err != nil {
						LogErrorf("ntfy notification failed mac=%s ip=%s: %v", mac, ip, err)
					}
				}
			}
		}
	}()
}
