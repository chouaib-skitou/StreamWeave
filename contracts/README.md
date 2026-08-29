# Contracts

This directory is the canonical source for versioned platform contracts.

## Planned structure

- `openapi/` — versioned REST contracts for the gateway and services;
- `events/` — versioned Kafka event envelopes and JSON Schemas.

Contracts must be reviewed before implementation, remain compatible within a version, and define ownership, authentication, authorization, idempotency, failure behavior, and migration strategy. Breaking changes require a new API or event schema version and an associated migration plan.

The repository is currently greenfield; no executable API or event contract has been added yet.
