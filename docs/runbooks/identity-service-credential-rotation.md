# Identity Service Credential Rotation Runbook

## Purpose

Rotate a machine credential without exposing the secret, interrupting service calls, or accepting the retired credential beyond the bounded overlap window.

## Preconditions

- Confirm the target service principal, current credential ID, replacement owner, and maintenance window.
- Generate the replacement secret in the approved secret manager or local secret-file provider. Never place it in Git, logs, tickets, shell history, or chat.
- Confirm the service principal remains `ACTIVE` and the replacement scope grant is unchanged or explicitly reviewed.

## Procedure

1. Provision the replacement credential as `ACTIVE` with a future or immediate `valid_from` timestamp.
2. Deliver the replacement secret to the target service through its secret reference and restart or reload the client safely.
3. Verify a service-token request succeeds with the replacement credential, the machine audience is correct, and the requested scopes are a subset of the principal grant.
4. Keep the previous credential active only for the documented overlap window. Monitor authentication failures and service-token issuance outcomes.
5. Retire the previous credential by setting `status=RETIRED` and `retired_at`; do not delete its audit or operational history.
6. Verify the old credential receives a generic `401`, the replacement remains valid, and no raw credential appears in logs or audit metadata.
7. Record `service_credential.rotated` as a sanitized audit action with the principal and credential IDs only.

## Failure and rollback

- If the replacement fails before retirement, keep the previous credential active, remove the failed secret reference, and investigate without logging the secret.
- If the previous credential was retired prematurely, restore only its validity state through the approved administrative path, then repeat verification and retirement with a controlled overlap.
- If compromise is suspected, retire both credentials, disable the service principal, revoke affected machine access at the consuming services, and follow the [security incident runbook](identity-security-incident.md).

## Validation checklist

- [ ] Replacement token issuance succeeds.
- [ ] Requested scopes outside the grant return `403` and issue no token.
- [ ] Previous credential is rejected after retirement.
- [ ] No secret, token, or private key is present in logs, traces, events, or configuration artifacts.
- [ ] Audit and metrics show the rotation without exposing sensitive values.
