# Commit Conventions

All committed project artifacts use English. Every commit in this repository — application code, infrastructure, contracts, documentation, CI, and repository maintenance — follows Conventional Commits.

## Format

```text
<type>(<scope>): <imperative subject>

<optional body>

<optional footer>
```

Rules:

- `type` is required;
- `scope` is optional and identifies one service or repository area;
- use a short imperative English subject, sentence case, and no final period;
- keep the header at 72 characters or fewer and the subject at 50 characters when practical;
- keep one logical outcome per commit;
- wrap a detailed body at approximately 72 characters;
- explain context, trade-offs, operational impact, migrations, and validation when they are not obvious from the diff.

## Allowed types

| Type | Use it for |
|---|---|
| `feat` | A new capability or user/system behavior |
| `fix` | A bug correction |
| `docs` | Documentation only |
| `style` | Formatting only, with no behavior change |
| `refactor` | Internal restructuring with no intended behavior change |
| `perf` | A measurable performance improvement |
| `test` | Adding or changing tests |
| `build` | Dependencies, build tooling, or packaging |
| `ci` | Continuous integration or delivery |
| `chore` | Repository maintenance without runtime behavior |
| `revert` | Reverting an earlier commit |
| `security` | A security fix or hardening change |

## Scopes

Use one short kebab-case scope when it improves navigation. Prefer the owning service or boundary:

```text
gateway, identity, orders, inventory, payments, fulfillment,
notifications, contracts, database, deployment, observability,
ci, docs, repo
```

Do not use several unrelated scopes in one commit. Split the change when the parts can be reviewed or reverted independently.

## Examples

```text
feat(orders): Accept asynchronous order creation
fix(inventory): Make stock release idempotent
docs(architecture): Record the Saga ownership decision
test(payments): Cover duplicate refund commands
ci(github): Validate contract compatibility
security(identity): Rotate refresh token families on reuse
chore(repo): Keep private BMAD files ignored
```

Avoid vague messages such as `update`, `changes`, `misc`, or `fix bug`.

## Body and footer

Use the body to explain why the change exists and how it was validated. A useful structure is:

```text
Problem: What was wrong or missing?
Decision: What changed and why?
Impact: What changes for operators or consumers?
Validation: Which checks or scenarios passed?
```

Use footers for traceability:

```text
Closes #123
Refs #456
BREAKING CHANGE: Explain the incompatibility and migration path.
Co-authored-by: Name <email@example.com>
```

Never put secrets, tokens, private keys, exploit details, or sensitive customer data in a commit message.

## Breaking changes

Mark an incompatible API, event, data, or operational change with `!` and explain the migration in a `BREAKING CHANGE:` footer when details are needed:

```text
feat(contracts)!: Replace the v1 order status shape

BREAKING CHANGE: Consumers must read `status.code` instead of the
top-level `status` field. Migration is documented in the contract PR.
```

Breaking changes require a new API or event schema version and an ADR when the architecture or migration strategy is affected.

## Reverts and security changes

Use the standard revert form and identify the original commit:

```text
revert: feat(orders): Accept asynchronous order creation

This reverts commit abcdef1234567890.
```

Use `security` for a security-specific change and describe the affected boundary and mitigation without exposing exploit material.

## Tooling policy

Commit linting and release automation are planned CI capabilities. When introduced, configure them from this document rather than adding a second incompatible convention. Local hooks may assist a developer but are not the source of truth because `.git/hooks/` is not versioned.
