# Orders Database Recovery Runbook

## Objectives

Target RPO is zero for committed local transactions and target RTO is 30 minutes, subject to the selected PostgreSQL provider.

## Procedure

1. Declare the incident and stop new mutation traffic if data integrity is uncertain.
2. Restore the latest verified backup plus WAL/PITR to a controlled instance.
3. Validate migrations, order/Saga/Inbox/Outbox/idempotency constraints, and row counts.
4. Reconcile Outbox publication, Inbox processing, downstream operation IDs, and active Sagas before reopening traffic.
5. Resume relay/consumers only after duplicate and compensation safeguards are confirmed.

## Safety

Do not run destructive down migrations or delete Inbox/Outbox/Saga records to make a restore appear clean. Redis can be rebuilt; PostgreSQL remains authoritative.
