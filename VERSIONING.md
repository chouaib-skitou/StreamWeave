# Versioning and Releases

The platform uses Semantic Versioning for coordinated releases. The current release automation is implemented with Release Please and runs from pushes to protected `main`.

## Version source

The current platform version is stored in [`VERSION`](VERSION). The initial development baseline is `0.1.0`.

All services may have separate image names, but artifacts from one platform release share the same platform version. Image digests are preferred over mutable tags for deployment.

## Automatic release flow

```text
Reviewed changes merged into main
          ↓
Release Please workflow runs
          ↓
Release PR updates VERSION and CHANGELOG.md
          ↓
Release PR is reviewed and merged
          ↓
GitHub Release + vX.Y.Z tag are created automatically
```

The workflow is PR-first so it remains compatible with the repository rule that `main` cannot receive direct pushes. A push to `main` creates or updates one release pull request; merging that generated PR publishes the release. The generated release commit does not create an infinite workflow loop.

## Version format

```text
MAJOR.MINOR.PATCH
```

- `MAJOR`: incompatible API, event, data, or operational contract change;
- `MINOR`: backward-compatible capability;
- `PATCH`: backward-compatible bug fix, performance, security, or internal correction.

Breaking REST changes require a new API version. Breaking event changes require a new event type or schema version even when the platform version also changes.

## Commit-to-release policy

Release Please analyzes English Conventional Commits:

| Commit | Release impact |
|---|---|
| `feat` | Minor |
| `fix`, `perf`, `refactor`, `security` | Patch when recognized by the configured release rules |
| `!` or `BREAKING CHANGE` | Major |
| `docs`, `style`, `test`, `build`, `ci`, `chore` | No release by default |
| `revert` | Patch when it changes released behavior |

Several breaking changes in one release produce one major version. Every breaking change must include migration notes and affected consumers.

## Development versions

Keep the platform below `1.0.0` while its public API and event contracts evolve. Document breaking changes and migration impact even before the first stable release. Choose `1.0.0` deliberately after the core order lifecycle, contracts, recovery behavior, and release evidence are validated.

## Release artifacts

Each published release should contain, as applicable:

- `VERSION` updated to the released SemVer;
- a generated `CHANGELOG.md` entry;
- a GitHub Release;
- an immutable tag such as `v1.2.3`;
- immutable container image references;
- Helm package references;
- migration and rollback notes;
- SBOM and provenance evidence when the CI pipeline supports them.

Never delete or rewrite a published tag or release. Fix a release through a new commit and a new version.

## Configuration and maintenance

- Workflow: `.github/workflows/release.yml`;
- Release configuration: `release-please-config.json`;
- Version manifest: `.release-please-manifest.json`;
- Commit rules: `COMMITS_CONVENTIONS.md`.

Keep the action version and configuration maintained through reviewed pull requests. Before enabling new release outputs such as container images, add their build and security gates to the release workflow.
