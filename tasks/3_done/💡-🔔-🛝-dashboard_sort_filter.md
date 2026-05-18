---
id: task-20260517-client-sort
title: Add dashboard sort and filter controls
status: 3_done
type: feature
priority: normal
effort: walk
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

Large clusters benefit from sorting by CPU% or memory to spot hot nodes quickly. Reloading the dashboard resets the sort order.

## Done when

- As long as the JS remains small and inside one template file.
- Dashboard can sort by any of the metrics populated.
- Optional filter can hide offline or stale nodes
- Preferences are stored in the browser, not the server

## Result

- Added client-side sort controls for name, CPU, memory, load, and age
- Added browser-persisted stale/offline filters
- Kept the implementation in the dashboard template without adding server API surface

## Forked follow-up

- `task-20260518-dashboard-metric-sort-completeness`: finish or explicitly define the full set of populated metrics that should be sortable
