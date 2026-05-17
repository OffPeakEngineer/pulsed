# Tasks

This folder is a Patchboard task board. Tasks are Markdown files, and the
folder containing a task is its workflow state.

## States
- -1_anti-feature
- 0_planning
- 1_todo
- 2_doing
- 3_done

Move a task file between folders to change its state. Git history is the audit
trail.

`-1_anti-feature` is for explicit non-goals: ideas that may sound useful but
would make psstd heavier, riskier, or less focused. Keep these as reference
points so future planning can explain why the project is not taking that path.
Anti-feature files still use the same filename tags and frontmatter shape.

## Task Shape
Each task gets assigned these three tags to help with high-level sorting and easy prioritization:
- Type: 🐞(bug)💡(feature)🔒(security)🛠️(maintenance)
- Priority:🔥(fire)⚠️(urgent)🔔(normal)⏳(idle)🧊(low)
- Effort: ⛰️(huge)🛩️(a trip)🏕️(a night)🍰(a piece of cake)🛝(a walk in the park)


### File Name Convention
[type]-[priority]-[effort]-[title_in_snake_case].md
Example: "💡-🧊-🛩️-historical_spark_lines.md"

Use exactly one emoji from each category, in this order. Do not omit the type
tag, and do not swap priority and effort.


### File Body
~~~markdown
---
id: task-YYYYMMDD-short-name
title: Short, concrete task title
status: 0_planning
type: feature
priority: normal
effort: walk
owner: your-name
created: YYYY-MM-DD
---

## Problem

What needs to change, and why?

## Done when

- The expected behavior is implemented
- Relevant tests or checks pass
~~~

The folder is authoritative for status. If frontmatter includes `status`,
it should match the parent folder exactly, such as `0_planning`.

## Code Annotations

Link code comments back to tasks with square brackets:

~~~text
TODO[task-YYYYMMDD-short-name]: describe the follow-up
FIXME[task-YYYYMMDD-short-name]: describe the known problem
~~~

Unlinked annotations such as `TODO:`, `XXX:`, and `WARN:` are useful inventory,
but they do not fail lint until they reference a task ID.
