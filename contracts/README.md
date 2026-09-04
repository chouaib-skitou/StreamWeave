# Contracts

This directory is the canonical source for versioned platform contracts.

## Planned structure

- `openapi/` — versioned REST contracts for the gateway and services;
- `events/` — versioned Kafka event envelopes and JSON Schemas.

Contracts must be reviewed before implementation, remain compatible within a version, and define ownership, authentication, authorization, idempotency, failure behavior, and migration strategy. Breaking changes require a new API or event schema version and an associated migration plan.

The Identity API and security audit event contracts are implemented baselines. The Gateway facade contract is now defined as the pre-implementation contract for the next service.

The first service contract is [`openapi/identity.openapi.yaml`](openapi/identity.openapi.yaml), with the security audit event at [`events/commerce.security.audit.v1.json`](events/commerce.security.audit.v1.json).

The public edge contract is [`openapi/gateway.openapi.yaml`](openapi/gateway.openapi.yaml). It routes only explicit Identity and Orders operations and does not change service ownership.
