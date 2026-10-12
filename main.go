package main

import (
	"embed"

	"github.com/anonymous-writer/dnsmasq-leases-ui/internal/app"
)

// Embed the complete web UI in the executable so the binary can run on its own.
//
//go:embed templates/index.html static/*
var webAssets embed.FS

func main() {
	app.Run(webAssets)
}
