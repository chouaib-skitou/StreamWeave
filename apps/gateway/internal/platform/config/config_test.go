package config

import "testing"

type mapSource map[string]string

func (m mapSource) Lookup(key string) (string, bool) { value, ok := m[key]; return value, ok }

func TestLoadDefaultsAndProductionInvariants(t *testing.T) {
	cfg, err := Load(mapSource{})
	if err != nil || cfg.HTTPAddr != ":8080" || cfg.HumanAudience != "platform-api" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	production := mapSource{"GATEWAY_ENVIRONMENT": "production", "GATEWAY_SERVICE_CLIENT_ID": "gateway", "GATEWAY_SERVICE_CLIENT_SECRET": "secret", "GATEWAY_IDENTITY_URL": "https://identity.example", "GATEWAY_ORDERS_URL": "https://orders.example", "GATEWAY_JWKS_URL": "https://identity.example/.well-known/jwks.json", "GATEWAY_REDIS_URL": "rediss://redis.example:6380/1"}
	if _, err := Load(production); err != nil {
		t.Fatalf("production rejected: %v", err)
	}
	production["GATEWAY_SERVICE_TOKEN"] = "static"
	if _, err := Load(production); err == nil {
		t.Fatal("static token accepted in production")
	}
	staging := mapSource{"GATEWAY_ENVIRONMENT": "staging", "GATEWAY_STATIC_SERVICE_TOKEN": "static"}
	if _, err := Load(staging); err == nil {
		t.Fatal("staging without service credentials accepted")
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	for _, values := range []mapSource{{"GATEWAY_ENVIRONMENT": "invalid"}, {"GATEWAY_MAX_RESPONSE_BYTES": "0"}, {"GATEWAY_CORS_ORIGINS": "*"}, {"GATEWAY_TRUST_PROXY_HEADERS": "maybe"}} {
		if _, err := Load(values); err == nil {
			t.Fatalf("invalid values accepted: %v", values)
		}
	}
}
