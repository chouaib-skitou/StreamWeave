# STORY-019 — Establish Orders service foundation

## Outcome

Create the Orders application boundary, service guide, configuration contract, API/relay lifecycle, and dependency ports without business logic leakage.

## Acceptance criteria

- [ ] Clean/Hexagonal package structure matches the blueprint.
- [ ] API and relay have independent lifecycle and health semantics.
- [ ] Domain imports no infrastructure package.
- [ ] Orders accepts only trusted Gateway service context.
- [ ] Architecture tests enforce dependency direction.

## Test plan

Architecture, configuration, startup/shutdown, health, and invalid-secret tests.

## API and event changes

Expose the internal Orders health/startup contract and prepare the versioned REST and Kafka adapters; no business event is emitted by boot.

## Data and failure behavior

Introduce configuration and transaction ports without schema ownership outside `orders_db`. Startup fails closed for invalid secrets, database, Kafka, or required telemetry configuration; liveness remains local.

## Observability and security

Emit lifecycle metrics and redacted structured logs. Run as non-root, reject unknown configuration, and never log credentials or payloads.

## Dependencies and out of scope

Depends on shared HTTP, Kafka, PostgreSQL, telemetry, and Gateway trust conventions. Business transitions, migrations, and downstream orchestration are subsequent stories.
