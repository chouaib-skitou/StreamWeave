# Staging Identity Environment

Staging uses the Mailtrap Email Sandbox. Configure the SMTP username and
password through a secret manager or a CI/CD secret store; do not commit the
values file with credentials. The sandbox captures every email for inspection
and must never deliver to real customers.

The NetworkPolicy allows SMTP only to the explicit Mailtrap CIDRs in
`values.yaml`. Refresh those `/32` entries from the current DNS records before
deployment if Mailtrap changes the Sandbox endpoint addresses.

Deploy with the shared Identity Helm chart and this file as an override:

```bash
helm upgrade --install identity deploy/identity/helm \
  --namespace identity-staging --create-namespace \
  -f deploy/identity/staging/values.yaml \
  --set-string secret.smtpUsername="$MAILTRAP_SANDBOX_USERNAME" \
  --set-string secret.smtpPassword="$MAILTRAP_SANDBOX_PASSWORD"
```
