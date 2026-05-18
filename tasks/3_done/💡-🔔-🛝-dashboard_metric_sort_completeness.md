---
id: task-20260518-dashboard-metric-sort-completeness
title: Complete dashboard metric sorting
status: 3_done
type: feature
priority: normal
effort: walk
creator: codex
owner: ""
created: 2026-05-18
forked_from: task-20260517-client-sort
---

## Problem

The dashboard now sorts by name, average CPU, memory percentage, load1, and age.
The original ticket said the dashboard should sort by any populated metrics, but
the implementation does not yet cover the full node snapshot shape or explicitly
define which metrics are intentionally sortable.

## Done when

- The sortable metric set is explicitly defined in code and UI
- Populated node metrics are either sortable or intentionally excluded in the ticket result
- Load sorting covers the load values users expect, not only load1 unless that is the intended scope
- CPU sorting handles the desired aggregate, such as average and/or max core
- Tests cover rendered sort data for each supported metric

## Result

- Sort controls now cover name, CPU average, CPU max, memory percent, memory used, memory total, load 1m, load 5m, load 15m, and age
- Dashboard cards render sortable data attributes for each supported metric
- Tests cover the rendered sort options and metric data
