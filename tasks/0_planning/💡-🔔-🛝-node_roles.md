---
id: task-20260517-labels
title: Add optional node roles
status: 0_planning
type: feature
priority: normal
effort: walk
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

Large heterogeneous clusters are harder to scan when every host looks equivalent. A small optional role label can help operators compare similar nodes without adding inventory management.

## Done when

- Optional `PSSTD_ROLE` environment variable is read, such as `db`, `worker`, or `cache`
- Role travels with gossip broadcasts as part of `NodeStats`
- Dashboard and terminal views render the role compactly
- Empty role preserves the current display
