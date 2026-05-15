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
