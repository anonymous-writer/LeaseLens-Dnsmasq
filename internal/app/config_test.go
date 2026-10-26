package app

import (
	"os"
	"testing"
)

func TestLoadConfigFromArgsOverridesEnvironment(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "5000")
	t.Setenv("DNSMASQ_LEASES_FILE", "/env/leases")
	t.Setenv("DNSMASQ_HOSTS_FILE", "/env/hosts")
	t.Setenv("KNOWN_MACS_FILE", "/env/known.json")
	t.Setenv("UI_SETTINGS_FILE", "")
	t.Setenv("LOG_LEVEL", "WARN")

	cfg, logLevel, err := loadConfigFromArgs([]string{
		"--host", "::",
		"--port", "8081",
		"--leases-file", "/tmp/leases",
		"--hosts-file", "/tmp/hosts",
		"--known-macs-file", "/tmp/known.json",
		"--dhcp-interface", "eth1",
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
	if cfg.DHCPInterface != "eth1" {
		t.Fatalf("DHCPInterface = %q, want eth1", cfg.DHCPInterface)
	}
	if logLevel != "DEBUG" {
		t.Fatalf("logLevel = %q, want DEBUG", logLevel)
	}
}

func TestLoadConfigFromArgsUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
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
	t.Setenv("CONFIG_FILE", "")
	_, _, err := loadConfigFromArgs([]string{"unexpected"})
	if err == nil {
		t.Fatal("expected positional argument to be rejected")
	}
}

func TestLoadConfigFromYAMLAndCommandLineOverride(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("UI_SETTINGS_FILE", "")
	configPath := t.TempDir() + "/config.yaml"
	content := `host: "127.0.0.1"
port: "3008"
leases_file: "/tmp/config.leases"
hosts_file: "/tmp/config.hosts"
known_macs_file: "/tmp/known-macs.json"
ui_settings_file: "/tmp/ui-settings.json"
dhcp_interface: "eth1"
log_level: "DEBUG"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, logLevel, err := loadConfigFromArgs([]string{"--config", configPath, "--port", "8081"})
	if err != nil {
		t.Fatalf("loadConfigFromArgs returned an error: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != "8081" {
		t.Fatalf("config/flag values = host %q port %q, want 127.0.0.1 and 8081", cfg.Host, cfg.Port)
	}
	if cfg.LeasePath != "/tmp/config.leases" || cfg.HostsPath != "/tmp/config.hosts" || cfg.KnownMACsPath != "/tmp/known-macs.json" {
		t.Fatalf("config file paths not applied: %+v", cfg)
	}
	if cfg.DHCPInterface != "eth1" {
		t.Fatalf("DHCPInterface from config = %q, want eth1", cfg.DHCPInterface)
	}
	if cfg.UISettingsPath != "/tmp/ui-settings.json" {
		t.Fatalf("UISettingsPath from config = %q, want /tmp/ui-settings.json", cfg.UISettingsPath)
	}
	if logLevel != "DEBUG" {
		t.Fatalf("logLevel = %q, want DEBUG", logLevel)
	}
}

func TestLoadConfigFromArgsRejectsUnknownConfigKey(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
	configPath := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(configPath, []byte("porrt: 3008\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, _, err := loadConfigFromArgs([]string{"--config=" + configPath}); err == nil {
		t.Fatal("expected unknown config key to be rejected")
	}
}

func TestUISettingsEnvironmentOverridesConfig(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(configPath, []byte("ui_settings_file: /config/ui.json\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("UI_SETTINGS_FILE", "/env/ui.json")
	cfg, _, err := loadConfigFromArgs([]string{"--config", configPath})
	if err != nil {
		t.Fatalf("loadConfigFromArgs returned an error: %v", err)
	}
	if cfg.UISettingsPath != "/env/ui.json" {
		t.Fatalf("UISettingsPath = %q, want environment override /env/ui.json", cfg.UISettingsPath)
	}
}
