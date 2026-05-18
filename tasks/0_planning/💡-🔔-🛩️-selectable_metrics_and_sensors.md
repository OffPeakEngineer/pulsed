---
id: task-20260518-selectable-metrics-sensors
title: Plan selectable metrics and host sensors
status: 0_planning
type: feature
priority: normal
effort: trip
creator: codex
owner: ""
created: 2026-05-18
---

## Problem

psstd currently renders a fixed set of metrics for every node. Real hosts may
have useful extra sensors, such as temperature, fan speed, power, battery, UPS,
HID, or I2C-attached devices, and not every node will support the same readings.

Operators should eventually be able to choose which metrics appear in the
dashboard and terminal view without making unsupported node sensors look broken.

## Planning questions

- Which metrics are core and always shown?
- Which metrics are optional per-node capabilities?
- Should metric selection be a browser preference, node-local config, cluster-wide config, or a mix?
- How should the UI represent a selected metric that only some nodes can report?
- Which operating-system sensor APIs are reliable enough on Linux, macOS, Windows, and containers?
- Which Go libraries or native command integrations should be considered for sensor discovery and reading?
- How often should slow or expensive sensor reads run compared with CPU/memory/load heartbeats?
- How should sensor values be normalized, named, unit-tagged, and versioned in the gossip payload?

## Done when

- Existing CPU, memory, load, age, and health metrics have an explicit display-selection model
- Optional metrics can be hidden or shown without adding server API surface unless the design justifies it
- Sensor capability discovery is defined separately from sensor reading
- Unsupported metrics render as unavailable, not stale/offline
- Sensor payloads include stable names, units, values, and collection timestamps
- The first implementation scope is small enough for one MR
- Follow-up tickets exist for OS-specific sensor support after research
