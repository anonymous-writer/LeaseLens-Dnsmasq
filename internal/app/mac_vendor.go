package app

// Local MAC-address vendor lookup using an IEEE OUI database.
// The database is loaded once and cached for the lifetime of the process.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	ouiOnce sync.Once
	ouiMap  map[string]string
	ouiLine = regexp.MustCompile(`(?i)^([0-9a-f]{6}|[0-9a-f]{2}[:-][0-9a-f]{2}[:-][0-9a-f]{2})\s+(?:\(hex\)\s+)?(.+?)\s*$`)
)

func macVendor(mac string) (vendor string, private bool) {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if !macRE.MatchString(mac) {
		return "", false
	}
	firstByte := mac[:2]
	// Locally administered addresses are commonly randomized by phones and laptops.
	// Their OUI does not reliably identify the hardware manufacturer.
	if value, err := parseHexByte(firstByte); err == nil && value&0x02 != 0 {
		return "", true
	}
	prefix := strings.ReplaceAll(mac, ":", "")[:6]
	return loadOUIDatabase()[prefix], false
}

func parseHexByte(value string) (byte, error) {
	var parsed byte
	for _, r := range value {
		parsed <<= 4
		switch {
		case r >= '0' && r <= '9':
			parsed |= byte(r - '0')
		case r >= 'A' && r <= 'F':
			parsed |= byte(r-'A') + 10
		default:
			return 0, os.ErrInvalid
		}
	}
	return parsed, nil
}

func loadOUIDatabase() map[string]string {
	ouiOnce.Do(func() {
		ouiMap = make(map[string]string)
		candidates := []string{}
		if configured := strings.TrimSpace(os.Getenv("OUI_DATABASE_FILE")); configured != "" {
			candidates = append(candidates, configured)
		}
		candidates = append(candidates,
			"/usr/share/ieee-data/oui.txt",
			"/var/lib/ieee-data/oui.txt",
			"/usr/share/arp-scan/ieee-oui.txt",
			"/etc/arp-scan/mac-vendor.txt",
			"/usr/share/misc/oui.txt",
			"/usr/share/hwdata/oui.txt",
		)
		for _, candidate := range candidates {
			entries, err := readOUIDatabase(candidate)
			if err != nil || len(entries) == 0 {
				continue
			}
			ouiMap = entries
			LogInfof("loaded MAC vendor database path=%s entries=%d", filepath.Clean(candidate), len(entries))
			return
		}
		LogWarnf("MAC vendor database not found; install ieee-data or set OUI_DATABASE_FILE")
	})
	return ouiMap
}

func readOUIDatabase(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make(map[string]string)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		match := ouiLine.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		prefix := strings.NewReplacer(":", "", "-", "").Replace(strings.ToUpper(match[1]))
		vendor := strings.TrimSpace(match[2])
		if strings.EqualFold(vendor, "(hex)") || vendor == "" {
			continue
		}
		entries[prefix] = vendor
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
