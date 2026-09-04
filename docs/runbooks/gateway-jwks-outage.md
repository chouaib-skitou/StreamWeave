# Runbook: Gateway JWKS or token-verifier outage

## Symptoms

- `gateway_jwks_refresh_total{outcome="error"}` increases.
- Protected traffic returns `503` or authentication failures increase.
- `/health/ready` may be `503` when the configured verifier policy cannot be enforced.

## Diagnosis

1. Check Gateway logs by `request_id`, `trace_id`, and `error_code`; never request or record a token.
2. Check Identity `/health/ready` and its JWKS endpoint from the Gateway network.
3. Verify issuer, audience, JWKS URL, TLS trust, DNS, and clock skew configuration.
4. Check whether the issue affects all `kid` values or only a newly rotated key.

## Recovery

1. Restore Identity/JWKS reachability or correct the environment configuration.
2. If a key rotation is in progress, confirm the public key is published before tokens signed with it are issued.
3. Restart only if the cache cannot recover through the bounded refresh path.
4. Validate one known-good token through a controlled staging test; do not use production customer credentials.

## Safety

Never disable signature verification, accept an unknown key, or add a symmetric fallback to restore availability. Record the incident and verify that no bearer value entered logs or traces.
