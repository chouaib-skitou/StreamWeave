# Identity Deployment Profiles

| Profile | Mail transport | Purpose |
|---|---|---|
| `local/` | MailHog SMTP (`localhost:1025`) | Real local email capture and HTML preview at `localhost:8025` |
| `staging/` | Mailtrap Email Sandbox | Safe shared-environment inspection without delivery to customers |
| `production/` | Mailtrap Email Sending | Verified-domain transactional delivery |

All profiles use the shared StreamWeave Identity image and Helm chart. The
canonical registry coordinate is
`ghcr.io/chouaib-skitou/streamweave-identity:v<version>`. Only the transport
configuration changes. The local Compose definition lives under `local/`; the
root directory contains only shared deployment assets and the Helm chart. SMTP credentials are injected through local ignored
`.env` files, CI/CD secret stores, or Kubernetes Secrets; they are never
committed.

The canonical local commands are documented in `local/README.md` and exposed
through `make identity-compose-up` and `make identity-compose-down`.
