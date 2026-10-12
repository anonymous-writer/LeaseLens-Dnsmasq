package app

import (
	"os/exec"
	"strings"
)

// detectDnsmasqVersion queries the native dnsmasq executable once at startup.
// It is called only by builds explicitly marked as native in the ARM64 workflow.
func detectDnsmasqVersion() string {
	output, err := exec.Command("dnsmasq", "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return parseDnsmasqVersion(string(output))
}

func parseDnsmasqVersion(output string) string {
	const marker = "dnsmasq version "
	lower := strings.ToLower(output)
	idx := strings.Index(lower, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(output[idx+len(marker):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimRight(fields[0], ",;")
}
