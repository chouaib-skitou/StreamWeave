package config

import "testing"

type source map[string]string

func (s source) Lookup(key string) (string, bool) { v, ok := s[key]; return v, ok }

func TestLoadDevelopmentDefaults(t *testing.T) {
	cfg, err := Load(source{})
	if err != nil || cfg.HTTPAddr != ":8080" || cfg.MetricsPath != "/metrics" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestProductionRequiresSecureDependencies(t *testing.T) {
	valid := source{"ORDERS_ENVIRONMENT": "production", "ORDERS_DATABASE_URL": "postgres://orders@db/orders?sslmode=verify-full", "ORDERS_KAFKA_BROKERS": "kafka:9093", "ORDERS_OTEL_ENDPOINT": "otel:4317", "ORDERS_GATEWAY_JWKS_URL": "https://identity.example.test/.well-known/jwks.json", "ORDERS_INVENTORY_JWKS_URL": "https://inventory.example.test/.well-known/jwks.json", "ORDERS_CURSOR_SECRET": "01234567890123456789012345678901"}
	if _, err := Load(valid); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"ORDERS_DATABASE_URL": "postgres://orders@db/orders?sslmode=disable", "ORDERS_OTEL_INSECURE": "true"} {
		candidate := source{}
		for k, v := range valid {
			candidate[k] = v
		}
		candidate[key] = value
		if _, err := Load(candidate); err == nil {
			t.Fatalf("accepted insecure %s", key)
		}
	}
}

func TestProductionApiDoesNotReceiveKafkaConfiguration(t *testing.T) {
	values := source{"ORDERS_ENVIRONMENT": "production", "ORDERS_DATABASE_URL": "postgres://orders@db/orders?sslmode=verify-full", "ORDERS_OTEL_ENDPOINT": "otel:4317", "ORDERS_GATEWAY_JWKS_URL": "https://identity.example.test/.well-known/jwks.json", "ORDERS_INVENTORY_JWKS_URL": "https://inventory.example.test/.well-known/jwks.json", "ORDERS_CURSOR_SECRET": "01234567890123456789012345678901"}
	if _, err := Load(values); err != nil {
		t.Fatal(err)
	}
	values["ORDERS_ROLE"] = "relay"
	if _, err := Load(values); err == nil {
		t.Fatal("accepted a production relay without Kafka configuration")
	}
}

func TestRejectsMalformedValues(t *testing.T) {
	for _, values := range []source{{"ORDERS_ENVIRONMENT": "invalid"}, {"ORDERS_METRICS_PATH": "/internal"}, {"ORDERS_READINESS_TIMEOUT": "bad"}, {"ORDERS_OTEL_INSECURE": "maybe"}} {
		if _, err := Load(values); err == nil {
			t.Fatalf("accepted %v", values)
		}
	}
}
