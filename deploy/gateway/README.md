# Gateway deployment

The Gateway is the public HTTP edge for StreamWeave. It listens on container port `8080`; local Compose exposes it as `http://localhost:8081` because Identity already uses host port `8080`.

## Local

Start Identity first, then copy `local/.env.example` to `local/.env`. Set a short-lived development `GATEWAY_SERVICE_TOKEN` issued for the `svc_gateway` principal and run:

```bash
make gateway-compose-up
```

The local observability endpoints are:

- Gateway: `http://localhost:8081`
- Swagger UI: `http://localhost:8081/v1/docs`
- Prometheus: `http://localhost:9091`
- Grafana: `http://localhost:3002`

Mail delivery remains an Identity concern and is provided by Identity's MailHog stack.

## Staging and production

Create the external Secret referenced by `secret.existingSecret` with at least:

- `GATEWAY_SERVICE_CLIENT_SECRET`
- `GATEWAY_REDIS_URL` (use `rediss://` in staging and production)

The chart also reads the non-sensitive `GATEWAY_SERVICE_CLIENT_ID` and scope list from its ConfigMap. Use a secret manager to inject the client secret, Redis URL, and any private endpoint credentials. Static service tokens are rejected outside development. Production values require HTTPS for Identity/Orders/JWKS, `rediss://` for Redis, and explicit NetworkPolicy destinations.

```bash
helm upgrade --install gateway deploy/gateway/helm \
  --namespace ecommerce-gateway --create-namespace \
  -f deploy/gateway/staging/values.yaml
```

Use the production values file for production and replace the example endpoints before deployment. The image is published by the main-branch release workflow as `ghcr.io/chouaib-skitou/streamweave-gateway:v<version>` with SBOM and provenance.
