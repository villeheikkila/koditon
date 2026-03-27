package telemetry

import "go.opentelemetry.io/otel/attribute"

type attrBuilder struct{}

var Attrs attrBuilder

func (attrBuilder) AuthProvider(value string) attribute.KeyValue {
	return attribute.String("auth.provider", value)
}

func (attrBuilder) AuthHasDeviceID(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_device_id", value)
}

func (attrBuilder) AuthHasUserAgent(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_user_agent", value)
}

func (attrBuilder) AuthHasIP(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_ip", value)
}

func (attrBuilder) AuthHasAuthCode(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_auth_code", value)
}

func (attrBuilder) AuthHasNonce(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_nonce", value)
}

func (attrBuilder) AuthHasToken(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_token", value)
}

func (attrBuilder) AuthHasRefreshToken(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_refresh_token", value)
}

func (attrBuilder) AuthHasAppleRefresh(value bool) attribute.KeyValue {
	return attribute.Bool("auth.has_apple_refresh", value)
}

func (attrBuilder) AuthType(value string) attribute.KeyValue {
	return attribute.String("auth.type", value)
}

func (attrBuilder) AuthScopesRequired(value []string) attribute.KeyValue {
	return attribute.StringSlice("auth.scopes.required", value)
}

func (attrBuilder) AuthSignOutScope(value string) attribute.KeyValue {
	return attribute.String("auth.sign_out_scope", value)
}

func (attrBuilder) AuthIsNewUser(value bool) attribute.KeyValue {
	return attribute.Bool("auth.is_new_user", value)
}

func (attrBuilder) StoragePrefix(value string) attribute.KeyValue {
	return attribute.String("storage.prefix", value)
}

func (attrBuilder) StorageMimeType(value string) attribute.KeyValue {
	return attribute.String("storage.mime_type", value)
}

func (attrBuilder) StorageHasSize(value bool) attribute.KeyValue {
	return attribute.Bool("storage.has_size", value)
}

func (attrBuilder) StorageHasOwnerID(value bool) attribute.KeyValue {
	return attribute.Bool("storage.has_owner_id", value)
}

func (attrBuilder) StorageHasObjectID(value bool) attribute.KeyValue {
	return attribute.Bool("storage.has_object_id", value)
}

func (attrBuilder) StorageHasObjectKey(value bool) attribute.KeyValue {
	return attribute.Bool("storage.has_object_key", value)
}

func (attrBuilder) OperationID(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_id", value)
}

func (attrBuilder) OperationSummary(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_summary", value)
}

func (attrBuilder) OperationPath(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_path", value)
}

func (attrBuilder) OperationMethod(value string) attribute.KeyValue {
	return attribute.String("koditon.operation_method", value)
}

func (attrBuilder) RateLimitAllowed(value bool) attribute.KeyValue {
	return attribute.Bool("rate_limit.allowed", value)
}

func (attrBuilder) RateLimitLimit(value int64) attribute.KeyValue {
	return attribute.Int64("rate_limit.limit", value)
}

func (attrBuilder) RateLimitRemaining(value float64) attribute.KeyValue {
	return attribute.Float64("rate_limit.remaining", value)
}

func (attrBuilder) RateLimitRefillPerSecond(value float64) attribute.KeyValue {
	return attribute.Float64("rate_limit.refill_per_second", value)
}

func (attrBuilder) RateLimitIdentityType(value string) attribute.KeyValue {
	return attribute.String("rate_limit.identity_type", value)
}

func (attrBuilder) RateLimitPerOperation(value bool) attribute.KeyValue {
	return attribute.Bool("rate_limit.per_operation", value)
}
