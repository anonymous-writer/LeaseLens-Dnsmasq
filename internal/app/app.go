package app

// Application entry point: load configuration, create dependencies, and start HTTP.
// Web assets are supplied as an embedded filesystem by the executable.

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
)

// Run loads configuration, wires application dependencies, and starts the HTTP server.
func Run(assets fs.FS, nativeBuild bool) {
	cfg, logLevel, err := loadConfigFromArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if cfg.SystemName == "" {
		if nativeBuild {
			cfg.SystemName = detectNativeSystemName()
		} else {
			cfg.SystemName = "LeaseLens"
		}
	}
	SetLogLevel(logLevel)

	db, err := openDatabase(cfg.DatabasePath, cfg.KnownMACsPath, cfg.UISettingsPath, cfg.DHCPEventsPath)
	if err != nil {
		LogFatalf("cannot initialize LeaseLens database: %v", err)
	}
	defer db.Close()
	known := newKnownMACs(db)
	tmpl, err := template.ParseFS(assets, "templates/index.html")
	if err != nil {
		LogFatalf("cannot parse embedded template templates/index.html: %v", err)
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		LogFatalf("cannot access embedded static assets: %v", err)
	}

	dnsmasqVersion := ""
	if nativeBuild {
		if validInterfaceName(cfg.DHCPInterface) {
			LogInfof("native IPv4 lease release action enabled interface=%s", cfg.DHCPInterface)
		} else {
			LogInfof("native IPv4 lease release action disabled; configure dhcp_interface to enable it")
		}
		dnsmasqVersion = detectDnsmasqVersion()
		if dnsmasqVersion == "" {
			LogDebugf("native build: dnsmasq version unavailable (dnsmasq --version failed or returned an unrecognized result)")
		} else {
			LogInfof("detected native dnsmasq version=%s", dnsmasqVersion)
		}
	}

	handler := newHTTPHandler(cfg, db, known, tmpl, staticFS, dnsmasqVersion, nativeBuild)
	address := net.JoinHostPort(cfg.Host, cfg.Port)
	LogInfof("leaselens-dnsmasq starting version=%s listen=%s", cfg.AppVersion, address)
	if err := http.ListenAndServe(address, handler); err != nil {
		LogFatalf("HTTP server stopped: %v", err)
	}
}
