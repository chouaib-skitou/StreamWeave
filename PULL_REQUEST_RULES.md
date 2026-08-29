# Pull Request Rules

These rules apply to all pull requests in the event-driven commerce platform.

## Scope and size

- Keep one clear outcome per pull request.
- Prefer small, reviewable vertical slices over broad mixed changes.
- Separate unrelated formatting, dependency, and refactoring work.
- Link the relevant issue, brief, PRD, story, ADR, contract, or runbook.

## Title

Use the same English Conventional Commit format as commits:

```text
feat(orders): Accept asynchronous order creation
```

Keep the title imperative, concise, sentence case, and free of a final period.

## Description

Every PR must explain:

- **Summary** — what changed and why;
- **Alternatives** — important options considered and rejected;
- **Review focus** — the files, invariants, contracts, or failure modes needing attention;
- **Testing** — exact checks run and their result;
- **Operational impact** — migrations, telemetry, rollout, rollback, and known risks.

## Required checks before requesting review

- [ ] Acceptance criteria are testable and satisfied.
- [ ] API, event, and data changes are documented.
- [ ] Failure handling covers timeout, retry, backoff, cancellation, idempotency, and duplicate delivery where relevant.
- [ ] Security and authorization impact is addressed.
- [ ] Tests and contract validation are added or updated.
- [ ] Critical behavior has logs, metrics, and traces.
- [ ] Generated code was regenerated from its source.
- [ ] No secrets, credentials, private keys, `.env` files, kubeconfig files, or real payment data are included.
- [ ] Documentation, ADRs, diagrams, and runbooks are updated when needed.

## Review conduct

Review comments must be specific, evidence-based, and constructive. Explain the risk or improvement requested. Authors should respond to every thread and resolve only after the concern is addressed.

## Merge policy

`main` and `dev` require reviewed pull requests. Direct pushes, force-pushes, and branch deletion are disabled. At least one approval and resolution of review conversations are required. Prefer squash merging for short-lived branches.
