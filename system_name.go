package app

import (
	"os"
	"strings"
)

// detectNativeSystemName returns a useful host label for standalone/native builds.
// Docker builds deliberately do not call this, so the container's hostname is
// never mistaken for the physical host name.
func detectNativeSystemName() string {
	hostname, _ := os.Hostname()
	hostname = strings.TrimSpace(hostname)

	// Raspberry Pi and some other embedded Linux systems expose the hardware
	// model through the device tree. The file may contain a trailing NUL byte.
	if modelBytes, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		model := strings.TrimSpace(strings.TrimRight(string(modelBytes), "\x00"))
		if model != "" {
			if hostname != "" && !strings.EqualFold(hostname, model) {
				return model + " · " + hostname
			}
			return model
		}
	}

	if hostname != "" {
		return hostname
	}
	return "Linux-System"
}
