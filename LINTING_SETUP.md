# Linting and Local Quality

The implementation target is Go 1.26.x. Formatting, static analysis, contract validation, dependency checks, and security scanning will become enforced CI checks as the corresponding source and workflows are added.

## Current repository state

This repository is still greenfield. No application module, root `Makefile`, CI workflow, or `.golangci.yml` is currently the source of truth for executable commands. Do not report a lint command as operational until its configuration and invocation exist in the repository.

## Required quality rules

- Keep Go code formatted with `gofmt`.
- Keep domain packages independent from HTTP, PostgreSQL, Kafka, Redis, Kubernetes, and OpenTelemetry.
- Keep HTTP and Kafka DTOs outside domain entities.
- Wrap errors with meaningful context and avoid panics for expected runtime errors.
- Do not hand-edit generated sqlc or contract code; update its source and regenerate it.
- Keep SQL under `database/<service>/queries` and migrations append-only after merge.
- Do not add an abstraction only to satisfy a pattern.

## Planned checks

Once Go modules exist, the project should add canonical Makefile or CI targets for at least:

```text
gofmt -l .
go vet ./...
go test ./...
golangci-lint run
```

The exact targets, versions, exclusions, generated-file policy, and integration-test prerequisites must be defined in the repository configuration when implemented. This document must not drift into a second command registry.

## Pre-commit policy

Local hooks can provide fast feedback, but they are not committed as `.git/hooks/` because Git does not version that directory. CI is authoritative. A documented bypass, if one is ever added, must be reserved for emergencies and must not bypass required remote checks.
