---
id: task-20260517-persist-metrics
title: Consider optional local history retention
status: 0_planning
type: feature
priority: low
effort: trip
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

The current model keeps recent gossip state only. Operators sometimes want to know whether a node was hot earlier, but persistent history can grow into a metrics product if the scope is not constrained.

## Done when

- A minimal retention format is proposed, such as compact local hourly snapshots
- Retention is bounded by count, age, or disk budget
- No HTTP history API is added unless the project intentionally reopens that surface
- The feature remains optional and has no dependency on external storage
