.PHONY: identity-generate identity-lint identity-test identity-test-integration identity-test-security identity-build identity-image identity-local-env identity-compose-up identity-compose-down identity-kind-deploy identity-smoke gateway-lint gateway-test gateway-build gateway-image gateway-local-env gateway-compose-up gateway-compose-down gateway-kind-deploy local-env local-config local-up local-down local-logs

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

gateway-lint:
	gofmt -l apps/gateway
	go vet ./apps/gateway/...

gateway-test:
	go test ./apps/gateway/... -race
	go test ./apps/gateway/internal/application/gateway -coverprofile gateway-coverage.out
	cmd /c "go tool cover -func=gateway-coverage.out"
	powershell -NoProfile -ExecutionPolicy Bypass -Command "$$coverage = (cmd /c 'go tool cover -func=gateway-coverage.out' | Select-String '^total:' | ForEach-Object { [double](($$_).ToString().Split()[-1].TrimEnd('%')) }); if ($$coverage -lt 85) { throw \"Gateway coverage is $$coverage%\" }; Write-Output \"Gateway coverage: $$coverage%\""

gateway-build:
	go build -trimpath -ldflags="-s -w" -o gateway ./apps/gateway/cmd/api

gateway-image:
	docker build -f deploy/gateway/Dockerfile -t streamweave-gateway:local .

gateway-local-env:
	powershell -NoProfile -Command "if (-not (Test-Path -LiteralPath 'deploy/gateway/local/.env')) { Copy-Item -LiteralPath 'deploy/gateway/local/.env.example' -Destination 'deploy/gateway/local/.env'; Write-Output 'Created deploy/gateway/local/.env from .env.example (local-only, ignored by Git).' }"

gateway-compose-up: gateway-local-env
	docker compose --env-file deploy/gateway/local/.env -f deploy/gateway/local/compose.yaml up -d --build

gateway-compose-down: gateway-local-env
	docker compose --env-file deploy/gateway/local/.env -f deploy/gateway/local/compose.yaml down -v

gateway-kind-deploy:
	helm upgrade --install gateway deploy/gateway/helm --namespace ecommerce-gateway --create-namespace

local-env:
	powershell -NoProfile -Command "if (-not (Test-Path -LiteralPath 'monitoring/local/.env')) { Copy-Item -LiteralPath 'monitoring/local/.env.example' -Destination 'monitoring/local/.env'; Write-Output 'Created monitoring/local/.env from .env.example (local-only, ignored by Git).' }"

local-config: local-env
	docker compose --env-file monitoring/local/.env -f deploy/local/compose.yaml -f monitoring/local/compose.yaml config --quiet

local-up: local-env
	docker compose --env-file monitoring/local/.env -f deploy/local/compose.yaml -f monitoring/local/compose.yaml up -d --build --remove-orphans

local-down: local-env
	docker compose --env-file monitoring/local/.env -f deploy/local/compose.yaml -f monitoring/local/compose.yaml down

local-logs: local-env
	docker compose --env-file monitoring/local/.env -f deploy/local/compose.yaml -f monitoring/local/compose.yaml logs -f
