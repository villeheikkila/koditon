package operationpolicy

import "testing"

func TestForMCPTool(t *testing.T) {
	policy, ok := ForMCPTool("koditon_search_products")
	if !ok {
		t.Fatal("expected policy for koditon_search_products")
	}
	if policy.RateLimit <= 0 {
		t.Fatalf("unexpected rate limit: %d", policy.RateLimit)
	}
	if policy.MaxAttempts < 1 {
		t.Fatalf("unexpected max attempts: %d", policy.MaxAttempts)
	}
}

func TestForAPIOperation(t *testing.T) {
	policy, ok := ForAPIOperation("oauth-token-create")
	if !ok {
		t.Fatal("expected policy for oauth-token-create")
	}
	if policy.Timeout <= 0 {
		t.Fatalf("unexpected timeout: %v", policy.Timeout)
	}

	if _, ok := ForAPIOperation("does-not-exist"); ok {
		t.Fatal("did not expect policy for unknown operation")
	}
}

func TestForAPIOperationServerCachePolicy(t *testing.T) {
	policy, ok := ForAPIOperation("config")
	if !ok {
		t.Fatal("expected policy for config")
	}
	if !policy.Cache.ServerEnabled() {
		t.Fatal("expected server cache policy to be enabled")
	}
	if policy.Cache.Server.Store != ServerCacheStoreRedis {
		t.Fatalf("unexpected server cache store: %q", policy.Cache.Server.Store)
	}
	if policy.Cache.Server.TTL <= 0 {
		t.Fatalf("unexpected server cache ttl: %v", policy.Cache.Server.TTL)
	}
}
