# Orders Security Design

## Data classification

| Data | Class | Rule |
|---|---|---|
| Order ID, status, timestamps | operational | safe when bounded and authorized |
| Customer reference | personal identifier | access-controlled; never a metric label |
| Address/contact snapshot | PII | encrypt at rest/in transit, minimize, redact from logs/events |
| Prices/totals | business-sensitive | integrity protected; no client authority |
| Payment reference | sensitive reference | store external ID only; no PAN/CVV/token |
| Auth context | secret-adjacent | validate, never log bearer material |

## Required controls

- Gateway machine authentication and trusted actor binding.
- TLS for HTTP, PostgreSQL, Kafka, and production telemetry; authenticated Kafka producers/consumers with topic ACLs.
- Strict JSON/body/header/query bounds and safe Problem Details errors.
- Explicit authorization at route and use-case levels; no IDOR or enumeration.
- Argon-free Orders: it never owns passwords or keys. Secrets come from external secret references.
- Redaction in logs, traces, metrics, Outbox/DLQ diagnostics, and support tooling.
- Audit privileged reads, cancellation, repair, DLQ replay, and compensation intervention through Orders' local audit row and transactional Outbox, using the shared security-audit schema with `producer=orders`.
- Dependency and container scanning, SBOM/provenance, non-root runtime, read-only filesystem, dropped capabilities, and pinned actions/images.
