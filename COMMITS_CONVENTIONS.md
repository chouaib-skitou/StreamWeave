# Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/) with a concise imperative subject:

```text
<type>(optional-scope): short description
```

Allowed types:

- `feat`: user-visible or system capability;
- `fix`: bug correction;
- `docs`: documentation only;
- `test`: tests only;
- `refactor`: behavior-preserving structural change;
- `perf`: performance improvement;
- `build`: build or dependency changes;
- `ci`: CI/CD changes;
- `chore`: maintenance and repository work.

Use a body when the reason, trade-off, migration, or failure mode is not obvious from the subject. Breaking changes must be called out explicitly with `!` or a `BREAKING CHANGE:` footer.
