# Contributing

## Before changing code or contracts

Read, in order:

1. `docs/project_context.md`;
2. the relevant brief, PRD, story, and architecture decision;
3. the relevant API or event contract;
4. the local service instructions when the target service exists.

## Branches and pull requests

Create a short-lived branch from the current `dev` branch:

```text
feat/<short-name>
fix/<short-name>
docs/<short-name>
chore/<short-name>
```

Open a pull request into `dev` for integration work. Promote validated work from `dev` to `main` through a separate reviewed pull request. Do not push directly to protected branches.

## Commit messages

Use Conventional Commits. Keep each commit focused and explain the user or system outcome in the pull request.

## Definition of done

- acceptance criteria are satisfied;
- unit, integration, contract, or end-to-end tests appropriate to the change pass;
- API and event contracts are updated when needed;
- generated code is refreshed through its generator;
- telemetry covers critical behavior;
- security and failure modes are addressed;
- architecture and decision records are updated when the design changes.
