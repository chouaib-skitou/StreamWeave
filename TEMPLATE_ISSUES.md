# Issue Template

## Title

Issue titles use the same English Conventional Commit convention as commits:

```text
<type>(<scope>): <imperative description>
```

Allowed types include `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, and `security`. Use a short kebab-case scope such as `orders`, `contracts`, `database`, or `observability`.

Examples:

```text
feat(orders): Add customer cancellation flow
fix(inventory): Make reservation release idempotent
docs(architecture): Explain Saga compensation policy
```

## Context

<!-- Describe the problem, affected actor, current behavior, and why it matters. -->

## Desired outcome

<!-- Describe the user or system outcome, not only the implementation. -->

## Acceptance criteria

- [ ] Given ... when ... then ...
- [ ] ...

## Affected boundaries

- Services:
- API contracts:
- Event contracts:
- Database or migration:
- Deployment or observability:

## Failure and security considerations

<!-- Include timeout, retry, backoff, cancellation, idempotency, duplicate delivery,
authorization, sensitive data, and rollback concerns where relevant. -->

## Test plan

- [ ] Domain/unit tests
- [ ] Repository/integration tests
- [ ] Contract tests
- [ ] End-to-end or failure-path tests
- [ ] Manual validation evidence

## Out of scope

<!-- Explicitly state what this issue will not implement. -->

## Notes and references

<!-- Link the brief, PRD, story, ADR, diagram, runbook, or related issue. -->
