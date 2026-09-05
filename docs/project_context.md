# Project Context — StreamWeave Event-Driven Commerce Platform

## Why this project exists

This repository is a portfolio-grade demonstration of designing and operating an event-driven distributed backend. It intentionally contains failure modes and operational concerns that a simple CRUD application would hide.

## Core scenario

A customer places an order. Inventory, payment, fulfillment and notification are independent services. No service may directly edit another service's database. Cross-service workflows are coordinated through versioned Kafka events.

## Architectural style

- modular monorepo;
- independently deployable services;
- Clean / Hexagonal Architecture inside each service;
- database-per-service;
- REST at the public boundary;
- Kafka for asynchronous integration;
- PostgreSQL as durable source of truth;
- Redis for ephemeral acceleration/coordination;
- Transactional Outbox for reliable event publication;
- idempotent Inbox/consumer processing;
- orchestrated Saga for order lifecycle;
- OpenTelemetry-first observability.

## Quality attributes, in order

1. correctness of business invariants;
2. recoverability after partial failure;
3. observability;
4. maintainability;
5. scalability;
6. developer experience;
7. raw throughput.

## Consistency model

Inside one service/database: strongly consistent local transaction where needed.
Across services: eventual consistency.
The API must expose intermediate states honestly rather than pretending the distributed workflow is synchronous.

## Reliability assumptions

- Kafka is at-least-once from the application's point of view.
- Events may be duplicated.
- Consumers may restart at any instruction boundary.
- Redis may disappear without durable-data loss.
- Network calls may timeout after the remote side has succeeded.
- Outbox relays may publish and crash before recording publication.
- Services are deployed independently.

## Security boundary

The project uses synthetic data in automated tests. Local Identity development uses MailHog for real SMTP capture, while shared environments use isolated Mailtrap SMTP destinations. Never store real secrets or cardholder data.

## Repository state

The repository contains the approved Identity and Gateway implementations, deployment manifests, quality CI, tests, email templates, environment profiles, observability stack, and API/event contracts. Orders documentation and foundation implementation contracts are complete on the Orders documentation branch; application code follows through the reviewed feature flow. Dependency integration remains gated by the explicit Inventory, Payments, and Fulfillment contracts.

The private BMAD installation, local agent instructions, original blueprint, BMAD guide, and generated BMAD output remain local-only and are excluded by `.gitignore`.

## Delivery governance

Human changes follow:

```text
short-lived branch → pull request to dev → pull request to main → merge to main
```

`main` and `dev` block force-pushes and branch deletion and require resolved review conversations. Approval is recommended but not mandatory for this solo-maintainer repository. The release workflow is a documented machine-owned exception: after a merge to `main`, Semantic Release updates `VERSION` and `CHANGELOG.md`, publishes the platform tag and GitHub Release, and synchronizes those two files to `dev`.

The release workflow does not replace application CI. Quality checks become authoritative only when their Makefile, scripts, contracts, and workflows exist and are required by the repository rules.

## Canonical documentation map

| Concern | Location |
|---|---|
| Product brief | `docs/brief.md` |
| PRD | `docs/prd/` |
| Architecture | `docs/architecture/` |
| ADRs | `docs/architecture/decisions/` |
| Epics and stories | `docs/stories/` |
| API and event contracts | `contracts/` (Identity baseline available) |
| Runbooks | `docs/runbooks/` |
| BMAD operating guide | local `docs/bmad/workflow.md` |

## Interview narrative

For every significant design choice, be able to explain:

- requirement;
- constraint;
- alternatives;
- decision;
- trade-off;
- failure mode;
- how the system is observed;
- what you would change at 10x scale.

## Current milestones

M0: context, product requirements, architecture, ADRs, and contracts plan.
M1: Identity and Gateway implementation baselines.
M2: Orders documentation and contract gate.
M3: Orders implementation, Kafka, Inbox, and Outbox.
M4: Inventory/payment/fulfillment Saga integration.
M5: compensation, DLQ, and recovery.
M6: shared observability and SLO evidence.
M7: Kubernetes, autoscaling, chaos, and load testing.
M8: architecture review and interview demo.
