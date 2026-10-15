package app

// Read and parse entries from the dnsmasq lease file.

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func formatLeaseEnd(v string) (string, bool) {
	if v == "0" {
		return "Never", true
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return "", false
	}
	return time.Unix(n, 0).Local().Format("2006-01-02 15:04:05"), true
}

func parseLeaseLine(line string, r *Reservations) (LeaseEntry, bool) {
	p := strings.Fields(line)
	if len(p) != 5 {
		return LeaseEntry{}, false
	}
	end, ok := formatLeaseEnd(p[0])
	if !ok {
		return LeaseEntry{}, false
	}
	expiry, _ := strconv.ParseInt(p[0], 10, 64)

	// dnsmasq lease formats differ by address family:
	// IPv4: expiry, MAC, IPv4, hostname, client-id
	// IPv6: expiry, IAID, IPv6, hostname, DUID
	macAddress := ""
	clientID := ""
	iaid := ""
	duid := ""
	if ip := net.ParseIP(p[2]); ip != nil && ip.To4() == nil {
		iaid = p[1]
		duid = strings.ToUpper(p[4])
	} else if ip != nil && ip.To4() != nil && macRE.MatchString(p[1]) {
		macAddress = strings.ToUpper(p[1])
		clientID = p[4]
	}

	return LeaseEntry{
		StaticIP:    p[0] == "0" || r.matches(p[1], p[2], p[3], p[4]),
		LeaseTime:   end,
		LeaseExpiry: expiry,
		MACAddress:  macAddress,
		ClientID:    clientID,
		IAID:        iaid,
		DUID:        duid,
		IPAddress:   p[2],
		Name:        p[3],
	}, true
}

func readLeaseEntries(leasePath, hostsPath string) ([]LeaseEntry, error) {
	f, err := os.Open(leasePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := readReservations(hostsPath)
	var leases []LeaseEntry
	s := bufio.NewScanner(f)
	for s.Scan() {
		if x, ok := parseLeaseLine(s.Text(), r); ok {
			leases = append(leases, x)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	LogDebugf("read DHCP lease entries count=%d", len(leases))
	return leases, nil
}

func readLeases(leasePath, hostsPath string) ([]LeaseEntry, error) {
	leases, err := readLeaseEntries(leasePath, hostsPath)
	if err != nil {
		return nil, err
	}
	addWebURLs(leases)
	LogDebugf("completed device reachability checks count=%d", len(leases))
	return leases, nil
}
