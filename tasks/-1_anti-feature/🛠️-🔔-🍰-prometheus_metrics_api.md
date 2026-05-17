---
id: task-20260517-metrics-export
title: Do not add a Prometheus metrics API by default
status: -1_anti-feature
type: maintenance
priority: normal
effort: cake
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

External monitoring tools can already monitor hosts directly, and psstd recently removed superfluous HTTP endpoints. A Prometheus-style export may be useful later, but adding `/metrics` now would expand the public surface area again.

## Done when

- README is clear that the HTTP surface is the dashboard, not a general metrics API
- If metrics export is still desired, the ticket describes a narrow opt-in design before implementation
- The design covers whether export should be HTTP, file-based, or left to existing host monitoring
- No unauthenticated endpoint is added by accident
