package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/hashicorp/memberlist"
)

const (
	envDB      = "PSSTD_DB"
	envHTTP    = "PSSTD_HTTP"
	envGossip  = "PSSTD_GOSSIP"
	envSeeds   = "PSSTD_SEEDS"
	envWeb     = "PSSTD_WEB" // "true" to enable HTTP, default true
	gossipPort = 7946
	httpPort   = 8080
)

func main() {
	hostname, _ := os.Hostname()

	dbPath     := envOr(envDB,     "/var/lib/psstd/state")
	httpAddr   := envOr(envHTTP,   fmt.Sprintf(":%d", httpPort))
	gossipAddr := envOr(envGossip, fmt.Sprintf(":%d", gossipPort))
	seeds      := splitCSV(envOr(envSeeds, ""))
	webEnabled := envOr(envWeb, "true") != "false"

	// ── Pebble ──────────────────────────────────────────────────────────────
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("pebble open: %v", err)
	}
	defer db.Close()

	// ── Gossip delegate ──────────────────────────────────────────────────────
	delegate := newKVDelegate(db)

	cfg := memberlist.DefaultLANConfig()
	cfg.Name = hostname
	cfg.BindAddr, cfg.BindPort = splitHostPort(gossipAddr)
	cfg.Delegate = delegate
	cfg.Events = newEventDelegate(db)
	cfg.Logger = log.New(os.Stderr, "[memberlist] ", log.LstdFlags)

	list, err := memberlist.Create(cfg)
	if err != nil {
		log.Fatalf("memberlist create: %v", err)
	}
	delegate.broadcasts.NumNodes = func() int { return list.NumMembers() }

	// ── Discovery ────────────────────────────────────────────────────────────
	// 1. Register ourselves via mDNS so peers can find us on LAN
	stopMDNS := registerMDNS(hostname, gossipPort)
	defer stopMDNS()

	// 2. Scan for existing peers (mDNS + any explicit seeds)
	discovered := discoverPeers(gossipPort)
	allSeeds := append(seeds, discovered...)
	if len(allSeeds) > 0 {
		if n, err := list.Join(allSeeds); err != nil {
			log.Printf("join warning (joined %d): %v", n, err)
		} else {
			log.Printf("joined cluster, %d peer(s)", n)
		}
	} else {
		log.Println("no peers found — running solo, will be discovered by others")
	}

	// ── Stats heartbeat ──────────────────────────────────────────────────────
	go statsLoop(hostname, db, delegate)

	// ── HTTP ─────────────────────────────────────────────────────────────────
	if webEnabled {
		mux := http.NewServeMux()
		mux.HandleFunc("/", makeHandler(db))
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, "ok")
		})
		log.Printf("psstd — node=%s http=%s gossip=%s web=true", hostname, httpAddr, gossipAddr)
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			log.Fatalf("http: %v", err)
		}
	} else {
		log.Printf("psstd — node=%s gossip=%s web=false", hostname, gossipAddr)
		select {} // block forever
	}
}

// ── Stats loop ───────────────────────────────────────────────────────────────

func statsLoop(hostname string, db *pebble.DB, d *kvDelegate) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		stats, err := collectStats(hostname)
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

// ── Event delegate (node leave/fail → mark offline immediately) ──────────────

type eventDelegate struct{ db *pebble.DB }

func newEventDelegate(db *pebble.DB) *eventDelegate { return &eventDelegate{db} }

func (e *eventDelegate) NotifyJoin(n *memberlist.Node) {
	log.Printf("[psstd] node joined: %s", n.Name)
}
func (e *eventDelegate) NotifyLeave(n *memberlist.Node) {
	log.Printf("[psstd] node left: %s", n.Name)
	markOffline(e.db, n.Name)
}
func (e *eventDelegate) NotifyUpdate(n *memberlist.Node) {}

func markOffline(db *pebble.DB, name string) {
	existing, closer, err := db.Get(keyFor(name))
	if err != nil {
		return
	}
	var s NodeStats
	if json.Unmarshal(existing, &s) == nil {
		s.UpdatedAt = 0 // zero ts → immediately stale in render
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
