# Runbook: Gateway upstream outage

## Symptoms

- `gateway_upstream_requests_total{outcome="timeout"|"unavailable"}` increases.
- Clients receive `503` or `504` Problem Details.
- Gateway liveness remains healthy; readiness is not used as a proxy for every business dependency.

## Diagnosis

1. Identify the logical upstream service and templated route from metrics.
2. Follow `request_id` and `trace_id` into Identity or Orders logs.
3. Check service health, resource saturation, network policy, DNS, and TLS.
4. Determine whether any mutation may have been accepted before the connection failed.

## Recovery

1. Restore the owning service or its dependency.
2. Do not replay ambiguous mutations manually. Clients must use the documented idempotency/read-after-failure flow.
3. Confirm the Gateway is not retrying non-idempotent requests.
4. Close the incident after error rate and latency return to baseline.
