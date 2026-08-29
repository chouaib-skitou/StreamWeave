# Versioning and Releases

The platform uses Semantic Versioning for coordinated releases. The release source is the reviewed history of protected `main`, and GitHub Releases are the canonical human-readable release record.

Release automation is a planned capability for this greenfield repository. Until its configuration and workflow are present and verified, do not claim that a release runs automatically.

## Version format

```text
MAJOR.MINOR.PATCH
```

- `MAJOR`: incompatible API, event, data, or operational contract change;
- `MINOR`: backward-compatible capability;
- `PATCH`: backward-compatible bug fix, performance, security, or internal correction.

Breaking REST changes require a new API version. Breaking event changes require a new event type or schema version even when the platform version also changes.

## Commit-to-release policy

When release automation is introduced, it should use the following project policy:

| Commit | Release impact |
|---|---|
| `feat` | Minor |
| `fix`, `perf`, `refactor`, `security` | Patch |
| `!` or `BREAKING CHANGE` | Major |
| `docs`, `style`, `test`, `build`, `ci`, `chore` | No release by default |
| `revert` | Patch when it changes released behavior |

Several breaking changes in the same release produce one major version. A breaking change must include migration notes and the affected consumers.

## Development versions

Keep the platform below `1.0.0` while the public API and event contracts are still evolving. The team must still document breaking changes and migration impact. Choose the first stable `1.0.0` deliberately after the core order lifecycle, contracts, recovery behavior, and release evidence are validated.

## Release artifacts

A completed release should produce, as applicable:

- an annotated or signed version tag such as `v1.2.3`;
- a GitHub Release with generated notes;
- a `CHANGELOG.md` entry;
- immutable container image references;
- Helm package references;
- migration and rollback notes;
- SBOM and provenance evidence when the CI pipeline supports them.

All services may have separate image names, but artifacts from one platform release share the same platform version. Image digests are preferred over mutable tags for deployment.

## Automation design

The future release workflow may use Semantic Release, release-please, or an equivalent tool. The chosen tool must:

1. run only from reviewed `main` history;
2. use the commit policy in `COMMITS_CONVENTIONS.md`;
3. create one release for grouped changes;
4. update the changelog and tag atomically as far as the tool allows;
5. publish artifacts only after tests, contract checks, and security gates pass;
6. never delete or rewrite a published tag or release.

The repository is not yet configured with a release workflow. Add the configuration, permissions, dry-run validation, and rollback procedure in the same change when enabling automation.
