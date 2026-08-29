# Versioning and Releases

The platform uses Semantic Versioning and Conventional Commits. Human changes follow the protected Git flow:

```text
feature branch
    → pull request to dev
    → pull request to main
    → merge to main
```

After a merge to `main`, GitHub Actions publishes the release directly. No release pull request or manual Codex action is required.

## Automatic release flow

```text
Merge reviewed PR into main
          ↓
Semantic Release analyzes Conventional Commits
          ↓
VERSION and CHANGELOG.md are updated and committed to main
          ↓
vX.Y.Z tag and GitHub Release are published
          ↓
VERSION and CHANGELOG.md are synchronized directly to dev
```

The release commit and the synchronization commit are machine-owned exceptions to the human PR flow. The workflow authenticates with the maintainer's `RELEASE_TOKEN` repository secret. This is required because GitHub does not allow the built-in GitHub Actions integration to bypass rulesets on a personal repository.

## Version source

The platform version is stored in [`VERSION`](VERSION). The current baseline is `0.1.1`.

The root `package.json` is CI-only release tooling and is not the platform version source.

## Version format

```text
MAJOR.MINOR.PATCH
```

- `MAJOR`: incompatible API, event, data, or operational contract change;
- `MINOR`: backward-compatible capability (`feat`);
- `PATCH`: bug fix, performance, security, or internal correction.

Breaking changes require `!` or a `BREAKING CHANGE:` footer and must include migration notes.

## Commit-to-release policy

| Commit | Release impact |
|---|---|
| `feat(scope): ...` | Minor |
| `fix(scope): ...` | Patch |
| `perf(scope): ...` | Patch |
| `refactor(scope): ...` | Patch |
| `security(scope): ...` | Patch |
| `type(scope)!: ...` or `BREAKING CHANGE:` | Major |
| `docs`, `style`, `test`, `build`, `ci`, `chore` | No release |

Merge commits are ignored by the release analyzer. Every human commit must still follow [`COMMITS_CONVENTIONS.md`](COMMITS_CONVENTIONS.md).

## Release artifacts

Each published release contains:

- updated `VERSION`;
- generated `CHANGELOG.md` entry;
- immutable tag `vX.Y.Z`;
- GitHub Release notes;
- synchronization of `VERSION` and `CHANGELOG.md` to `dev`.

Never delete or rewrite a published tag. Correct a release with a new Conventional Commit and a new version.

## Configuration

- Workflow: `.github/workflows/release.yml`;
- Semantic Release configuration: `.releaserc.cjs`;
- Version writer: `scripts/update-version.cjs`;
- CI-only dependencies: `package.json` and `package-lock.json`.

Release automation is direct by design. Branch protections still require pull requests for normal contributors; the sole repository administrator is exempted so the workflow's maintainer token can publish the two release metadata commits. Force-pushes and branch deletion remain disabled.
