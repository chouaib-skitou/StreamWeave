# StreamWeave Orders Service

Orders is the authoritative owner of the order aggregate and its customer-facing asynchronous lifecycle. The initial runtime foundation is implemented; business workflows follow as separately verified slices from the approved Orders PRD, architecture, contracts, ADRs, stories, and runbooks.

The service will expose an internal ClusterIP API on container port `8080` and local host port `8082`. Public traffic enters through Gateway on `http://localhost:8081`.

## Source of truth

- Product requirements: [`../../docs/prd/PRD-003-orders-service.md`](../../docs/prd/PRD-003-orders-service.md)
- Architecture: [`../../docs/architecture/orders-service.md`](../../docs/architecture/orders-service.md)
- API contract: [`../../contracts/openapi/orders.openapi.yaml`](../../contracts/openapi/orders.openapi.yaml)
- Event and command contracts: [`../../docs/design/orders-kafka.md`](../../docs/design/orders-kafka.md) and [`../../contracts/events/`](../../contracts/events/)
- Delivery stories: [`../../docs/stories/EPIC-004-orders-service.md`](../../docs/stories/EPIC-004-orders-service.md)

The documentation gate is accepted. Each subsequent workflow slice must preserve the boundaries above and update its contract before implementation when behavior changes.
