package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardDoesNotLinkNodesWithoutWebURL(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UnixNano()
	nodes := []NodeStats{
		{
			Name:      "sync-only",
			Version:   appVersion,
			UpdatedAt: now,
			CPU:       []float64{10},
			MemTotal:  100,
			MemUsed:   25,
		},
		{
			Name:      "web-node",
			Version:   appVersion,
			WebURL:    "https://psstd.example.com/?psstd_node=web-node",
			UpdatedAt: now,
			CPU:       []float64{20},
			MemTotal:  100,
			MemUsed:   30,
		},
	}
	for _, node := range nodes {
		if err := dbSet(db, node); err != nil {
			t.Fatalf("set %s: %v", node.Name, err)
		}
	}

	req := httptest.NewRequest("GET", "/?theme=dark&palette=monochrome", nil)
	rr := httptest.NewRecorder()
	makeHandler(db, "web-node").ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<span class="node-name">sync-only</span>`) {
		t.Fatalf("sync-only node was not rendered as non-link text:\n%s", body)
	}
	if strings.Contains(body, `<a href="/">sync-only</a>`) || strings.Contains(body, `>sync-only</a>`) {
		t.Fatalf("sync-only node rendered as a link:\n%s", body)
	}
	if !strings.Contains(body, `>web-node</a>`) {
		t.Fatalf("web node was not rendered as a link:\n%s", body)
	}
}

func TestNodeHealthDistinguishesFreshStaleOffline(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		node NodeStats
		want healthState
	}{
		{
			name: "fresh",
			node: NodeStats{UpdatedAt: now.Add(-2 * time.Second).UnixNano(), TTLSeconds: 10},
			want: healthFresh,
		},
		{
			name: "stale",
			node: NodeStats{UpdatedAt: now.Add(-7 * time.Second).UnixNano(), TTLSeconds: 10},
			want: healthStale,
		},
		{
			name: "offline",
			node: NodeStats{UpdatedAt: now.Add(-11 * time.Second).UnixNano(), TTLSeconds: 10},
			want: healthOffline,
		},
		{
			name: "zero timestamp offline",
			node: NodeStats{},
			want: healthOffline,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nodeHealth(tc.node).State; got != tc.want {
				t.Fatalf("state = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestSummarizeClusterCountsStatesAndHottestOnlineNode(t *testing.T) {
	now := time.Now()
	summary := summarizeCluster([]NodeStats{
		{Name: "fresh-hot", UpdatedAt: now.UnixNano(), TTLSeconds: 10, CPU: []float64{20}, Load: [3]float64{0.5}},
		{Name: "stale-hot", UpdatedAt: now.Add(-7 * time.Second).UnixNano(), TTLSeconds: 10, CPU: []float64{70}, Load: [3]float64{1.2}},
		{Name: "offline", UpdatedAt: 0, TTLSeconds: 10, CPU: []float64{99}, Load: [3]float64{9}},
	})

	if summary.Fresh != 1 || summary.Stale != 1 || summary.Offline != 1 {
		t.Fatalf("counts = fresh %d stale %d offline %d, want 1/1/1", summary.Fresh, summary.Stale, summary.Offline)
	}
	if !summary.HasHot || summary.Hottest != "fresh-hot" {
		t.Fatalf("hottest = %q has=%t, want fresh-hot", summary.Hottest, summary.HasHot)
	}
}

func TestDashboardRendersClusterSummaryAndStateData(t *testing.T) {
	db := openTestDB(t)
	now := time.Now()
	nodes := []NodeStats{
		{Name: "fresh", Version: appVersion, UpdatedAt: now.UnixNano(), TTLSeconds: 10, CPU: []float64{20}, MemTotal: 100, MemUsed: 30},
		{Name: "stale", Version: appVersion, UpdatedAt: now.Add(-7 * time.Second).UnixNano(), TTLSeconds: 10, CPU: []float64{80}, MemTotal: 100, MemUsed: 40},
		{Name: "offline", Version: appVersion, UpdatedAt: 0, TTLSeconds: 10, CPU: []float64{90}, MemTotal: 100, MemUsed: 50},
	}
	for _, node := range nodes {
		if err := dbSet(db, node); err != nil {
			t.Fatalf("set %s: %v", node.Name, err)
		}
	}

	req := httptest.NewRequest("GET", "/?theme=dark&palette=monochrome", nil)
	rr := httptest.NewRecorder()
	makeHandler(db, "fresh").ServeHTTP(rr, req)

	body := rr.Body.String()
	for _, want := range []string{
		`online 1`,
		`stale 1`,
		`offline 1`,
		`data-state="stale"`,
		`id="sort-select"`,
		`id="hide-offline"`,
		`value="cpuAvg"`,
		`value="cpuMax"`,
		`value="memPct"`,
		`value="memUsed"`,
		`value="memTotal"`,
		`value="load1"`,
		`value="load5"`,
		`value="load15"`,
		`data-cpu-avg="20.000"`,
		`data-cpu-max="20.000"`,
		`data-mem-pct="30.000"`,
		`data-mem-used="30"`,
		`data-mem-total="100"`,
		`data-load1="0.000"`,
		`data-load5="0.000"`,
		`data-load15="0.000"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, body)
		}
	}
}

func TestCollectStatsUsesConfiguredTTL(t *testing.T) {
	stats, err := collectStats("node-a", "http://node-a:8080", "v1", 42*time.Second)
	if err != nil {
		t.Fatalf("collectStats: %v", err)
	}
	if stats.TTLSeconds != 42 {
		t.Fatalf("ttl seconds = %d, want 42", stats.TTLSeconds)
	}
}
