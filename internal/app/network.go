package app

// Probe devices and discover their HTTP/HTTPS management URLs.

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func searchDomains() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Fields(s.Text())
		if len(p) < 2 || (p[0] != "search" && p[0] != "domain") {
			continue
		}
		for _, v := range p[1:] {
			v = strings.Trim(v, ".")
			if v != "" && !contains(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}

func hostnameCandidates(name string) []string {
	if name == "" || name == "*" {
		return nil
	}
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	if name == "" {
		return nil
	}
	if strings.Contains(name, ".") {
		return []string{name}
	}
	var out []string
	for _, s := range searchDomains() {
		x := name + "." + s
		if !contains(out, x) {
			out = append(out, x)
		}
	}
	x := name + ".local"
	if !contains(out, x) {
		out = append(out, x)
	}
	return out
}

func resolvesToIP(host, ip string, timeout time.Duration) bool {
	target := net.ParseIP(ip)
	if target == nil || strings.TrimSpace(host) == "" {
		return false
	}
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}

	// Use the context-aware resolver instead of wrapping net.LookupIP in a
	// goroutine. The DNS request can now be cancelled when its deadline expires.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if addr.IP.Equal(target) {
			return true
		}
	}
	return false
}

func checkWebHostname(ip, name, webURL string, timeout time.Duration) string {
	if webURL == "" {
		return ""
	}
	scheme := "http"
	if strings.HasPrefix(webURL, "https://") {
		scheme = "https"
	}
	for _, h := range hostnameCandidates(name) {
		if resolvesToIP(h, ip, timeout) {
			return scheme + "://" + h
		}
	}
	return ""
}

func formatWebURL(scheme, ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if parsed.To4() == nil {
		return scheme + "://[" + ip + "]"
	}
	return scheme + "://" + ip
}

func checkWebUI(ip string, timeout time.Duration) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}

	// IPv6 literals must be enclosed in brackets when used in an HTTP URL.
	requestHost := ip
	if parsed.To4() == nil {
		requestHost = "[" + ip + "]"
	}

	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	for _, scheme := range []string{"http", "https"} {
		transport := &http.Transport{}
		if scheme == "https" {
			// Many local device web interfaces use self-signed certificates.
			// This probe only detects whether the management endpoint responds;
			// it does not establish trust in the device certificate.
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		}
		client := &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		req, err := http.NewRequest(http.MethodGet, scheme+"://"+requestHost+"/", nil)
		if err != nil {
			transport.CloseIdleConnections()
			continue
		}
		req.Header.Set("User-Agent", "leaselens-dnsmasq")
		resp, err := client.Do(req)
		if err != nil {
			transport.CloseIdleConnections()
			continue
		}
		// Drain only a small amount: this permits connection cleanup without
		// downloading potentially large device pages during a reachability scan.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32*1024))
		_ = resp.Body.Close()
		transport.CloseIdleConnections()
		if resp.StatusCode >= 100 && resp.StatusCode <= 599 {
			return formatWebURL(scheme, ip)
		}
	}
	return ""
}

// pingReachable performs one bounded ICMP probe. If ping is unavailable or
// the probe cannot be performed, it returns false; callers should avoid
// interpreting that alone as definitive proof that a device is offline.

func pingReachable(ip string, timeout time.Duration) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	if timeout <= 0 {
		timeout = 1200 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// BusyBox and iputils support -c 1 and -W; the container image already
	// provides ping in the deployment environment.
	waitSeconds := int((timeout + time.Second - 1) / time.Second)
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(max(1, waitSeconds)), ip)
	return cmd.Run() == nil
}

func addWebURLs(leases []LeaseEntry) {
	if len(leases) == 0 {
		return
	}
	timeout := envDuration("WEB_UI_TIMEOUT", 500*time.Millisecond)
	workers := envInt("WEB_UI_MAX_WORKERS", 16)
	if workers < 1 {
		workers = 1
	}
	if workers > len(leases) {
		workers = len(leases)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range jobs {
				leases[n].WebURL = checkWebUI(leases[n].IPAddress, timeout)
				switch {
				case leases[n].WebURL != "":
					leases[n].Status = "online" // green: HTTP/HTTPS responded
				case pingReachable(leases[n].IPAddress, envDuration("PING_TIMEOUT", 1200*time.Millisecond)):
					leases[n].Status = "reachable" // yellow: host responds to ping, no web UI
				default:
					leases[n].Status = "offline" // red: no response to supported probes
				}
			}
		}()
	}
	for i := range leases {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for i := range leases {
		leases[i].WebHostURL = checkWebHostname(leases[i].IPAddress, leases[i].Name, leases[i].WebURL, timeout)
	}
}
