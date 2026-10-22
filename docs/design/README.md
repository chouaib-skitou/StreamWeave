# Design

Store design specifications here: service interaction diagrams, API and event design notes, data model sketches, operational flows, and interface decisions. Product UX is out of scope for the first version; this directory still captures developer-facing interaction and system design artifacts.

Identity data, flow, and deployment specifications are [`identity-data-model.md`](identity-data-model.md), [`identity-flows.md`](identity-flows.md), and [`identity-deployment.md`](identity-deployment.md).

Gateway design specifications are the [route matrix](gateway-route-matrix.md), [interaction flows](gateway-flows.md), [authentication design](gateway-authentication.md), [resilience policy](gateway-resilience.md), [implementation contract](gateway-implementation-contract.md), [deployment design](gateway-deployment.md), [observability design](gateway-observability.md), and [security checklist](gateway-security.md).

Orders design specifications are the [domain model](orders-domain-model.md), [state machine](orders-state-machine.md), [Saga](orders-saga.md), [data model](orders-data-model.md), [pricing boundary](orders-pricing-boundary.md), [reconciliation contract](orders-reconciliation.md), [notification boundary](orders-notifications.md), [authentication context](orders-auth-context.md), [Kafka contract](orders-kafka.md), [idempotency policy](orders-idempotency.md), [consistency policy](orders-consistency.md), [security design](orders-security.md), [observability design](orders-observability.md), and [deployment design](orders-deployment.md).
