# StreamWeave Orders Service

Orders is the authoritative owner of the order aggregate and its customer-facing asynchronous lifecycle. The service accepts asynchronous order requests, persists the order and Saga boundary transactionally, publishes Outbox messages, consumes dependency facts through an Inbox, and exposes status and cancellation operations.

The service will expose an internal ClusterIP API on container port `8080` and local host port `8082`. Public traffic enters through Gateway on `http://localhost:8081`.

## Source of truth

- Product requirements: [`../../docs/prd/PRD-003-orders-service.md`](../../docs/prd/PRD-003-orders-service.md)
- Architecture: [`../../docs/architecture/orders-service.md`](../../docs/architecture/orders-service.md)
- API contract: [`../../contracts/openapi/orders.openapi.yaml`](../../contracts/openapi/orders.openapi.yaml)
- Event and command contracts: [`../../docs/design/orders-kafka.md`](../../docs/design/orders-kafka.md) and [`../../contracts/events/`](../../contracts/events/)
- Delivery stories: [`../../docs/stories/EPIC-004-orders-service.md`](../../docs/stories/EPIC-004-orders-service.md)

The runtime is split into three binaries: `orders` (HTTP API), `orders-relay` (Kafka Outbox publisher and Inbox consumer), and `orders-migrate` (controlled schema migration job). The API never publishes directly to Kafka and the relay never serves public HTTP traffic.

Production deployments run `orders-migrate` as a Helm pre-install/pre-upgrade hook before the API and relay roll out; the migration runner holds a PostgreSQL advisory lock. The API readiness probe checks PostgreSQL and the relay probes its own process. Outbox publishing is at-least-once, Inbox processing is idempotent, retries are durable, and exhausted dependency messages are sent to `commerce.dlq.v1`.
