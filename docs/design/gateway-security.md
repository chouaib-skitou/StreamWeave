# Gateway security checklist

- Explicit route allowlist; no open proxy.
- Strict method, path, content-type, body-size, and header validation.
- EdDSA-only JWT verification with issuer/audience/claim/key allowlists.
- One bounded JWKS refresh for an unknown `kid`, then fail closed.
- Strip spoofable internal headers before adding trusted context.
- No human bearer propagation to downstream services, logs, traces, or events.
- Route-aware Redis rate limits with fail-closed protected policies.
- No automatic retries for auth or mutations.
- Generic RFC 9457 errors and no internal topology disclosure.
- TLS for staging/production dependency connections; no production `sslmode=disable` equivalent.
- Non-root image, read-only filesystem where supported, minimal capabilities, and restrictive network policy.
- Secrets only through environment injection/secret references; never in Git.
