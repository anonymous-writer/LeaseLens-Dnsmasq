package app

// Application entry point: load configuration, create dependencies, and start HTTP.
// Web assets are supplied as an embedded filesystem by the executable.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
	if strings.TrimSpace(cfg.DHCPEventAction) != "" {
		if err := runDHCPNotification(cfg); err != nil {
			LogWarnf("DHCP ntfy event: %v", err)
			os.Exit(0) // Notifications must never break dnsmasq lease processing.
		}
		return
	}

	db, err := openConfiguredDatabase(cfg.DatabaseType, cfg.DatabasePath, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresSSLMode, cfg.KnownMACsPath, cfg.UISettingsPath, cfg.DHCPEventsPath)
	if err != nil {
		LogFatalf("cannot initialize LeaseLens database: %v", err)
	}
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
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	LogInfof("leaselens-dnsmasq starting version=%s listen=%s", cfg.AppVersion, address)

	// Handle SIGINT/SIGTERM so the HTTP server can finish active requests before
	// SQLite is checkpointed and closed. This allows SQLite to remove its WAL/SHM
	// sidecar files when the database is cleanly shut down.
	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			LogErrorf("HTTP server stopped unexpectedly: %v", err)
		}
	case <-shutdownSignal.Done():
		LogInfof("shutdown signal received; stopping HTTP server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			LogWarnf("graceful HTTP shutdown failed: %v; forcing close", err)
			if closeErr := server.Close(); closeErr != nil {
				LogWarnf("force close HTTP server: %v", closeErr)
			}
		}
		cancel()
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			LogWarnf("HTTP server exit: %v", err)
		}
	}

	checkpointAndCloseDatabase(db)
	LogInfof("LeaseLens shutdown complete")
}
