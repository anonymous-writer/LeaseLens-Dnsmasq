package app

// Probe devices and discover their HTTP/HTTPS management URLs.

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	name = strings.TrimSuffix(name, ".")
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
	if target == nil {
		return false
	}

	ch := make(chan bool, 1)
	go func() {
		addrs, err := net.LookupIP(host)
		if err != nil {
			ch <- false
			return
		}
		for _, a := range addrs {
			if a.Equal(target) {
				ch <- true
				return
			}
		}
		ch <- false
	}()

	select {
	case ok := <-ch:
		return ok
	case <-time.After(timeout):
		return false
	}
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

	for _, scheme := range []string{"http", "https"} {
		client := &http.Client{Timeout: timeout}
		if scheme == "https" {
			client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		}
		req, err := http.NewRequest(http.MethodGet, scheme+"://"+requestHost+"/", nil)
		if err != nil {
			continue
		}
		req.Header.Set("Connection", "close")
		req.Header.Set("User-Agent", "leaselens-dnsmasq")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// BusyBox and iputils support -c 1 and -W; the container image already
	// provides ping in the deployment environment.
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(max(1, int(timeout.Seconds()))), ip)
	return cmd.Run() == nil
}

type reachabilityRecord struct {
	CheckedAt  int64  `json:"checkedAt"`
	Status     string `json:"status"`
	WebURL     string `json:"webUrl"`
	LastOnline int64  `json:"lastOnline"`
}

var reachabilityMu sync.Mutex
var reachabilityCache = map[string]reachabilityRecord{}
var reachabilityLoaded bool

func reachabilityPath() string { return getenv("LEASELENS_STATUS_FILE", "/data/last-online.json") }
func loadReachabilityCache() {
	if reachabilityLoaded {
		return
	}
	reachabilityLoaded = true
	b, err := os.ReadFile(reachabilityPath())
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &reachabilityCache)
}
func saveReachabilityCache() {
	path := reachabilityPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		LogErrorf("cannot create status cache directory: %v", err)
		return
	}
	b, err := json.MarshalIndent(reachabilityCache, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0644); err != nil {
		LogErrorf("cannot save status cache: %v", err)
		return
	}
	if err = os.Rename(tmp, path); err != nil {
		LogErrorf("cannot replace status cache: %v", err)
	}
}

func addWebURLs(leases []LeaseEntry) {
	// Cache network probes for five minutes per IP/MAC; page refreshes only read cached results.
	reachabilityMu.Lock()
	defer reachabilityMu.Unlock()
	loadReachabilityCache()
	now := time.Now()
	var due []int
	for i := range leases {
		key := strings.ToLower(strings.TrimSpace(leases[i].MACAddress)) + "|" + leases[i].IPAddress
		rec, ok := reachabilityCache[key]
		if ok {
			leases[i].Status, leases[i].WebURL, leases[i].LastOnline = rec.Status, rec.WebURL, rec.LastOnline
		}
		if !ok || now.Unix()-rec.CheckedAt >= int64(envDuration("DEVICE_CHECK_INTERVAL", 5*time.Minute).Seconds()) {
			due = append(due, i)
		}
	}
	if len(due) == 0 {
		for i := range leases {
			leases[i].WebHostURL = checkWebHostname(leases[i].IPAddress, leases[i].Name, leases[i].WebURL, envDuration("WEB_UI_TIMEOUT", 500*time.Millisecond))
		}
		return
	}
	probeLeases(leases, due)
	for _, i := range due {
		key := strings.ToLower(strings.TrimSpace(leases[i].MACAddress)) + "|" + leases[i].IPAddress
		old := reachabilityCache[key]
		last := old.LastOnline
		if leases[i].Status == "online" || leases[i].Status == "reachable" {
			last = now.Unix()
		}
		leases[i].LastOnline = last
		reachabilityCache[key] = reachabilityRecord{CheckedAt: now.Unix(), Status: leases[i].Status, WebURL: leases[i].WebURL, LastOnline: last}
	}
	saveReachabilityCache()
	for i := range leases {
		leases[i].WebHostURL = checkWebHostname(leases[i].IPAddress, leases[i].Name, leases[i].WebURL, envDuration("WEB_UI_TIMEOUT", 500*time.Millisecond))
	}
}

func probeLeases(leases []LeaseEntry, due []int) {
	if len(leases) == 0 {
		return
	}
	timeout := envDuration("WEB_UI_TIMEOUT", 500*time.Millisecond)
	workers := envInt("WEB_UI_MAX_WORKERS", 4)
	if workers < 1 {
		workers = 1
	}
	if workers > len(due) {
		workers = len(due)
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
	for _, i := range due {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}
