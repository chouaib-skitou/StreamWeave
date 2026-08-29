# Production Identity Environment

Production uses Mailtrap Email Sending as the SMTP relay. Configure a verified
sending domain and inject `IDENTITY_SMTP_PASSWORD` from a secret manager. The
password value is an API token, while the SMTP username is `api`.

The production Helm values keep the local test mailbox disabled and require a
mounted signing key. Review sender domain, SPF/DKIM/DMARC, alerting, rate
limits, and secret rotation before rollout.

```bash
helm upgrade --install identity deploy/identity/helm \
  --namespace identity-production --create-namespace \
  -f deploy/identity/production/values.yaml \
  --set-string secret.smtpUsername=api \
  --set-string secret.smtpPassword="$MAILTRAP_SMTP_PASSWORD"
```
