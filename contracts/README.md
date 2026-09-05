# Contracts

This directory is the canonical source for versioned platform contracts.

## Planned structure

- `openapi/` — versioned REST contracts for the gateway and services;
- `events/` — versioned Kafka event envelopes and JSON Schemas.

Contracts must be reviewed before implementation, remain compatible within a version, and define ownership, authentication, authorization, idempotency, failure behavior, and migration strategy. Breaking changes require a new API or event schema version and an associated migration plan.

The Identity API and security audit event contracts are implemented baselines. The Gateway facade contract and Orders API/event contract are approved implementation contracts; the remaining business-service contracts will be added before their implementation.

The first service contract is [`openapi/identity.openapi.yaml`](openapi/identity.openapi.yaml), with the security audit event at [`events/commerce.security.audit.v1.json`](events/commerce.security.audit.v1.json).

The public edge contract is [`openapi/gateway.openapi.yaml`](openapi/gateway.openapi.yaml). It routes only explicit Identity and Orders operations and does not change service ownership.

The Orders service contract is [`openapi/orders.openapi.yaml`](openapi/orders.openapi.yaml). Orders-owned and counterpart workflow schemas are catalogued in [`events/README.md`](events/README.md).
