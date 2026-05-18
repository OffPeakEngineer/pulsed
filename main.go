package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/hashicorp/memberlist"
)

type cliOptions struct {
	List bool
}

const (
	envDB       = "PSSTD_DB"
	envHTTP     = "PSSTD_HTTP"
	envGossip   = "PSSTD_GOSSIP"
	envSeeds    = "PSSTD_SEEDS"
	envHTTPAd   = "PSSTD_ADVERTISE_HTTP"
	envWeb      = "PSSTD_WEB" // "true" to enable HTTP, default true
	envNodeName = "PSSTD_NODE_NAME"
	envNodeTTL  = "PSSTD_NODE_TTL"
	gossipPort  = 7946
	httpPort    = 8080
)

func main() {
	opts := parseCLI(os.Args[1:])
	hostname, _ := os.Hostname()
	nodeName, err := nodeNameFromEnv(hostname)
	if err != nil {
		log.Fatalf("node name: %v", err)
	}
	nodeTTL, err := nodeTTLFromEnv()
	if err != nil {
		log.Fatalf("node ttl: %v", err)
	}

	dbPath := envOr(envDB, "./data")
	httpAddr := envOr(envHTTP, fmt.Sprintf(":%d", httpPort))
	gossipAddr := envOr(envGossip, fmt.Sprintf(":%d", gossipPort))
	seeds := splitCSV(envOr(envSeeds, ""))
	webEnabled := envOr(envWeb, "true") != "false"
	webURL := advertisedHTTPURL(httpAddr)
	if override := os.Getenv(envHTTPAd); override != "" {
		webURL = override
	}
	if !webEnabled {
		webURL = ""
	}

	// ── Local state ─────────────────────────────────────────────────────────
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		if pebbleLockHeld(err) {
			log.Printf("psstd already appears to own %s; starting terminal mirror instead", dbPath)
			runTerminalMirror(nodeName, gossipAddr, seeds, opts.List)
			return
		}
		log.Fatalf("pebble open: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()
	if err := purgeOfflineDifferentVersion(db, appVersion); err != nil {
		log.Printf("stale version purge: %v", err)
	}

	// ── Peer sync ───────────────────────────────────────────────────────────
	delegate := newKVDelegate(db, appVersion)

	cfg := memberlist.DefaultLANConfig()
	cfg.Name = nodeName
	cfg.BindAddr, cfg.BindPort = splitHostPort(gossipAddr)
	cfg.Delegate = delegate
	cfg.Events = newEventDelegate(db, appVersion)
	cfg.Logger = log.New(os.Stderr, "[memberlist] ", log.LstdFlags)

	list, err := memberlist.Create(cfg)
	if err != nil {
		if addressInUse(err) {
			if closeErr := db.Close(); closeErr != nil {
				log.Printf("db close before terminal mirror: %v", closeErr)
			}
			db = nil
			log.Printf("psstd already appears to be listening on %s; starting terminal mirror instead", gossipAddr)
			runTerminalMirror(nodeName, gossipAddr, seeds, opts.List)
			return
		}
		log.Fatalf("memberlist create: %v", err)
	}
	delegate.broadcasts.NumNodes = func() int { return list.NumMembers() }

	// ── Discovery ────────────────────────────────────────────────────────────
	// 1. Register ourselves via mDNS so peers can find us on LAN
	stopMDNS := registerMDNS(nodeName, cfg.BindPort)
	defer stopMDNS()

	// 2. Scan for existing peers (mDNS + any explicit seeds)
	discovered := discoverPeers()
	allSeeds := append(seeds, discovered...)
	joinedPeers := 0
	joinErr := error(nil)
	if len(allSeeds) > 0 {
		joinedPeers, joinErr = list.Join(allSeeds)
	}
	logStartupConfig(startupConfig{
		NodeName:    nodeName,
		DBPath:      dbPath,
		HTTPAddr:    httpAddr,
		WebURL:      webURL,
		GossipAddr:  gossipAddr,
		WebEnabled:  webEnabled,
		Version:     appVersion,
		NodeTTL:     nodeTTL,
		SeedCount:   len(seeds),
		MDNSCount:   len(discovered),
		JoinedPeers: joinedPeers,
		JoinErr:     joinErr,
	})

	// ── Stats heartbeat ─────────────────────────────────────────────────────
	go statsLoop(nodeName, webURL, appVersion, nodeTTL, db, delegate)

	// ── HTTP ─────────────────────────────────────────────────────────────────
	if webEnabled {
		mux := http.NewServeMux()
		mux.HandleFunc("/", makeHandler(db, nodeName))
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			log.Fatalf("http: %v", err)
		}
	} else {
		select {} // block forever
	}
}

func parseCLI(args []string) cliOptions {
	fs := flag.NewFlagSet("psstd", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	var opts cliOptions
	fs.BoolVar(&opts.List, "l", false, "render terminal mirror as a vertical node list")
	fs.BoolVar(&opts.List, "list", false, "render terminal mirror as a vertical node list")
	_ = fs.Parse(args)
	return opts
}

type startupConfig struct {
	NodeName    string
	DBPath      string
	HTTPAddr    string
	WebURL      string
	GossipAddr  string
	WebEnabled  bool
	Version     string
	NodeTTL     time.Duration
	SeedCount   int
	MDNSCount   int
	JoinedPeers int
	JoinErr     error
}

func logStartupConfig(cfg startupConfig) {
	log.Print(startupSummary(cfg))
}

func startupSummary(cfg startupConfig) string {
	join := "solo"
	if cfg.JoinErr != nil {
		join = fmt.Sprintf("warning joined=%d error=%q", cfg.JoinedPeers, cfg.JoinErr)
	} else if cfg.JoinedPeers > 0 {
		join = fmt.Sprintf("joined=%d", cfg.JoinedPeers)
	}
	return fmt.Sprintf("psstd startup: version=%s node=%s db=%s web=%t http=%s advertise=%s gossip=%s ttl=%s seeds=%d mdns=%d join=%s",
		cfg.Version, cfg.NodeName, cfg.DBPath, cfg.WebEnabled, cfg.HTTPAddr, cfg.WebURL, cfg.GossipAddr, cfg.NodeTTL, cfg.SeedCount, cfg.MDNSCount, join)
}

func pebbleLockHeld(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "lock") &&
		(strings.Contains(msg, "resource temporarily unavailable") ||
			strings.Contains(msg, "held") ||
			strings.Contains(msg, "being used") ||
			strings.Contains(msg, "already in use") ||
			strings.Contains(msg, "access is denied"))
}

func addressInUse(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "bind: only one usage of each socket address")
}

func runTerminalMirror(hostname, gossipAddr string, seeds []string, listMode bool) {
	tmpDir, err := os.MkdirTemp("", "psstd-view-*")
	if err != nil {
		log.Fatalf("terminal mirror temp db: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := pebble.Open(filepath.Join(tmpDir, "data"), &pebble.Options{})
	if err != nil {
		log.Fatalf("terminal mirror db: %v", err)
	}
	defer db.Close()

	delegate := newKVDelegate(db, appVersion)
	cfg := memberlist.DefaultLANConfig()
	cfg.Name = fmt.Sprintf("%s-view-%d", hostname, os.Getpid())
	cfg.BindAddr = "0.0.0.0"
	cfg.BindPort = 0
	cfg.Delegate = delegate
	cfg.Logger = log.New(os.Stderr, "[memberlist:view] ", log.LstdFlags)

	list, err := memberlist.Create(cfg)
	if err != nil {
		log.Fatalf("terminal mirror memberlist: %v", err)
	}
	defer list.Shutdown()
	delegate.broadcasts.NumNodes = func() int { return list.NumMembers() }

	allSeeds := terminalMirrorSeeds(gossipAddr, seeds)
	if n, err := list.Join(allSeeds); err != nil {
		log.Printf("terminal mirror join warning (joined %d): %v", n, err)
	} else {
		log.Printf("terminal mirror joined cluster, %d peer(s)", n)
	}

	terminalRenderLoop(db, listMode)
}

func terminalMirrorSeeds(gossipAddr string, seeds []string) []string {
	out := append([]string{}, seeds...)
	host, port := splitHostPort(gossipAddr)
	if host != "" && host != "0.0.0.0" && host != "::" {
		out = append(out, net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	}
	out = append(out, net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	out = append(out, net.JoinHostPort("localhost", fmt.Sprintf("%d", port)))
	return append(out, discoverPeers()...)
}

// ── Stats loop ───────────────────────────────────────────────────────────────

func statsLoop(hostname, webURL, version string, ttl time.Duration, db *pebble.DB, d *kvDelegate) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		stats, err := collectStats(hostname, webURL, version, ttl)
		if err != nil {
			log.Printf("stats error: %v", err)
			continue
		}
		if err := dbSet(db, stats); err != nil {
			log.Printf("db write: %v", err)
			continue
		}
		d.broadcast(stats)
	}
}

// ── Event delegate (node leave/fail -> mark offline immediately) ─────────────

type eventDelegate struct {
	db      *pebble.DB
	version string
}

func newEventDelegate(db *pebble.DB, version string) *eventDelegate {
	return &eventDelegate{db: db, version: version}
}

func (e *eventDelegate) NotifyJoin(n *memberlist.Node) {
	log.Printf("[psstd] node joined: %s", n.Name)
}
func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	log.Printf("[psstd] node left: %s", n.Name)
	markOffline(e.db, n.Name, e.version)
}
func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {}

func markOffline(db *pebble.DB, name, version string) {
	existing, closer, err := db.Get(keyFor(name))
	if err != nil {
		return
	}
	var s NodeStats
	if json.Unmarshal(existing, &s) == nil {
		if s.Version != version {
			closer.Close()
			if err := db.Delete(keyFor(name), pebble.Sync); err != nil {
				log.Printf("purge stale offline node %s: %v", name, err)
			}
			return
		}
		s.UpdatedAt = 0 // zero timestamp renders as offline immediately
		b, _ := json.Marshal(s)
		db.Set(keyFor(name), b, pebble.Sync)
	}
	closer.Close()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func nodeNameFromEnv(hostname string) (string, error) {
	override, ok := os.LookupEnv(envNodeName)
	if !ok || override == "" {
		if strings.TrimSpace(hostname) == "" {
			return "", fmt.Errorf("hostname is empty; set %s", envNodeName)
		}
		return hostname, nil
	}
	if strings.TrimSpace(override) != override || override == "" {
		return "", fmt.Errorf("%s must not be empty or have leading/trailing whitespace", envNodeName)
	}
	if strings.ContainsAny(override, " \t\r\n") {
		return "", fmt.Errorf("%s must not contain whitespace", envNodeName)
	}
	return override, nil
}

func nodeTTLFromEnv() (time.Duration, error) {
	value, ok := os.LookupEnv(envNodeTTL)
	if !ok || value == "" {
		return defaultNodeTTL, nil
	}
	ttl, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 15s or 1m: %w", envNodeTTL, err)
	}
	if ttl < 2*time.Second {
		return 0, fmt.Errorf("%s must be at least 2s", envNodeTTL)
	}
	return ttl, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "0.0.0.0", gossipPort
	}
	port := gossipPort
	fmt.Sscanf(portStr, "%d", &port)
	if host == "" {
		host = "0.0.0.0"
	}
	return host, port
}

func advertisedHTTPURL(httpAddr string) string {
	host, portStr, err := net.SplitHostPort(httpAddr)
	if err != nil {
		host = ""
		portStr = fmt.Sprintf("%d", httpPort)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = firstAdvertisableIP()
	}
	if host == "" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, portStr)
}

func firstAdvertisableIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return ""
}
