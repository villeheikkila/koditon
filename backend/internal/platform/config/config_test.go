package config

import "testing"

const testPEM = "-----BEGIN PRIVATE KEY-----\nAA==\n-----END PRIVATE KEY-----"

func TestFromEnvMapAPIValid(t *testing.T) {
	t.Parallel()
	_, err := FromEnvMap(baseEnv(map[string]string{
		"APP_MODE":                  "api",
		"AUTH_JWT_SIGNING_KEY":      testPEM,
		"AUTH_JWT_ISSUER":           "koditon-test",
		"AUTH_APPLE_BUNDLE_ID":      "com.example.koditon",
		"AUTH_APPLE_TEAM_ID":        "TEAMID",
		"AUTH_APPLE_PRIVATE_KEY_ID": "KEYID",
		"AUTH_APPLE_PRIVATE_KEY":    testPEM,
	}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
}

func TestFromEnvMapConsumerRequiresProviderSettings(t *testing.T) {
	t.Parallel()
	_, err := FromEnvMap(baseEnv(map[string]string{
		"APP_MODE": "consumer",
	}))
	if err == nil {
		t.Fatal("expected missing consumer provider settings to fail")
	}
}

func TestFromEnvMapRejectsProductionLocalhostPasskey(t *testing.T) {
	t.Parallel()
	_, err := FromEnvMap(baseEnv(map[string]string{
		"APP_ENV":                   "production",
		"APP_MODE":                  "api",
		"AUTH_JWT_SIGNING_KEY":      testPEM,
		"AUTH_JWT_ISSUER":           "koditon-test",
		"AUTH_APPLE_BUNDLE_ID":      "com.example.koditon",
		"AUTH_APPLE_TEAM_ID":        "TEAMID",
		"AUTH_APPLE_PRIVATE_KEY_ID": "KEYID",
		"AUTH_APPLE_PRIVATE_KEY":    testPEM,
		"AUTH_PASSKEY_RP_ID":        "localhost",
		"AUTH_PASSKEY_RP_ORIGINS":   "http://localhost:3000",
	}))
	if err == nil {
		t.Fatal("expected production localhost passkey config to fail")
	}
}

func TestFromEnvMapDatabasePoolDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := FromEnvMap(baseEnv(map[string]string{
		"APP_MODE":                  "api",
		"AUTH_JWT_SIGNING_KEY":      testPEM,
		"AUTH_JWT_ISSUER":           "koditon-test",
		"AUTH_APPLE_BUNDLE_ID":      "com.example.koditon",
		"AUTH_APPLE_TEAM_ID":        "TEAMID",
		"AUTH_APPLE_PRIVATE_KEY_ID": "KEYID",
		"AUTH_APPLE_PRIVATE_KEY":    testPEM,
	}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
	if cfg.Database.MaxConns != 10 {
		t.Fatalf("MaxConns = %d, want 10", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 2 {
		t.Fatalf("MinConns = %d, want 2", cfg.Database.MinConns)
	}
}

func TestFromEnvMapRejectsInvalidDatabasePoolConfig(t *testing.T) {
	t.Parallel()
	_, err := FromEnvMap(baseEnv(map[string]string{
		"APP_MODE":                  "api",
		"AUTH_JWT_SIGNING_KEY":      testPEM,
		"AUTH_JWT_ISSUER":           "koditon-test",
		"AUTH_APPLE_BUNDLE_ID":      "com.example.koditon",
		"AUTH_APPLE_TEAM_ID":        "TEAMID",
		"AUTH_APPLE_PRIVATE_KEY_ID": "KEYID",
		"AUTH_APPLE_PRIVATE_KEY":    testPEM,
		"DB_MAX_CONNS":              "1",
		"DB_MIN_CONNS":              "2",
	}))
	if err == nil {
		t.Fatal("expected invalid database pool config to fail")
	}
}

func TestFromEnvMapTelemetryDisabledByDefault(t *testing.T) {
	t.Parallel()
	cfg, err := FromEnvMap(validAPIEnv(map[string]string{}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
	if cfg.Telemetry != nil {
		t.Fatalf("Telemetry = %#v, want nil", cfg.Telemetry)
	}
}

func TestFromEnvMapTelemetryEnabled(t *testing.T) {
	t.Parallel()
	cfg, err := FromEnvMap(validAPIEnv(map[string]string{
		"OTEL_ENABLED":                "true",
		"OTEL_SERVICE_NAME":           "koditon-test",
		"OTEL_SERVICE_VERSION":        "test-version",
		"OTEL_SERVICE_INSTANCE_ID":    "test-instance",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
		"OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
		"OTEL_EXPORTER_OTLP_HEADERS":  "Authorization=Bearer dev-token,X-Test=yes",
		"OTEL_TRACES_SAMPLER":         "parentbased_traceidratio",
		"OTEL_TRACES_SAMPLER_ARG":     "0.25",
	}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
	if cfg.Telemetry == nil {
		t.Fatal("Telemetry = nil, want config")
	}
	if cfg.Telemetry.ServiceName != "koditon-test" {
		t.Fatalf("ServiceName = %q", cfg.Telemetry.ServiceName)
	}
	if cfg.Telemetry.ServiceVersion != "test-version" {
		t.Fatalf("ServiceVersion = %q", cfg.Telemetry.ServiceVersion)
	}
	if cfg.Telemetry.ServiceInstanceID != "test-instance" {
		t.Fatalf("ServiceInstanceID = %q", cfg.Telemetry.ServiceInstanceID)
	}
	if cfg.Telemetry.OTLPEndpoint != "http://localhost:4318" {
		t.Fatalf("OTLPEndpoint = %q", cfg.Telemetry.OTLPEndpoint)
	}
	if cfg.Telemetry.OTLPHeaders["Authorization"] != "Bearer dev-token" || cfg.Telemetry.OTLPHeaders["X-Test"] != "yes" {
		t.Fatalf("OTLPHeaders = %#v", cfg.Telemetry.OTLPHeaders)
	}
	if cfg.Telemetry.SamplerArg != "0.25" {
		t.Fatalf("SamplerArg = %q", cfg.Telemetry.SamplerArg)
	}
}

func TestFromEnvMapTelemetryLegacyEndpointFallback(t *testing.T) {
	t.Parallel()
	cfg, err := FromEnvMap(validAPIEnv(map[string]string{
		"OTEL_OTLP_ENDPOINT": "otel-collector:4317",
		"OTEL_OTLP_PROTOCOL": "grpc",
		"OTEL_OTLP_INSECURE": "true",
	}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
	if cfg.Telemetry == nil {
		t.Fatal("Telemetry = nil, want config")
	}
	if cfg.Telemetry.OTLPEndpoint != "otel-collector:4317" {
		t.Fatalf("OTLPEndpoint = %q", cfg.Telemetry.OTLPEndpoint)
	}
	if cfg.Telemetry.OTLPProtocol != "grpc" {
		t.Fatalf("OTLPProtocol = %q", cfg.Telemetry.OTLPProtocol)
	}
	if !cfg.Telemetry.OTLPInsecure {
		t.Fatal("OTLPInsecure = false, want true")
	}
}

func TestFromEnvMapTelemetrySDKDisabledWins(t *testing.T) {
	t.Parallel()
	cfg, err := FromEnvMap(validAPIEnv(map[string]string{
		"OTEL_ENABLED":      "true",
		"OTEL_SDK_DISABLED": "true",
	}))
	if err != nil {
		t.Fatalf("FromEnvMap returned error: %v", err)
	}
	if cfg.Telemetry != nil {
		t.Fatalf("Telemetry = %#v, want nil", cfg.Telemetry)
	}
}

func TestFromEnvMapRejectsInvalidTelemetryRatio(t *testing.T) {
	t.Parallel()
	_, err := FromEnvMap(validAPIEnv(map[string]string{
		"OTEL_ENABLED":      "true",
		"OTEL_SAMPLE_RATIO": "1.5",
	}))
	if err == nil {
		t.Fatal("expected invalid telemetry ratio to fail")
	}
}

func validAPIEnv(overrides map[string]string) map[string]string {
	values := baseEnv(map[string]string{
		"APP_MODE":                  "api",
		"AUTH_JWT_SIGNING_KEY":      testPEM,
		"AUTH_JWT_ISSUER":           "koditon-test",
		"AUTH_APPLE_BUNDLE_ID":      "com.example.koditon",
		"AUTH_APPLE_TEAM_ID":        "TEAMID",
		"AUTH_APPLE_PRIVATE_KEY_ID": "KEYID",
		"AUTH_APPLE_PRIVATE_KEY":    testPEM,
	})
	for key, value := range overrides {
		values[key] = value
	}
	return values
}

func baseEnv(overrides map[string]string) map[string]string {
	values := map[string]string{
		"APP_HOST":             "127.0.0.1",
		"APP_PORT":             "8080",
		"APP_SHUTDOWN_TIMEOUT": "10s",
		"APP_ENV":              "development",
		"APP_MODE":             "api",
		"LOG_LEVEL":            "info",
		"DATABASE_URL":         "postgres://postgres:postgres@localhost:5432/koditon?sslmode=disable",
	}
	for key, value := range overrides {
		values[key] = value
	}
	return values
}
