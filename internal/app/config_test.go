package app

import "testing"

func TestLoadConfigFromArgsOverridesEnvironment(t *testing.T) {
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "5000")
	t.Setenv("DNSMASQ_LEASES_FILE", "/env/leases")
	t.Setenv("DNSMASQ_HOSTS_FILE", "/env/hosts")
	t.Setenv("KNOWN_MACS_FILE", "/env/known.json")
	t.Setenv("LOG_LEVEL", "WARN")

	cfg, logLevel, err := loadConfigFromArgs([]string{
		"--host", "::",
		"--port", "8081",
		"--leases-file", "/tmp/leases",
		"--hosts-file", "/tmp/hosts",
		"--known-macs-file", "/tmp/known.json",
		"--log-level", "DEBUG",
	})
	if err != nil {
		t.Fatalf("loadConfigFromArgs returned an error: %v", err)
	}
	if cfg.Host != "::" || cfg.Port != "8081" {
		t.Fatalf("listen config = %q:%q, want :: port 8081", cfg.Host, cfg.Port)
	}
	if cfg.LeasePath != "/tmp/leases" || cfg.HostsPath != "/tmp/hosts" || cfg.KnownMACsPath != "/tmp/known.json" {
		t.Fatalf("file flags not applied: %+v", cfg)
	}
	if logLevel != "DEBUG" {
		t.Fatalf("logLevel = %q, want DEBUG", logLevel)
	}
}

func TestLoadConfigFromArgsUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("HOST", "192.0.2.1")
	t.Setenv("PORT", "9000")
	t.Setenv("DNSMASQ_LEASES_FILE", "/custom/leases")
	t.Setenv("LOG_LEVEL", "ERROR")

	cfg, logLevel, err := loadConfigFromArgs(nil)
	if err != nil {
		t.Fatalf("loadConfigFromArgs returned an error: %v", err)
	}
	if cfg.Host != "192.0.2.1" || cfg.Port != "9000" || cfg.LeasePath != "/custom/leases" {
		t.Fatalf("environment defaults not applied: %+v", cfg)
	}
	if logLevel != "ERROR" {
		t.Fatalf("logLevel = %q, want ERROR", logLevel)
	}
}

func TestLoadConfigFromArgsRejectsPositionalArguments(t *testing.T) {
	_, _, err := loadConfigFromArgs([]string{"unexpected"})
	if err == nil {
		t.Fatal("expected positional argument to be rejected")
	}
}
