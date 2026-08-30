# Production Identity Environment

Production uses Mailtrap Email Sending as the SMTP relay. Configure a verified
sending domain and inject `IDENTITY_SMTP_PASSWORD` from a secret manager. The
password value is an API token, while the SMTP username is `api`.

The production Helm values keep the local test mailbox disabled, use two
replicas, run migrations as a pre-upgrade Job, and read credentials from an
ExternalSecret-managed Kubernetes Secret named `identity-external`. The
Secret must contain `IDENTITY_DATABASE_URL`, `IDENTITY_REDIS_URL`,
`IDENTITY_SMTP_USERNAME`, `IDENTITY_SMTP_PASSWORD`, and `signing-key.pem`.
The separate `identity-signing-key-history` Secret contains retained public
keys named `<kid>.pub.pem` during the rotation overlap window.
Review sender domain, SPF/DKIM/DMARC, alerting, rate limits, and key rotation
before rollout.

```bash
helm upgrade --install identity deploy/identity/helm \
  --namespace identity-production --create-namespace \
  -f deploy/identity/production/values.yaml
```

Create or synchronize `identity-external` with the approved secret manager
before running Helm. Do not pass SMTP or database credentials through Helm
flags or commit them to values files.
