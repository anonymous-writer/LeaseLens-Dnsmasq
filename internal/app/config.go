package app

// Load configuration from environment variables and command-line flags.

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return d
}

func envDuration(k string, d time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if x, e := time.ParseDuration(v); e == nil {
			return x
		}
		if n, e := strconv.ParseFloat(v, 64); e == nil {
			return time.Duration(n * float64(time.Second))
		}
	}
	return d
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Config contains paths, server settings, and build metadata.
type Config struct {
	Host          string
	Port          string
	LeasePath     string
	HostsPath     string
	KnownMACsPath string
	AppVersion    string
	ReleaseDate   string
	RepoURL       string
}

// loadConfig reads runtime options from environment variables.
func loadConfig() Config {
	return Config{
		Host:          getenv("HOST", "0.0.0.0"),
		Port:          getenv("PORT", "5000"),
		LeasePath:     getenv("DNSMASQ_LEASES_FILE", "/var/lib/dnsmasq/dnsmasq.leases"),
		HostsPath:     getenv("DNSMASQ_HOSTS_FILE", "/var/lib/dnsmasq/dnsmasq.dhcphosts"),
		KnownMACsPath: getenv("KNOWN_MACS_FILE", "/data/known-macs.json"),
		AppVersion:    getenv("APP_VERSION", "dev"),
		ReleaseDate:   getenv("APP_RELEASE_DATE", ""),
		RepoURL:       getenv("REPO_URL", "https://github.com/anonymous-writer/dnsmasq-leases-ui"),
	}
}

// loadConfigFromArgs keeps environment-variable compatibility while allowing
// command-line flags to override environment values.
func loadConfigFromArgs(args []string) (Config, string, error) {
	cfg := loadConfig()
	logLevel := getenv("LOG_LEVEL", "INFO")
	flags := flag.NewFlagSet("dnsmasq-leases-ui", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: dnsmasq-leases-ui [options]")
		fmt.Fprintln(flags.Output(), "Command-line options override their corresponding environment variables.")
		fmt.Fprintln(flags.Output())
		fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}

	flags.StringVar(&cfg.Host, "host", cfg.Host, "Address to listen on (environment: HOST)")
	flags.StringVar(&cfg.Port, "port", cfg.Port, "TCP port to listen on (environment: PORT)")
	flags.StringVar(&cfg.LeasePath, "leases-file", cfg.LeasePath, "Path to dnsmasq lease file (environment: DNSMASQ_LEASES_FILE)")
	flags.StringVar(&cfg.HostsPath, "hosts-file", cfg.HostsPath, "Path to optional dnsmasq DHCP hosts file (environment: DNSMASQ_HOSTS_FILE)")
	flags.StringVar(&cfg.KnownMACsPath, "known-macs-file", cfg.KnownMACsPath, "Path to persistent remembered-MAC JSON file (environment: KNOWN_MACS_FILE)")
	flags.StringVar(&logLevel, "log-level", logLevel, "Log level: DEBUG, INFO, WARN, or ERROR (environment: LOG_LEVEL)")

	if err := flags.Parse(args); err != nil {
		return Config{}, "", err
	}
	if flags.NArg() > 0 {
		return Config{}, "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	return cfg, logLevel, nil
}
