# StreamWeave Orders Service

Orders is the authoritative owner of the order aggregate and its customer-facing asynchronous lifecycle. The service accepts asynchronous order requests, persists the order and Saga boundary transactionally, publishes Outbox messages, consumes dependency facts through an Inbox, and exposes status and cancellation operations.

The service exposes an internal ClusterIP API on container port `8080` and local host port `8082`. Public traffic enters through Gateway on `http://localhost:8081`.

For a complete local manual flow, import [`../../contracts/postman/orders-service.postman_collection.json`](../../contracts/postman/orders-service.postman_collection.json)
and [`../../contracts/postman/orders.local.postman_environment.json`](../../contracts/postman/orders.local.postman_environment.json).
Run the collection's `Local setup` folder first; it creates a local demo customer,
reads the verification email from MailHog, logs in through Gateway, and stores the
tokens used by the order requests.

Local Swagger URLs:

- Gateway: `http://localhost:8081/v1/docs`
- Orders: `http://localhost:8082/v1/docs`
- Identity: `http://localhost:8080/v1/docs`

## Source of truth

- Product requirements: [`../../docs/prd/PRD-003-orders-service.md`](../../docs/prd/PRD-003-orders-service.md)
- Architecture: [`../../docs/architecture/orders-service.md`](../../docs/architecture/orders-service.md)
- API contract: [`../../contracts/openapi/orders.openapi.yaml`](../../contracts/openapi/orders.openapi.yaml)
- Event and command contracts: [`../../docs/design/orders-kafka.md`](../../docs/design/orders-kafka.md) and [`../../contracts/events/`](../../contracts/events/)
- Delivery stories: [`../../docs/stories/EPIC-004-orders-service.md`](../../docs/stories/EPIC-004-orders-service.md)

The runtime is split into three binaries: `orders` (HTTP API), `orders-relay` (Kafka Outbox publisher and Inbox consumer), and `orders-migrate` (controlled schema migration job). The API never publishes directly to Kafka and the relay never serves public HTTP traffic.

Production deployments run `orders-migrate` as a Helm pre-install/pre-upgrade hook before the API and relay roll out; the migration runner holds a PostgreSQL advisory lock. The API readiness probe checks PostgreSQL and the relay probes its own process. Outbox publishing is at-least-once, Inbox processing is idempotent, retries are durable, and exhausted dependency messages are sent to `commerce.dlq.v1`.
