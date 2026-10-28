package app

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// readDHCPEventState reads the bounded tab-separated log written by the
// dnsmasq dhcp-script. It never performs network probes.
type dhcpEventState struct {
	status    string
	lastEvent time.Time
	lastAdd   time.Time
}

func readDHCPEventState(path string) map[string]dhcpEventState {
	out := make(map[string]dhcpEventState)
	if strings.TrimSpace(path) == "" {
		return out
	}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 4 {
			continue
		}
		ts, err := time.Parse("2006-01-02 15:04:05 -0700", fields[0])
		if err != nil {
			continue
		}
		action := strings.ToLower(fields[1])
		mac := strings.ToUpper(strings.TrimSpace(fields[3]))
		if mac == "" {
			continue
		}
		state := out[mac]
		// Track the latest recorded event separately from a real "add" login.
		// "old" restores an existing lease and is not a fresh login.
		if state.lastEvent.IsZero() || ts.After(state.lastEvent) {
			state.status = action
			state.lastEvent = ts
		}
		if action == "add" && ts.After(state.lastAdd) {
			state.lastAdd = ts
		}
		out[mac] = state
	}
	return out
}

func applyDHCPEventState(leases []LeaseEntry, path string, now time.Time) {
	states := readDHCPEventState(path)
	for i := range leases {
		mac := strings.ToUpper(strings.TrimSpace(leases[i].MACAddress))
		state, ok := states[mac]
		if !ok {
			continue
		}
		leases[i].DHCPEventStatus = state.status
		if !state.lastEvent.IsZero() {
			leases[i].LastDHCPEventAge = formatElapsed(now.Sub(state.lastEvent))
		}
		if !state.lastAdd.IsZero() {
			leases[i].LastDHCPAddAt = state.lastAdd.Local().Format("2006-01-02 15:04:05")
			leases[i].LastDHCPAddAge = formatElapsed(now.Sub(state.lastAdd))
		}
	}
}

func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// recordDHCPEvent stores a dnsmasq event in the database and trims oldest rows
// so the table never grows beyond maxEntries.
func recordDHCPEvent(db *sql.DB, action, mac, ip, hostname string, maxEntries int) error {
	if maxEntries < 1 {
		maxEntries = 100
	}
	now := time.Now()
	action = strings.ToLower(strings.TrimSpace(action))
	mac = strings.ToUpper(strings.TrimSpace(mac))
	ip = strings.TrimSpace(ip)
	hostname = strings.TrimSpace(hostname)
	keyInput := fmt.Sprintf("%s\t%s\t%s\t%s\t%s", now.Format(time.RFC3339Nano), action, ip, mac, hostname)
	sum := sha256.Sum256([]byte(keyInput))
	key := hex.EncodeToString(sum[:])
	_, err := db.Exec(adaptSQL(`INSERT OR IGNORE INTO dhcp_events(event_key,event_time,action,ip_address,mac,hostname) VALUES(?,?,?,?,?,?)`),
		key, now.Format(time.RFC3339), action, ip, mac, hostname)
	if err != nil {
		return fmt.Errorf("insert DHCP event: %w", err)
	}
	if err := pruneDHCPEvents(db, maxEntries); err != nil {
		return err
	}
	LogDebugf("database: recorded DHCP event action=%s mac=%s ip=%s retained_limit=%d", action, mac, ip, maxEntries)
	return nil
}

// pruneDHCPEvents retains only the newest maxEntries records.
func pruneDHCPEvents(db *sql.DB, maxEntries int) error {
	if maxEntries < 1 {
		maxEntries = 100
	}
	_, err := db.Exec(adaptSQL(`DELETE FROM dhcp_events WHERE id NOT IN (SELECT id FROM dhcp_events ORDER BY event_time DESC, id DESC LIMIT ?)`), maxEntries)
	if err != nil {
		return fmt.Errorf("prune DHCP event history to %d entries: %w", maxEntries, err)
	}
	return nil
}

// applyDHCPEventStateFromDB hydrates lease event details from the persistent database.
func applyDHCPEventStateFromDB(leases []LeaseEntry, db *sql.DB, now time.Time) error {
	rows, err := db.Query(`SELECT action, event_time, mac FROM dhcp_events ORDER BY event_time DESC, id DESC`)
	if err != nil {
		return fmt.Errorf("read DHCP events from database: %w", err)
	}
	defer rows.Close()
	states := make(map[string]dhcpEventState)
	for rows.Next() {
		var action, eventTime, mac string
		if err := rows.Scan(&action, &eventTime, &mac); err != nil {
			return fmt.Errorf("scan DHCP event: %w", err)
		}
		ts, err := parseDHCPEventTime(eventTime)
		if err != nil {
			continue
		}
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if mac == "" {
			continue
		}
		state, exists := states[mac]
		if !exists {
			state.status = strings.ToLower(action)
			state.lastEvent = ts
		}
		if strings.EqualFold(action, "add") && ts.After(state.lastAdd) {
			state.lastAdd = ts
		}
		states[mac] = state
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate DHCP events: %w", err)
	}
	for i := range leases {
		mac := strings.ToUpper(strings.TrimSpace(leases[i].MACAddress))
		state, ok := states[mac]
		if !ok {
			continue
		}
		leases[i].DHCPEventStatus = state.status
		if !state.lastEvent.IsZero() {
			leases[i].LastDHCPEventAge = formatElapsed(now.Sub(state.lastEvent))
		}
		if !state.lastAdd.IsZero() {
			leases[i].LastDHCPAddAt = state.lastAdd.Local().Format("2006-01-02 15:04:05")
			leases[i].LastDHCPAddAge = formatElapsed(now.Sub(state.lastAdd))
		}
	}
	return nil
}

func parseDHCPEventTime(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05 -0700", value)
}
