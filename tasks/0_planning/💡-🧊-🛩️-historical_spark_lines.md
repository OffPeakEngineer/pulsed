---
id: task-20260517-sparklines
title: Add lightweight trend sparklines
status: 0_planning
type: feature
priority: low
effort: trip
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

Dashboard shows only current metrics. Users can't see if a node is trending up or down without checking periodically. This is especially valuable during load spikes.

## Done when

- Dashboard renders compact CPU and memory trends per node
- Trends use short local in-memory history first
- No persistent storage or new HTTP API is required for the first version
- Terminal view either renders the same trend or explicitly defers it
