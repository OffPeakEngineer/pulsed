---
id: task-20260518-node-health-ttl-config
title: Make node health TTL configurable
status: 3_done
type: feature
priority: normal
effort: cake
creator: codex
owner: ""
created: 2026-05-18
forked_from: task-20260517-health
---

## Problem

Fresh, stale, and offline rendering now uses a TTL published by the origin node,
but the origin currently only publishes psstd's built-in default. Operators
still cannot tune how long a node wants its last heartbeat to be considered
fresh or stale.

## Done when

- A node-local configuration value controls the TTL published in that node's heartbeat
- Empty configuration preserves the current default
- Invalid TTL values fail clearly or fall back with an explicit warning
- README documents when and how to tune the TTL
- Tests cover default, configured, and invalid TTL behavior

## Result

- Added `PSSTD_NODE_TTL` duration parsing with a `15s` default and clear invalid-value failures
- Published the configured TTL in each node heartbeat
- Documented TTL tuning in README
- Added unit coverage for default, configured, invalid, and heartbeat-published TTL behavior
