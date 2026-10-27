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

// runDHCPNotification is invoked by the same executable from dnsmasq's dhcp-script.
// It reads known_devices from the configured database; no JSON device list is used.
func runDHCPNotification(cfg Config) error {
	action := strings.ToLower(strings.TrimSpace(cfg.DHCPEventAction))
	mac := strings.ToUpper(strings.TrimSpace(cfg.DHCPEventMAC))
	ip := net.ParseIP(strings.TrimSpace(cfg.DHCPEventIP))
	ipText := strings.TrimSpace(cfg.DHCPEventIP)
	if ip != nil && ip.To4() != nil {
		ipText = ip.To4().String()
	}
	if cfg.NTFYDebug || cfg.DatabaseDebug {
		LogDebugf("DHCP event entered action=%q mac=%q ip=%q hostname=%q ntfy_enabled=%t",
			action, mac, ipText, cfg.DHCPEventHostname, cfg.NTFYEnabled)
	}
	if !macRE.MatchString(mac) {
		return fmt.Errorf("DHCP event: invalid or missing MAC address (received %q)", cfg.DHCPEventMAC)
	}

	// Always persist the DHCP event, even when ntfy is disabled. This makes
	// the event history independent from push-notification preferences.
	if cfg.DatabaseDebug {
		LogDebugf("database: opening configured database type=%q for DHCP event", cfg.DatabaseType)
	}
	db, err := openConfiguredDatabase(cfg.DatabaseType, cfg.DatabasePath, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresSSLMode, cfg.KnownMACsPath, cfg.UISettingsPath, cfg.DHCPEventsPath)
	if err != nil {
		return fmt.Errorf("database: open configured database type=%q: %w", cfg.DatabaseType, err)
	}
	defer checkpointAndCloseDatabase(db)
	if cfg.DatabaseDebug {
		LogDebugf("database: connection ready type=%q", cfg.DatabaseType)
	}
	if strings.TrimSpace(cfg.DHCPEventsPath) != "" {
		if err := importLegacyDHCPEvents(db, cfg.DHCPEventsPath); err != nil {
			LogWarnf("database: legacy DHCP log import failed path=%s: %v", cfg.DHCPEventsPath, err)
		}
	}
	if err := recordDHCPEvent(db, action, mac, ipText, cfg.DHCPEventHostname, cfg.DHCPEventsMaxEntries); err != nil {
		return fmt.Errorf("database: persist DHCP event: %w", err)
	}

	if action != "add" {
		LogDebugf("ntfy: skipped action=%q; only action=add triggers unknown-device alerts", action)
		return nil
	}
	if ip == nil || ip.To4() == nil {
		LogDebugf("ntfy: skipped invalid/non-IPv4 address %q", cfg.DHCPEventIP)
		return nil
	}
	if !cfg.NTFYEnabled {
		LogInfof("ntfy: skipped DHCP event because ntfy_enabled=false")
		return nil
	}
	if !cfg.NTFYNotifyUnknownIPv4 {
		LogInfof("ntfy: skipped DHCP event because ntfy_notify_unknown_ipv4=false")
		return nil
	}
	if err := validateNTFYURL(cfg.NTFYURL); err != nil {
		return fmt.Errorf("ntfy: invalid ntfy_url: %w", err)
	}
	if cfg.NTFYTimeoutSeconds < 1 || cfg.NTFYTimeoutSeconds > 120 {
		cfg.NTFYTimeoutSeconds = 10
	}

	if cfg.DatabaseDebug {
		LogDebugf("database: querying known_devices for mac=%s", mac)
	}
	var found int
	err = db.QueryRow(adaptSQL(`SELECT 1 FROM known_devices WHERE mac=? COLLATE NOCASE`), mac).Scan(&found)
	if err == nil {
		LogInfof("ntfy: skipped known device mac=%s", mac)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("database: query known_devices for mac=%s: %w", mac, err)
	}
	LogInfof("ntfy: device is unknown mac=%s ip=%s", mac, ip.To4().String())

	hostname := strings.TrimSpace(cfg.DHCPEventHostname)
	if hostname == "" || hostname == "*" {
		hostname = "(unbekannt)"
	}
	body := fmt.Sprintf("Neues unbekanntes IPv4-Gerät erkannt\nHostname: %s\nIPv4-Adresse: %s\nMAC-Adresse: %s\nDHCP-Ereignis: add", hostname, ip.To4().String(), mac)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.NTFYTimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.NTFYURL, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Title", "LeaseLens: neues IPv4-Gerät")
	req.Header.Set("Priority", validNTFYPriority(cfg.NTFYPriority))
	req.Header.Set("Tags", "computer,warning")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.NTFYToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.NTFYToken)
	}
	client := &http.Client{Timeout: time.Duration(cfg.NTFYTimeoutSeconds) * time.Second}
	if cfg.NTFYDebug {
		LogDebugf("ntfy: sending HTTP POST timeout=%ds priority=%s", cfg.NTFYTimeoutSeconds, validNTFYPriority(cfg.NTFYPriority))
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send notification to %s: %w", cfg.NTFYURL, err)
	}
	defer resp.Body.Close()
	if cfg.NTFYDebug {
		LogDebugf("ntfy: HTTP response status=%d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: server returned HTTP %d", resp.StatusCode)
	}
	LogInfof("ntfy: notification sent successfully for unknown IPv4 device mac=%s ip=%s http_status=%d", mac, ip.To4().String(), resp.StatusCode)
	return nil
}

// notifyDeviceMarkedNew sends a push when a user explicitly changes a device
// back to the NEW state in the UI. DHCP add events for unknown devices are
// handled by runDHCPNotification; this covers the manual status transition.
func notifyDeviceMarkedNew(cfg Config, rawMAC string) error {
	if !cfg.NTFYEnabled || !cfg.NTFYNotifyUnknownIPv4 {
		return nil
	}
	mac := strings.ToUpper(strings.TrimSpace(rawMAC))
	if !macRE.MatchString(mac) {
		return fmt.Errorf("invalid MAC address %q", rawMAC)
	}
	if err := validateNTFYURL(cfg.NTFYURL); err != nil {
		return fmt.Errorf("invalid ntfy_url: %w", err)
	}
	timeout := cfg.NTFYTimeoutSeconds
	if timeout < 1 || timeout > 120 {
		timeout = 10
	}
	hostname, ipAddress := "(unbekannt)", "(nicht verfügbar)"
	if leases, err := readLeaseEntries(cfg.LeasePath, cfg.HostsPath); err == nil {
		for _, lease := range leases {
			if strings.EqualFold(strings.TrimSpace(lease.MACAddress), mac) {
				if name := strings.TrimSpace(lease.Name); name != "" && name != "*" {
					hostname = name
				}
				if ip := net.ParseIP(strings.TrimSpace(lease.IPAddress)); ip != nil && ip.To4() != nil {
					ipAddress = ip.To4().String()
				}
				break
			}
		}
	}
	body := fmt.Sprintf("Gerätestatus auf NEU gesetzt\nHostname: %s\nIPv4-Adresse: %s\nMAC-Adresse: %s\nEreignis: Status manuell auf NEU gesetzt", hostname, ipAddress, mac)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.NTFYURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Title", "LeaseLens: Gerät auf NEU gesetzt")
	req.Header.Set("Priority", validNTFYPriority(cfg.NTFYPriority))
	req.Header.Set("Tags", "computer,warning")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.NTFYToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.NTFYToken)
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned HTTP %d", resp.StatusCode)
	}
	LogInfof("ntfy: notification sent for device manually marked NEW mac=%s http_status=%d", mac, resp.StatusCode)
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
