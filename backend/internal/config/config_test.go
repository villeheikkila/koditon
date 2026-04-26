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
