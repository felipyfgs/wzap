---
name: internal-tools
description: Design, implement, or extend internal tools that help employees operate the system safely and efficiently, while respecting access controls and audit requirements. Use when building for internal staff interacting with production-adjacent systems.
---
# Skill: Internal tools development

## Purpose

Design, implement, or extend internal tools that help employees operate the system safely and efficiently, while respecting access controls and audit requirements.

## When to use this skill

- The audience is **internal staff** (engineers, SREs, support, operations, etc.).
- The tool interacts with **production-adjacent systems** (feature flags, incidents, customer data, sessions, etc.).
- The change is scoped to internal workflows and does not directly alter customer-facing UX.

## Inputs

- **User personas** and teams who will use the tool.
- **Workflows** to support (create/update actions, approvals, review flows).
- **Systems touched**: services, queues, flags, and data stores.
- **Risk classification**: what can go wrong if the tool misbehaves or is misused.

## Out of scope

- Tools that require new identity providers or SSO integrations.
- Changes that bypass existing approval or change-management processes.
- Direct manual-write tooling for core financial or compliance systems without explicit approval.

## Conventions

- Use the **standard stack** already used in the repo (Fiber, zerolog, pgx, dto patterns).
- Apply **role-based access control** and logging patterns consistently.
- Prefer **read-only views and guarded actions** (confirmation dialogs, requiring justification text, etc.) for high-risk operations.

## Required behavior

1. Implement flows that make the happy path fast while making destructive actions clearly intentional.
2. Ensure all state changes are logged with **who**, **what**, and **when**, and link to existing audit/logging infrastructure.
3. Provide clear feedback on success, errors, and partial failures.
4. Design for operational debugging: include ids, timestamps, and links to related systems.

## Required artifacts

- Backend handler and service changes in the appropriate modules.
- **Automated tests** for critical operations (at least unit tests).

## Implementation checklist

1. Clarify workflow boundaries and risk level with stakeholders.
2. Identify existing components, endpoints, and patterns to reuse.
3. Implement the backend handlers, service logic, and data access using established abstractions.
4. Add safeguards: confirmations, rate limiting, or approvals depending on risk.
5. Wire up logging so usage and failures are visible.
6. Add or update tests.

## Verification

- `go test -v -race ./...`
- `golangci-lint run ./...`

The skill is complete when:

- Validation commands pass.
- Flows behave correctly in staging or an equivalent environment.
- Stakeholders can perform their target workflows without manual DB access or unsafe workarounds.

## Safety and escalation

- If an operation could cause **irreversible data loss or external customer impact**, require higher-level approvals.
- If you discover that existing internal tools bypass critical controls, document this clearly and escalate.
