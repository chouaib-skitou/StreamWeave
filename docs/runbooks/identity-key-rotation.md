# Runbook: Identity Signing-Key Rotation

## Scope

Rotate Ed25519 signing keys without invalidating still-valid access tokens unexpectedly.

## Preconditions

- A new private key is provisioned through the approved secret-file or KMS provider.
- The new public key has a unique `kid`.
- The operator can inspect readiness, JWKS, issuance metrics, and verification errors.

## Procedure

1. Provision the new active private key at the configured secret-file/KMS boundary and export its public key as `<old-kid>.pub.pem` files in `IDENTITY_SIGNING_KEY_HISTORY_DIR` for every still-valid old key.
2. Deploy with the new `IDENTITY_SIGNING_KEY_ID`; the service publishes the active key and all `.pub.pem` history keys through JWKS.
3. Confirm new tokens use the new `kid`, old tokens still verify, and the JWKS response contains no private material.
4. Retain the old public key for at least the maximum access-token TTL and configured clock-skew allowance.
5. Remove the old `.pub.pem` file only after the overlap window and verify no old-key traffic remains. Never reuse a published `kid`.

## Abort and recovery

If new tokens cannot be verified, stop issuance, restore the previous active key, retain both public keys, and investigate key-provider or JWKS cache errors. Never overwrite a published key ID.

## Evidence

Record the key IDs, timestamps, verification checks, and operator without recording private key material.
