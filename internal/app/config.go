package app

// Load configuration from environment variables and command-line flags.

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Linker-overridable defaults for native builds. Runtime configuration can override these.
var buildVersion = "dev"
var buildReleaseDate = ""

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
	WorkingDir       string
	DatabasePath     string
	DatabaseType     string
	PostgresHost     string
	PostgresPort     string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string
	Host             string
	Port             string
	LeasePath        string
	HostsPath        string
	KnownMACsPath    string
	UISettingsPath   string
	DHCPEventsPath   string
	DHCPInterface    string
	AppVersion       string
	ReleaseDate      string
	RepoURL          string
	SystemName       string
}

// loadConfig reads runtime options from environment variables.
func loadConfig() Config {
	return Config{
		WorkingDir:       getenv("WORKING_DIR", ""),
		DatabasePath:     getenv("DATABASE_PATH", "/data/leaselens.db"),
		DatabaseType:     getenv("DATABASE_TYPE", "sqlite"),
		PostgresHost:     getenv("POSTGRES_HOST", "127.0.0.1"),
		PostgresPort:     getenv("POSTGRES_PORT", "5432"),
		PostgresDatabase: getenv("POSTGRES_DATABASE", "leaselens"),
		PostgresUser:     getenv("POSTGRES_USER", "leaselens"),
		PostgresPassword: getenv("POSTGRES_PASSWORD", ""),
		PostgresSSLMode:  getenv("POSTGRES_SSLMODE", "require"),
		Host:             getenv("HOST", "0.0.0.0"),
		Port:             getenv("PORT", "5000"),
		LeasePath:        getenv("DNSMASQ_LEASES_FILE", "/var/lib/dnsmasq/dnsmasq.leases"),
		HostsPath:        getenv("DNSMASQ_HOSTS_FILE", "/var/lib/dnsmasq/dnsmasq.dhcphosts"),
		KnownMACsPath:    getenv("KNOWN_MACS_FILE", "/data/known-macs.json"),
		UISettingsPath:   getenv("UI_SETTINGS_FILE", "/data/ui-settings.json"),
		DHCPEventsPath:   getenv("DNSMASQ_EVENTS_FILE", ""),
		DHCPInterface:    getenv("DNSMASQ_INTERFACE", ""),
		AppVersion:       getenv("APP_VERSION", buildVersion),
		ReleaseDate:      getenv("APP_RELEASE_DATE", buildReleaseDate),
		RepoURL:          getenv("REPO_URL", "https://github.com/anonymous-writer/leaselens-dnsmasq"),
		SystemName:       getenv("SYSTEM_NAME", ""),
	}
}

// loadConfigFromArgs keeps environment-variable compatibility while allowing
// values from a YAML-style config file and command-line flags. Precedence is:
// defaults < environment variables < config file < command-line flags.
func loadConfigFromArgs(args []string) (Config, string, error) {
	cfg := loadConfig()
	logLevel := getenv("LOG_LEVEL", "INFO")
	configPath := getenv("CONFIG_FILE", "")

	// Discover --config before parsing the rest of the options so file values
	// can become defaults that explicit command-line options override.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return Config{}, "", fmt.Errorf("--config requires a file path")
			}
			i++
			configPath = args[i]
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		}
	}

	if configPath != "" {
		if err := applyConfigFile(configPath, &cfg, &logLevel); err != nil {
			return Config{}, "", err
		}
	}
	// Keep UI_SETTINGS_FILE as an optional environment override for containers.
	// Explicit command-line flags still take precedence over environment values.
	if v := os.Getenv("UI_SETTINGS_FILE"); v != "" {
		cfg.UISettingsPath = v
	}

	flags := flag.NewFlagSet("leaselens-dnsmasq", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: leaselens-dnsmasq [options]")
		fmt.Fprintln(flags.Output(), "Precedence: defaults < environment < config file < command line.")
		fmt.Fprintln(flags.Output())
		fmt.Fprintln(flags.Output(), "Options:")
		flags.PrintDefaults()
	}

	flags.StringVar(&configPath, "config", configPath, "Path to YAML-style configuration file (environment: CONFIG_FILE)")
	flags.StringVar(&cfg.WorkingDir, "working-dir", cfg.WorkingDir, "Directory for persistent LeaseLens data (environment: WORKING_DIR)")
	flags.StringVar(&cfg.DatabasePath, "database-file", cfg.DatabasePath, "Path to SQLite database (environment: DATABASE_PATH)")
	flags.StringVar(&cfg.Host, "host", cfg.Host, "Address to listen on (environment: HOST)")
	flags.StringVar(&cfg.Port, "port", cfg.Port, "TCP port to listen on (environment: PORT)")
	flags.StringVar(&cfg.LeasePath, "leases-file", cfg.LeasePath, "Path to dnsmasq lease file (environment: DNSMASQ_LEASES_FILE)")
	flags.StringVar(&cfg.HostsPath, "hosts-file", cfg.HostsPath, "Path to optional dnsmasq DHCP hosts file (environment: DNSMASQ_HOSTS_FILE)")
	flags.StringVar(&cfg.KnownMACsPath, "known-macs-file", cfg.KnownMACsPath, "Path to persistent remembered-MAC JSON file (environment: KNOWN_MACS_FILE)")
	flags.StringVar(&cfg.UISettingsPath, "ui-settings-file", cfg.UISettingsPath, "Path to persistent UI settings JSON file (environment: UI_SETTINGS_FILE)")
	flags.StringVar(&cfg.DHCPInterface, "dhcp-interface", cfg.DHCPInterface, "Interface used by dhcp_release for native IPv4 lease release (environment: DNSMASQ_INTERFACE)")
	flags.StringVar(&logLevel, "log-level", logLevel, "Log level: DEBUG, INFO, WARN, or ERROR (environment: LOG_LEVEL)")
	flags.StringVar(&cfg.AppVersion, "app-version", cfg.AppVersion, "Application version (environment: APP_VERSION)")
	flags.StringVar(&cfg.ReleaseDate, "release-date", cfg.ReleaseDate, "Application release date (environment: APP_RELEASE_DATE)")
	flags.StringVar(&cfg.RepoURL, "repo-url", cfg.RepoURL, "Repository URL (environment: REPO_URL)")
	flags.StringVar(&cfg.SystemName, "system-name", cfg.SystemName, "System display name (environment: SYSTEM_NAME)")

	if err := flags.Parse(args); err != nil {
		return Config{}, "", err
	}
	if flags.NArg() > 0 {
		return Config{}, "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	return cfg, logLevel, nil
}

// applyConfigFile reads a deliberately small YAML subset: flat scalar key/value
// pairs. This keeps configuration support dependency-free and easy to edit.
func applyConfigFile(path string, cfg *Config, logLevel *string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	knownPathSet, uiPathSet, eventsPathSet, databasePathSet := false, false, false, false
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf("config file %q line %d: expected key: value", path, lineNo)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("config file %q line %d: value for %q is empty", path, lineNo, key)
		}
		// Accept either quoted or unquoted scalar values. A whitespace-prefixed
		// inline comment is stripped; '#' inside a quoted value is retained.
		if strings.HasPrefix(value, "\"") {
			if end := strings.LastIndex(value, "\""); end > 0 {
				value = value[1:end]
			}
		} else if strings.HasPrefix(value, "'") {
			if end := strings.LastIndex(value, "'"); end > 0 {
				value = value[1:end]
			}
		} else if comment := strings.Index(value, " #"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}

		switch key {
		case "working_dir", "working-dir", "data_dir", "data-dir":
			cfg.WorkingDir = value
		case "database_path", "database-path", "database_file", "database-file":
			cfg.DatabasePath = value
			databasePathSet = true
		case "database_type", "database-type":
			cfg.DatabaseType = strings.ToLower(value)
		case "postgres_host", "postgres-host":
			cfg.PostgresHost = value
		case "postgres_port", "postgres-port":
			cfg.PostgresPort = value
		case "postgres_database", "postgres-database":
			cfg.PostgresDatabase = value
		case "postgres_user", "postgres-user":
			cfg.PostgresUser = value
		case "postgres_password", "postgres-password":
			cfg.PostgresPassword = value
		case "postgres_sslmode", "postgres-sslmode":
			cfg.PostgresSSLMode = value
		case "host":
			cfg.Host = value
		case "port":
			cfg.Port = value
		case "leases_file", "leases-file":
			cfg.LeasePath = value
		case "hosts_file", "hosts-file":
			cfg.HostsPath = value
		case "known_macs_file", "known-macs-file":
			cfg.KnownMACsPath = value
			knownPathSet = true
		case "ui_settings_file", "ui-settings-file":
			cfg.UISettingsPath = value
			uiPathSet = true
		case "dhcp_events_file", "dhcp-events-file":
			cfg.DHCPEventsPath = value
			eventsPathSet = true
		case "dhcp_interface", "dhcp-interface":
			cfg.DHCPInterface = value
		case "log_level", "log-level":
			*logLevel = value
		case "app_version", "app-version":
			cfg.AppVersion = value
		case "release_date", "release-date":
			cfg.ReleaseDate = value
		case "repo_url", "repo-url":
			cfg.RepoURL = value
		case "system_name", "system-name":
			cfg.SystemName = value
		default:
			return fmt.Errorf("config file %q line %d: unknown key %q", path, lineNo, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	// A working directory supplies defaults for persistent files. Explicit file
	// paths in the config remain supported for backwards compatibility.
	if strings.TrimSpace(cfg.WorkingDir) != "" {
		if !databasePathSet && os.Getenv("DATABASE_PATH") == "" {
			cfg.DatabasePath = filepath.Join(cfg.WorkingDir, "leaselens.db")
		}
		if !knownPathSet {
			cfg.KnownMACsPath = filepath.Join(cfg.WorkingDir, "known-macs.json")
		}
		if !uiPathSet {
			cfg.UISettingsPath = filepath.Join(cfg.WorkingDir, "ui-settings.json")
		}
		if !eventsPathSet {
			cfg.DHCPEventsPath = filepath.Join(cfg.WorkingDir, "dhcp-events.log")
		}
	}
	return nil
}
