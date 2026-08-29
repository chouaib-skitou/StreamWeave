# Workspace Notes

This is a greenfield Go monorepo for an event-driven e-commerce order management platform. The intended deployable applications are `gateway`, `identity`, `orders`, `inventory`, `payments`, `fulfillment`, and `notifications`.

## Quick start

The repository does not yet contain application modules, a root `Makefile`, Docker Compose configuration, or CI workflows. Do not run or document commands as available until their files exist and have been verified.

Before implementation, read:

1. `docs/project_context.md`;
2. `docs/brief.md`;
3. the relevant PRD, ADR, story, and contract;
4. the applicable service-level instructions.

## Architecture reminders

- Keep domain logic independent of infrastructure.
- Each business service owns its database.
- Use REST at client and query boundaries and Kafka for asynchronous domain integration.
- Use Transactional Outbox for durable changes that publish events.
- Make consumers idempotent because delivery is at least once.
- Keep Redis disposable and never use it as the source of truth for orders, payments, inventory, or Saga state.
- Use integer minor units for money and RFC3339 UTC timestamps.

## Development workflow

Create a short-lived branch from `dev`:

```text
feat/<short-name>
fix/<short-name>
docs/<short-name>
chore/<short-name>
```

Use English Conventional Commit messages and open a pull request into `dev`. Promote validated work to `main` through a separate pull request. Do not commit directly to protected branches.

## Validation expectations

When implementation exists, validation should cover domain invariants, persistence, contracts, duplicate delivery, timeouts after remote success, compensation, observability, authorization, migrations, and deployment behavior. Use the canonical commands defined by the Makefile and CI rather than copying commands into this file.

## Local-only material

BMAD installation files, agent skills, the original blueprint, agent instructions, BMAD run output, and the local `docs/bmad-v6-guide.md` are intentionally ignored by Git and remain available only in the local workspace.
