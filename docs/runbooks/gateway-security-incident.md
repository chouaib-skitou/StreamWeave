# Runbook: Gateway security incident

## Trigger examples

- Sudden invalid-token or rate-limit spikes.
- Suspected spoofed actor headers or bearer-token exposure.
- Unexpected route access, proxy behavior, or service-credential failures.

## Immediate actions

1. Preserve request IDs, trace IDs, timestamps, route templates, and reason classes; redact all credentials.
2. Restrict or remove affected ingress paths using reviewed deployment configuration.
3. Rotate the Gateway service credential through Identity if compromise is suspected.
4. Revoke affected Identity sessions/token families when the actor or signing key is implicated.
5. Inspect access logs for topology disclosure and secret leakage.

## Investigation questions

- Was an internal header accepted from the client?
- Was a token forwarded to an upstream or observability backend?
- Did a Redis failure create an unintended bypass?
- Did any mutation receive an unsafe retry?
- Were only documented route templates reachable?

## Closure

Patch and test the control, record the impact and evidence, rotate credentials/keys as required, and update the relevant ADR/runbook before restoring normal traffic.
