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
	if !cfg.NTFYEnabled || !cfg.NTFYNotifyUnknownIPv4 {
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(cfg.DHCPEventAction))
	// dnsmasq's "old" action restores an existing lease and is not a new device.
	if action != "add" {
		return nil
	}
	mac := strings.ToUpper(strings.TrimSpace(cfg.DHCPEventMAC))
	if !macRE.MatchString(mac) {
		return fmt.Errorf("invalid or missing MAC address")
	}
	ip := net.ParseIP(strings.TrimSpace(cfg.DHCPEventIP))
	if ip == nil || ip.To4() == nil {
		return nil
	}
	if err := validateNTFYURL(cfg.NTFYURL); err != nil {
		return err
	}
	if cfg.NTFYTimeoutSeconds < 1 || cfg.NTFYTimeoutSeconds > 120 {
		cfg.NTFYTimeoutSeconds = 10
	}

	db, err := openConfiguredDatabase(cfg.DatabaseType, cfg.DatabasePath, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresSSLMode, cfg.KnownMACsPath, cfg.UISettingsPath, cfg.DHCPEventsPath)
	if err != nil {
		return fmt.Errorf("open configured database: %w", err)
	}
	defer checkpointAndCloseDatabase(db)

	var found int
	err = db.QueryRow(adaptSQL(`SELECT 1 FROM known_devices WHERE mac=? COLLATE NOCASE`), mac).Scan(&found)
	if err == nil {
		return nil // explicitly known device
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("check known_devices: %w", err)
	}

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
		return fmt.Errorf("send ntfy notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy server returned HTTP %d", resp.StatusCode)
	}
	LogInfof("ntfy notification sent for unknown IPv4 device mac=%s ip=%s", mac, ip.To4().String())
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
