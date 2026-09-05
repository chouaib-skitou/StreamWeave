# Orders Saga Recovery Runbook

## Trigger

An order remains in a non-terminal state beyond the SLO or enters `FAILED_REQUIRES_REVIEW`.

## Procedure

1. Capture order ID, Saga ID, current phase, aggregate version, correlation ID, and last sanitized failure.
2. Inspect Inbox/Outbox history and downstream operation IDs.
3. Determine whether the last remote operation is unknown, successful, failed, or never accepted.
4. Resume the existing step or issue the matching compensation operation ID; never create a fresh blind operation.
5. Verify the expected fact event and state transition, then close the incident with an audit record.

## Escalation

Payment/refund ambiguity and compensation failure require Finance/Platform operator review. No operator may force `CONFIRMED`, `COMPLETED`, or `CANCELLED` without the named repair policy.
