package main

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func withoutNodeNameEnv(t *testing.T) {
	t.Helper()
	old, hadOld := os.LookupEnv(envNodeName)
	if err := os.Unsetenv(envNodeName); err != nil {
		t.Fatalf("unset %s: %v", envNodeName, err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(envNodeName, old)
		} else {
			_ = os.Unsetenv(envNodeName)
		}
	})
}

func TestNodeNameFromEnvUsesHostnameByDefault(t *testing.T) {
	withoutNodeNameEnv(t)

	got, err := nodeNameFromEnv("host-a")
	if err != nil {
		t.Fatalf("nodeNameFromEnv: %v", err)
	}
	if got != "host-a" {
		t.Fatalf("name = %q, want host-a", got)
	}
}

func TestNodeNameFromEnvUsesOverride(t *testing.T) {
	t.Setenv(envNodeName, "test-node")

	got, err := nodeNameFromEnv("host-a")
	if err != nil {
		t.Fatalf("nodeNameFromEnv: %v", err)
	}
	if got != "test-node" {
		t.Fatalf("name = %q, want test-node", got)
	}
}

func TestNodeNameFromEnvRejectsWhitespaceOverride(t *testing.T) {
	for _, value := range []string{" test-node", "test-node ", "test node", "\t"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(envNodeName, value)
			if got, err := nodeNameFromEnv("host-a"); err == nil {
				t.Fatalf("name = %q, want error", got)
			}
		})
	}
}

func TestNodeTTLFromEnvDefaultAndOverride(t *testing.T) {
	t.Setenv(envNodeTTL, "")
	got, err := nodeTTLFromEnv()
	if err != nil {
		t.Fatalf("default ttl: %v", err)
	}
	if got != defaultNodeTTL {
		t.Fatalf("default ttl = %s, want %s", got, defaultNodeTTL)
	}

	t.Setenv(envNodeTTL, "45s")
	got, err = nodeTTLFromEnv()
	if err != nil {
		t.Fatalf("override ttl: %v", err)
	}
	if got != 45*time.Second {
		t.Fatalf("override ttl = %s, want 45s", got)
	}
}

func TestNodeTTLFromEnvRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"soon", "1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(envNodeTTL, value)
			if got, err := nodeTTLFromEnv(); err == nil {
				t.Fatalf("ttl = %s, want error", got)
			}
		})
	}
}

func TestStartupSummaryIncludesJoinOutcome(t *testing.T) {
	base := startupConfig{
		Version:    "v1",
		NodeName:   "node-a",
		DBPath:     "./data",
		HTTPAddr:   ":8080",
		WebURL:     "http://node-a:8080",
		GossipAddr: ":7946",
		WebEnabled: true,
		NodeTTL:    15 * time.Second,
		SeedCount:  2,
		MDNSCount:  1,
	}

	joined := base
	joined.JoinedPeers = 3
	if got := startupSummary(joined); !strings.Contains(got, "join=joined=3") {
		t.Fatalf("joined summary missing outcome: %s", got)
	}

	solo := base
	if got := startupSummary(solo); !strings.Contains(got, "join=solo") {
		t.Fatalf("solo summary missing outcome: %s", got)
	}

	warn := base
	warn.JoinedPeers = 1
	warn.JoinErr = errors.New("partial join")
	if got := startupSummary(warn); !strings.Contains(got, `join=warning joined=1 error="partial join"`) {
		t.Fatalf("warning summary missing outcome: %s", got)
	}
}

func TestParseCLIListFlag(t *testing.T) {
	for _, args := range [][]string{{"-l"}, {"--list"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if opts := parseCLI(args); !opts.List {
				t.Fatalf("List = false, want true")
			}
		})
	}
}
