# Product Brief — Event-Driven E-commerce Order Management Platform

## Problem

Simple CRUD examples hide the hard parts of commerce: partial failure, duplicate messages, uncertain network outcomes, concurrent inventory changes, payment compensation, and operational recovery. This project demonstrates how to coordinate an order lifecycle across independent services while preserving business invariants.

## Product outcome

Build a portfolio-grade Go platform where a customer can create and observe an order while inventory, payment, fulfillment, notification, and authentication capabilities remain independently owned and deployable.

## Core experience

1. An authenticated customer submits an order through the public REST API.
2. The Orders service persists the order and an outbox record in one local transaction.
3. Versioned Kafka events coordinate inventory reservation, payment authorization, fulfillment, and notification.
4. The customer observes honest intermediate and terminal states.
5. Duplicate delivery, timeouts, dependency outages, and compensation are visible, testable, and recoverable.

## Scope

### In scope

- customer authentication and authorization;
- order creation, retrieval, and authorized cancellation;
- inventory products, stock, and reservations;
- simulated payment authorization and refund;
- fulfillment and shipment lifecycle;
- notification delivery logging;
- durable orchestrated Saga state;
- transactional Outbox and idempotent Inbox/consumer processing;
- versioned OpenAPI and event contracts;
- Docker Compose, kind, Helm, and OpenTelemetry-based observability;
- CI validation, security scanning, and release evidence.

### Out of scope

- real card processing or PCI scope;
- real customer email delivery;
- frontend implementation for the first version;
- production cloud accounts;
- GraphQL, service mesh, distributed databases, and event sourcing everywhere;
- abstractions without a demonstrated stable need.

## Quality priorities

1. Correctness of business invariants
2. Recoverability after partial failure
3. Observability
4. Maintainability
5. Scalability
6. Developer experience
7. Raw throughput

## Success evidence

- business services have explicit ownership and no cross-service database access;
- order creation returns `202 Accepted` and exposes asynchronous progress honestly;
- local durable changes and emitted events use the transactional outbox;
- consumers remain correct under duplicate and out-of-order delivery;
- payment and inventory failures trigger observable, idempotent compensation;
- contracts, tests, telemetry, deployment manifests, and runbooks support the architecture story.

## Delivery path

```text
brief → PRD → architecture and ADRs → epics and stories → implementation
→ tests and observability → deployment → validation evidence
```
