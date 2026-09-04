# Runbook: Gateway Redis rate-limit outage

## Symptoms

- `gateway_rate_limiter_errors_total` increases.
- Anonymous authentication and protected mutations return safe `503` responses according to policy.
- Health may remain locally live while readiness reflects inability to enforce required policy.

## Diagnosis

1. Check Redis reachability, TLS, authentication, connection pool saturation, and latency.
2. Confirm only the Gateway rate-limit namespace is affected; Redis is not a source of durable business state.
3. Inspect route/limit-class metrics, not raw keys containing subjects or IPs.

## Recovery

1. Restore Redis capacity or connectivity.
2. Confirm TTL-bearing namespaced operations are working.
3. Verify `429` behavior with a controlled test and confirm protected traffic no longer fails closed unnecessarily.

## Safety

Do not switch to unlimited fallback in production. Any emergency policy change requires an explicit reviewed configuration and incident record.
