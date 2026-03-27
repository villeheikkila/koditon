package operationpolicy

import "testing"

func TestForMCPTool(t *testing.T) {
	policy, ok := ForMCPTool("maku_search_products")
	if !ok {
		t.Fatal("expected policy for maku_search_products")
	}
	if policy.RateLimit <= 0 {
		t.Fatalf("unexpected rate limit: %d", policy.RateLimit)
	}
	if policy.MaxAttempts < 1 {
		t.Fatalf("unexpected max attempts: %d", policy.MaxAttempts)
	}
}

func TestForAPIOperation(t *testing.T) {
	policy, ok := ForAPIOperation("check-in-create")
	if !ok {
		t.Fatal("expected policy for check-in-create")
	}
	if policy.Timeout <= 0 {
		t.Fatalf("unexpected timeout: %v", policy.Timeout)
	}

	if _, ok := ForAPIOperation("does-not-exist"); ok {
		t.Fatal("did not expect policy for unknown operation")
	}
}

func TestForAPIOperationCachePolicy(t *testing.T) {
	policy, ok := ForAPIOperation("category-get-all")
	if !ok {
		t.Fatal("expected policy for category-get-all")
	}
	if !policy.Cache.HTTPEnabled() {
		t.Fatal("expected HTTP cache policy to be enabled")
	}
	if policy.Cache.HTTP.Scope != CacheScopePrivate {
		t.Fatalf("unexpected cache scope: %q", policy.Cache.HTTP.Scope)
	}
	if policy.Cache.HTTP.MaxAge <= 0 {
		t.Fatalf("unexpected cache max age: %v", policy.Cache.HTTP.MaxAge)
	}
	if policy.Cache.ServerEnabled() {
		t.Fatal("did not expect server cache policy for category-get-all")
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
