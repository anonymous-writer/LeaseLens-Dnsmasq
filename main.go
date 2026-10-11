package main

import "github.com/anonymous-writer/dnsmasq-leases-ui/internal/app"

// main is the small executable entry point. Application implementation lives
// in internal/app, grouped by responsibility.
func main() {
	app.Run()
}
