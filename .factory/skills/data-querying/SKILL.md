---
name: data-querying
description: Query internal data services to answer well-scoped questions, producing results and artifacts that are safe to share and easy to re-run. Use when stakeholders need metrics, trends, or data slices from internal warehouses, marts, or reporting APIs.
---
# Skill: Internal data querying

## Purpose

Query internal data services to answer well-scoped questions, producing results and artifacts that are safe to share and easy to re-run.

## When to use this skill

- A stakeholder asks for **metrics, trends, or slices** that rely on internal data.
- The answers can be derived from existing **warehouses, marts, or reporting APIs**.
- The request needs a **reproducible query** and not an ad-hoc manual export.

## Inputs

- **Business question**: one or two sentences describing what we want to know.
- **Time range and filters**: date boundaries, customer segments, environments, etc.
- **Source systems**: names of warehouses, schemas, or APIs to use.
- **Data sensitivity notes**: whether PII, financial data, or regulated data is involved.

## Out of scope

- Direct queries against production OLTP databases unless explicitly allowed.
- Creating new pipelines or ingestion jobs.
- Sharing raw PII or secrets outside approved destinations.

## Conventions

- Use the **preferred query layer** (repo layer in `internal/repo/`) instead of raw SQL when available.
- Follow established **naming and folder conventions** for saved queries or analysis notebooks.
- Respect internal **data classification and access control** policies.

## Required behavior

1. Translate the business question into a precise query spec (metrics, dimensions, filters).
2. Choose appropriate sources and explain tradeoffs if multiple options exist.
3. Write queries that are performant and cost-conscious for the target system.
4. Produce both **results** and a **re-runnable query artifact** (SQL or Go repo method).

## Required artifacts

- Query text (SQL or Go repo method) checked into the appropriate repo or folder.
- A short **analysis summary** capturing methodology, assumptions, and caveats.

## Implementation checklist

1. Clarify the business question, time range, and filters.
2. Identify the best data source(s) based on freshness, completeness, and governance.
3. Draft the query, validate it on a limited time window or sample.
4. Check for joins, filters, and aggregations that could distort the answer; fix as needed.
5. Save the query in the approved location with a descriptive name.
6. Capture results and summarize key findings and limitations.

## Verification

- `go test -v -race ./internal/repo/...`
- Manual spot checks against known benchmarks or historical reports.

The skill is complete when:

- The query runs successfully within acceptable time and cost bounds.
- Results match expectations or known reference points.
- The query and results are documented enough for another engineer or analyst to reuse.

## Safety and escalation

- If the query touches **sensitive or regulated data**, confirm that the destination is an approved location before including any sample rows.
- If you identify data quality issues, call them out prominently in the analysis summary.
