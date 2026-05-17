---
id: task-20260517-multicluster
title: Explore multi-cluster browser view
status: 0_planning
type: feature
priority: low
effort: huge
owner: ""
created: 2026-05-17
creator: "copilot"
---

## Problem

Some operators may run independent psstd clusters for environments like dev, staging, and prod. Switching between browser tabs is tedious, but adding cross-cluster behavior can easily make psstd feel like a control plane instead of a simple cluster view.

## Done when

- The intended use case is written down with a concrete example
- A design keeps clusters fully independent, with no cross-cluster gossip or shared state
- The browser URL can select a configured cluster without adding a server-side API
- The implementation can be deferred if it adds configuration or UI weight out of proportion to the benefit
