# STORY-026 — Finalize Orders Kafka contracts

## Outcome

Publish and consume authenticated, versioned command/event schemas with compatibility and DLQ rules.

## Acceptance criteria

- [ ] Every message has the approved envelope and safe data schema.
- [ ] Commands and facts use separate topics and ACLs.
- [ ] `order_id` is the partition key.
- [ ] Unknown versions/producers and invalid causation are quarantined.
- [ ] v1 compatibility is additive-only and tested.

## Test plan

Schema validation, compatibility, ACL, header propagation, and DLQ replay tests.

## API and event changes

Register the Orders facts and commands under `contracts/events/`, using separate topics, explicit producer ownership, and the common v1 envelopes. Notifications consumes Orders facts under its own group according to [`orders-notifications.md`](../design/orders-notifications.md); Orders does not own notification delivery commands.

## Data and failure behavior

No business database change beyond message metadata requirements. Reject unknown schema versions, invalid ACL identity, wrong partition keys, missing causation, and unsafe payloads before domain handling.

## Observability and security

Propagate W3C trace context and correlation metadata while excluding tokens and sensitive payloads. Enforce TLS, authenticated producers/consumers, topic ACLs, environment isolation, and additive-only v1 evolution.

## Dependencies and out of scope

Depends on the common envelope schemas, Kafka platform policy, and Inventory/Payments/Fulfillment contracts. Broker cluster operations and downstream schemas outside the Orders boundary are out of scope.
