package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/cockroachdb/pebble/v2"
	"github.com/hashicorp/memberlist"
)

type NodeStats struct {
	Name      string     `json:"name"`
	CPU       []float64  `json:"cpu"`
	MemUsed   uint64     `json:"mu"`
	MemTotal  uint64     `json:"mt"`
	Load      [3]float64 `json:"ld"`
	UpdatedAt int64      `json:"ts"` // unix nano, LWW key
}

func keyFor(name string) []byte { return []byte("node/" + name) }

// ── Pebble helpers ────────────────────────────────────────────────────────────

func dbSet(db *pebble.DB, s NodeStats) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Set(keyFor(s.Name), b, pebble.Sync)
}

func dbMergeLWW(db *pebble.DB, s NodeStats) error {
	existing, closer, err := db.Get(keyFor(s.Name))
	if err == nil {
		var cur NodeStats
		if json.Unmarshal(existing, &cur) == nil && cur.UpdatedAt >= s.UpdatedAt {
			closer.Close()
			return nil
		}
		closer.Close()
	}
	return dbSet(db, s)
}

func dbScanAll(db *pebble.DB) ([]NodeStats, error) {
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("node/"),
		UpperBound: []byte("node0"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var out []NodeStats
	for iter.First(); iter.Valid(); iter.Next() {
		var s NodeStats
		if json.Unmarshal(iter.Value(), &s) == nil {
			out = append(out, s)
		}
	}
	return out, iter.Error()
}

func dbSnapshot(db *pebble.DB) ([]byte, error) {
	nodes, err := dbScanAll(db)
	if err != nil {
		return nil, err
	}
	return json.Marshal(nodes)
}

// ── Delegate ──────────────────────────────────────────────────────────────────

type kvDelegate struct {
	db         *pebble.DB
	broadcasts *memberlist.TransmitLimitedQueue
	mu         sync.Mutex
}

func newKVDelegate(db *pebble.DB) *kvDelegate {
	return &kvDelegate{
		db: db,
		broadcasts: &memberlist.TransmitLimitedQueue{
			NumNodes:       func() int { return 1 },
			RetransmitMult: 3,
		},
	}
}

func (d *kvDelegate) NodeMeta(_ int) []byte { return nil }

func (d *kvDelegate) NotifyMsg(buf []byte) {
	if len(buf) == 0 {
		return
	}
	cp := make([]byte, len(buf))
	copy(cp, buf)
	var s NodeStats
	if json.Unmarshal(cp, &s) != nil {
		return
	}
	if err := dbMergeLWW(d.db, s); err != nil {
		log.Printf("NotifyMsg merge: %v", err)
	}
}

func (d *kvDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *kvDelegate) LocalState(_ bool) []byte {
	snap, err := dbSnapshot(d.db)
	if err != nil {
		log.Printf("LocalState: %v", err)
		return nil
	}
	return snap
}

func (d *kvDelegate) MergeRemoteState(buf []byte, _ bool) {
	if len(buf) == 0 {
		return
	}
	var nodes []NodeStats
	if json.Unmarshal(buf, &nodes) != nil {
		return
	}
	for _, s := range nodes {
		if err := dbMergeLWW(d.db, s); err != nil {
			log.Printf("MergeRemoteState: %v", err)
		}
	}
}

func (d *kvDelegate) broadcast(s NodeStats) {
	b, _ := json.Marshal(s)
	d.broadcasts.QueueBroadcast(&simpleBroadcast{b})
}

type simpleBroadcast struct{ msg []byte }

func (b *simpleBroadcast) Invalidates(other memberlist.Broadcast) bool {
	ob, ok := other.(*simpleBroadcast)
	if !ok {
		return false
	}
	var a, bv NodeStats
	if json.Unmarshal(b.msg, &a) != nil || json.Unmarshal(ob.msg, &bv) != nil {
		return false
	}
	return a.Name == bv.Name
}
func (b *simpleBroadcast) Message() []byte { return b.msg }
func (b *simpleBroadcast) Finished()       {}
