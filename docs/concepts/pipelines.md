---
title: Pipelines
description: Reactive pipeline dependency graph
---

# Pipelines

A pipeline is the dependency graph derived from Transform and DataSet manifests
(`dk.yaml` files). Each Transform declares its inputs and outputs; the graph
is built automatically by scanning those declarations.

## Overview

There is no separate pipeline manifest. The graph emerges from the individual
Transform and DataSet manifests already present in your project:

- **Transforms** declare `spec.inputs` and `spec.outputs` (DataSet references).
- **DataSets** are the nodes that connect transforms together.
- **Triggers** on each Transform control when it runs (schedule, on-change, manual).

## Viewing the Graph

```bash
# Show full dependency graph
dk pipeline show

# Show graph leading to a specific destination DataSet
dk pipeline show --destination event-summary

# Render as Mermaid diagram
dk pipeline show --output mermaid

# Render as Graphviz DOT
dk pipeline show --output dot

# JSON adjacency list
dk pipeline show --output json

# Scan specific directories
dk pipeline show --scan-dir ./transforms --scan-dir ./assets
```

## Output Formats

| Format    | Description                          |
|-----------|--------------------------------------|
| `text`    | Text tree (default)                  |
| `mermaid` | Mermaid diagram                      |
| `json`    | JSON adjacency list                  |
| `dot`     | Graphviz DOT format                  |

## Scheduling

Scheduling is configured via the `trigger` field on a Transform manifest:

```yaml
# In dk.yaml (Transform)
spec:
  trigger:
    policy: schedule
    schedule:
      cron: "0 6 * * *"
      timezone: America/New_York
```

| Field                         | Required | Default | Description                                                    |
|-------------------------------|----------|---------|----------------------------------------------------------------|
| `trigger.policy`              | Yes      | —       | Trigger policy (schedule, on-change, manual, composite)        |
| `trigger.schedule.cron`       | Yes*     | —       | Standard 5-field cron expression (* when policy is schedule)   |
| `trigger.schedule.timezone`   | No       | UTC     | IANA timezone for cron evaluation                              |

## CLI Commands

| Command              | Description                        |
|----------------------|------------------------------------|
| `dk pipeline show`   | Display pipeline dependency graph  |
