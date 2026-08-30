.PHONY: identity-generate identity-lint identity-test identity-test-integration identity-test-security identity-build identity-image identity-local-env identity-compose-up identity-compose-down identity-kind-deploy identity-smoke

identity-generate:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate

identity-lint:
	gofmt -l .
	go vet ./...

identity-test:
	go test ./apps/identity/...
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-identity-coverage.ps1

identity-test-integration:
	go test -tags=integration ./apps/identity/tests/integration/...

identity-test-security:
	go test -tags=security ./apps/identity/tests/security/...

identity-build:
	go build -trimpath -ldflags="-s -w" -o identity ./apps/identity/cmd/api

identity-image:
	docker build -f deploy/identity/Dockerfile -t streamweave-identity:local .

identity-local-env:
	powershell -NoProfile -Command "if (-not (Test-Path -LiteralPath 'deploy/identity/local/.env')) { Copy-Item -LiteralPath 'deploy/identity/local/.env.example' -Destination 'deploy/identity/local/.env'; Write-Output 'Created deploy/identity/local/.env from .env.example (local-only, ignored by Git). Update passwords if needed.' }"

identity-compose-up: identity-local-env
	docker compose --env-file deploy/identity/local/.env -f deploy/identity/local/compose.yaml up -d --build

identity-compose-down: identity-local-env
	docker compose --env-file deploy/identity/local/.env -f deploy/identity/local/compose.yaml down -v

identity-kind-deploy:
	helm upgrade --install identity deploy/identity/helm --namespace identity --create-namespace

identity-smoke:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/identity-smoke.ps1
