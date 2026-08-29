# Local Identity Environment

This profile runs Identity with PostgreSQL, Redis, Kafka, OpenTelemetry, and
MailHog. MailHog is a real local SMTP server: Identity sends the same multipart
HTML/text emails used by the SMTP deployment rather than logging or simulating
tokens.

```powershell
Copy-Item .env.example .env
docker compose --env-file .env -f compose.yaml up --build
```

- Identity API: `http://localhost:8080`
- MailHog inbox: `http://localhost:8025`
- MailHog SMTP: `localhost:1025` (container address: `identity-mailhog:1025`)

Use the Postman collection in `contracts/postman`. The collection reads the
verification and password-reset tokens from the MailHog API, then calls the
Identity confirmation endpoints.
