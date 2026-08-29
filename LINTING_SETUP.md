# Linting and Local Quality

The implementation target is Go 1.26.x with formatting, static analysis, contract validation, and security checks running in CI as those tools are introduced.

## Rules

- Keep `gofmt` output clean.
- Keep domain packages independent from infrastructure.
- Do not hand-edit generated sqlc code.
- Do not add a generic abstraction without two stable consumers and a documented need.
- Keep HTTP and Kafka DTOs outside domain entities.
- Wrap errors with meaningful context and avoid panics for expected runtime errors.

The exact project commands belong in the root `Makefile` and CI workflows once implementation begins; this document records the quality expectations, not a second command source of truth.
