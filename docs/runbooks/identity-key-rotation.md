# Runbook: Identity Signing-Key Rotation

## Scope

Rotate Ed25519 signing keys without invalidating still-valid access tokens unexpectedly.

## Preconditions

- A new private key is provisioned through the approved secret-file or KMS provider.
- The new public key has a unique `kid`.
- The operator can inspect readiness, JWKS, issuance metrics, and verification errors.

## Procedure

1. Load the new key as a standby key and verify its public representation locally.
2. Publish both old and new public keys through JWKS.
3. Mark the new key active for new token issuance.
4. Confirm new tokens use the new `kid` and old tokens still verify.
5. Retain the old public key for at least the maximum access-token TTL and configured clock-skew allowance.
6. Remove the old public key only after the overlap window and verify no old-key traffic remains.

## Abort and recovery

If new tokens cannot be verified, stop issuance, restore the previous active key, retain both public keys, and investigate key-provider or JWKS cache errors. Never overwrite a published key ID.

## Evidence

Record the key IDs, timestamps, verification checks, and operator without recording private key material.
