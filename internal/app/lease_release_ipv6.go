package app

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// removeIPv6LeasesForUniqueHostname removes IPv6 lease records only when the
// hostname belongs to exactly one IPv4 lease matching ip and mac. dnsmasq's
// IPv6 lease format contains IAID/DUID, not the device MAC, so hostname is
// the only cross-family link available in this lease file.
func removeIPv6LeasesForUniqueHostname(path, hostname, ip, mac string) (int, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return 0, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var lines []string
	targetFound := false
	ipv4NameCount := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		fields := strings.Fields(line)
		if len(fields) != 5 || !strings.EqualFold(fields[3], hostname) {
			continue
		}
		addr := net.ParseIP(fields[2])
		if addr == nil {
			continue
		}
		if addr.To4() != nil {
			ipv4NameCount++
			if addr.To4().String() == ip && strings.EqualFold(fields[1], mac) {
				targetFound = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	// Do not guess if hostname is shared or the IPv4 lease disappeared.
	if !targetFound || ipv4NameCount != 1 {
		return 0, nil
	}

	var kept []string
	removed := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 5 && strings.EqualFold(fields[3], hostname) {
			addr := net.ParseIP(fields[2])
			if addr != nil && addr.To4() == nil {
				removed++
				continue
			}
		}
		kept = append(kept, line)
	}
	if removed == 0 {
		return 0, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dnsmasq.leases-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return 0, err
	}
	for _, line := range kept {
		if _, err := fmt.Fprintln(tmp, line); err != nil {
			tmp.Close()
			return 0, err
		}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return 0, err
	}
	return removed, nil
}
