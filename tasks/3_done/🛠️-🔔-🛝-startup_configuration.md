---
id: task-20260517-cli
title: Print a clear startup configuration
status: 3_done
type: maintenance
priority: normal
effort: walk
creator: codex
owner: ""
created: 2026-05-17
---

## Problem 1

psstd is easiest to like when it feels obvious what it is doing: which database it owns, which HTTP URL it advertises, which gossip address it listens on, and whether it joined peers. Today that information exists in logs, but it is not shaped as a concise operator-facing summary.

## Done when

- Startup logs show DB path, HTTP listen address, advertised URL, gossip listen address, web enabled state, and version
- Seed and mDNS discovery results are summarized without noisy repetition

## Result

- Added a concise startup summary including version, node, DB path, web state, HTTP listen address, advertised URL, gossip address, explicit seed count, and mDNS discovery count
- Split the CLI mode flag cleanup into `task-20260518-cli-mode-flags` for planning

## Forked follow-up

- `task-20260518-startup-join-outcome-summary`: include the actual join outcome in the operator-facing startup summary
