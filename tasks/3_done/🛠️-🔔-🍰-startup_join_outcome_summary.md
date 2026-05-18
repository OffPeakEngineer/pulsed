---
id: task-20260518-startup-join-outcome-summary
title: Include join outcome in startup summary
status: 3_done
type: maintenance
priority: normal
effort: cake
creator: codex
owner: ""
created: 2026-05-18
forked_from: task-20260517-cli
---

## Problem

Startup logging now prints a concise configuration summary and quiets repeated
mDNS discovery lines, but the actual join outcome is still reported separately.
The original operator-facing problem asks for the startup summary to make it
obvious whether psstd joined peers or is running solo.

## Done when

- The startup summary includes joined peer count or solo status
- Join warnings still include enough detail for troubleshooting
- Seed count, mDNS discovery count, and join result are not repeated noisily
- Tests cover startup summary formatting for joined, solo, and warning cases where practical

## Result

- Startup summary now logs solo, joined, or warning join status in the same operator-facing line as configuration
- Join warning summaries include joined count and error text
- Removed the separate joined/solo startup log line to avoid repetition
- Added tests for joined, solo, and warning startup summary formatting
