---
id: task-20260517-health
title: Distinguish stale nodes from offline nodes
status: 3_done
type: feature
priority: normal
effort: walk
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

Offline nodes and recently stale nodes can look too similar. Operators should quickly see whether a node just went quiet or has been offline long enough to treat as gone.

## Done when

- Fresh, stale, and offline states have distinct rendering
- The staleness threshold is controlled by the origin node (each node controls the 'time' it'd like to be known for)
- Terminal and web views use the same state calculation
- Existing offline purge/version behavior still works

## Result

- Added node heartbeat TTL metadata and shared fresh/stale/offline state calculation
- Terminal and web rendering now use the shared health state
- Offline purge/version behavior still flows through the same `nodeRecordOffline` helper

## Forked follow-up

- `task-20260518-node-health-ttl-config`: expose and validate the origin node's TTL as configuration instead of only publishing the built-in default
