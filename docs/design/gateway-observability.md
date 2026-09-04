# Gateway observability design

## Structured logs

Logs are JSON and include `timestamp`, `level`, `service`, `message`, `route`, `method`, `status`, `request_id`, `trace_id`, `span_id`, `correlation_id`, `duration_ms`, and a bounded `error_code` when applicable. They may include an upstream logical service name, never an internal hostname.

The following are prohibited: `Authorization`, access/refresh tokens, passwords, cookies, raw request/response bodies, full query strings, email addresses, cardholder data, and unbounded user-controlled values.

## Metrics

Use low-cardinality labels only:

- `gateway_http_requests_total{route,method,status}`
- `gateway_http_request_duration_seconds{route,method}`
- `gateway_auth_failures_total{route,reason_class}`
- `gateway_rate_limit_rejections_total{route,limit_class}`
- `gateway_rate_limiter_errors_total{limit_class}`
- `gateway_jwks_refresh_total{outcome}`
- `gateway_upstream_requests_total{service,route,outcome}`
- `gateway_upstream_duration_seconds{service,route}`
- `gateway_in_flight_requests`

Never use request IDs, user IDs, order IDs, tokens, or raw URLs as metric labels.

## Traces

Gateway creates a server span and bounded internal client spans. W3C trace context is propagated. Span attributes use route templates, method, status, service, and error class. Sensitive headers and bodies are excluded by default.

## Alerts

The first production alert set should cover sustained 5xx/503, p95 latency, rate-limit rejection spikes, JWKS refresh failures, limiter errors, upstream timeout ratio, and readiness loss. Alert rules belong with the deployment/monitoring stack and must use the metric names above; this document does not invent a second telemetry vocabulary.

## Dashboards

The Gateway dashboard should show request rate/error rate/latency, auth failures, 429s, JWKS health, Redis limiter health, upstream dependency outcomes, in-flight load, and readiness. Panels group by route template and logical service only.
