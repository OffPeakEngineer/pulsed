---
id: task-20260517-client-sort
title: Add dashboard sort and filter controls
status: 0_planning
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

- Dashboard can sort by name, CPU, memory, load, and freshness
- Optional filter can hide offline or stale nodes
- Preferences are stored in the browser, not the server
- The no-JavaScript server-rendered dashboard remains usable
