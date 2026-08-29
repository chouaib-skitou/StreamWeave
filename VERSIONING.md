# Versioning

Use one platform-wide Semantic Version for a release. Services remain independently deployable and may have separate image names, but artifacts produced from one platform release share the same source version.

- Breaking REST changes require a new API version.
- Breaking event changes require a new event type or schema version.
- Database migrations use expand, migrate, contract sequencing.
- Release tags must point to reviewed commits on `main`.
- Images and Helm packages should use immutable release references.
