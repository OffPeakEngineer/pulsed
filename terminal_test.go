package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderTerminalNodesSortsByName(t *testing.T) {
	now := time.Now().UnixNano()
	out := renderTerminalNodes([]NodeStats{
		{Name: "z-node", Version: appVersion, CPU: []float64{1}, MemTotal: 1, UpdatedAt: now},
		{Name: "a-node", Version: appVersion, CPU: []float64{1}, MemTotal: 1, UpdatedAt: now},
	})

	first := strings.Index(out, "a-node")
	second := strings.Index(out, "z-node")
	if first < 0 || second < 0 {
		t.Fatalf("missing rendered nodes:\n%s", out)
	}
	if first > second {
		t.Fatalf("nodes were not sorted by name:\n%s", out)
	}
}

func TestRenderTerminalGridPacksStableRows(t *testing.T) {
	now := time.Now().UnixNano()
	out := renderTerminalGrid([]NodeStats{
		{Name: "c-node", Version: appVersion, CPU: []float64{1}, MemTotal: 1, UpdatedAt: now},
		{Name: "a-node", Version: appVersion, CPU: []float64{1}, MemTotal: 1, UpdatedAt: now},
		{Name: "b-node", Version: appVersion, CPU: []float64{1}, MemTotal: 1, UpdatedAt: now},
	}, terminalCellWidth*2+terminalCellGap)

	first := strings.Index(out, "a-node")
	second := strings.Index(out, "b-node")
	third := strings.Index(out, "c-node")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("missing rendered nodes:\n%s", out)
	}
	if first > second || second > third {
		t.Fatalf("grid nodes were not sorted by name:\n%s", out)
	}
	if strings.Count(out, "╭") != 3 {
		t.Fatalf("grid did not render three bordered cells:\n%s", out)
	}
}

func TestTerminalColumns(t *testing.T) {
	if got := terminalColumns(terminalCellWidth - 1); got != 1 {
		t.Fatalf("narrow columns = %d, want 1", got)
	}
	if got := terminalColumns(terminalCellWidth*2 + terminalCellGap); got != 2 {
		t.Fatalf("wide columns = %d, want 2", got)
	}
}
