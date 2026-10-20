package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// readDHCPEventState reads the bounded tab-separated log written by the
// dnsmasq dhcp-script. It never performs network probes.
type dhcpEventState struct {
	status  string
	lastAdd time.Time
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
		// Keep latest event status by timestamp; "old" is dnsmasq restoring
		// an existing lease and is not treated as a fresh login.
		state.status = action
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
