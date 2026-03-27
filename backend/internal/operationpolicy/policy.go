package operationpolicy

import (
	"strings"
	"time"
)

// Policy describes operation-level resilience and abuse-control behavior.
// It is transport-agnostic and can be consumed by HTTP and MCP layers.
type Policy struct {
	Timeout              time.Duration
	MaxAttempts          int
	RetryBaseBackoff     time.Duration
	RetryableStatuses    []int
	RetryJitter          bool
	RetryTransportErrors bool
	RateLimit            int
	RateWindow           time.Duration
	Mutation             bool
	IdempotencyRequired  bool
	IdempotencyTTL       time.Duration
	Cache                CachePolicy
}

type CacheScope string

const (
	CacheScopeNone    CacheScope = ""
	CacheScopePrivate CacheScope = "private"
	CacheScopePublic  CacheScope = "public"
)

type CachePolicy struct {
	HTTP   HTTPCachePolicy
	Server ServerCachePolicy
}

type HTTPCachePolicy struct {
	Scope                CacheScope
	MaxAge               time.Duration
	StaleWhileRevalidate time.Duration
	StaleIfError         time.Duration
	MustRevalidate       bool
	UseETag              bool
	UseLastModified      bool
}

type ServerCacheStore string

const (
	ServerCacheStoreNone  ServerCacheStore = ""
	ServerCacheStoreRedis ServerCacheStore = "redis"
)

type ServerCachePolicy struct {
	Store              ServerCacheStore
	TTL                time.Duration
	VaryByAuthIdentity bool
	VaryHeaders        []string
}

func (p CachePolicy) HTTPEnabled() bool {
	return p.HTTP.Enabled()
}

func (p CachePolicy) ServerEnabled() bool {
	return p.Server.Enabled()
}

func (p HTTPCachePolicy) Enabled() bool {
	return p.Scope != CacheScopeNone && p.MaxAge > 0
}

func (p ServerCachePolicy) Enabled() bool {
	return p.Store != ServerCacheStoreNone && p.TTL > 0
}

var (
	idempotencyTTL   = 24 * time.Hour
	catalogHTTPCache = HTTPCachePolicy{
		Scope:                CacheScopePrivate,
		MaxAge:               5 * time.Minute,
		StaleWhileRevalidate: time.Minute,
		StaleIfError:         24 * time.Hour,
		UseETag:              true,
		UseLastModified:      true,
	}
	configServerCache = ServerCachePolicy{
		Store: ServerCacheStoreRedis,
		TTL:   5 * time.Minute,
	}

	readSearchPolicy = Policy{
		Timeout:          4 * time.Second,
		MaxAttempts:      2,
		RetryBaseBackoff: 120 * time.Millisecond,
		RetryableStatuses: []int{
			429,
			502,
			503,
			504,
		},
		RetryJitter:          true,
		RetryTransportErrors: true,
		RateLimit:            120,
		RateWindow:           time.Minute,
		Mutation:             false,
		IdempotencyTTL:       idempotencyTTL,
	}
	checkInWritePolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		RetryTransportErrors: true,
		RateLimit:            24,
		RateWindow:           time.Minute,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
	}
	checkInUploadRequestPolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		RetryTransportErrors: true,
		RateLimit:            24,
		RateWindow:           time.Minute,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
	}
	checkInUploadConfirmPolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		RetryTransportErrors: true,
		RateLimit:            48,
		RateWindow:           time.Minute,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
	}
	defaultMutationPolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		RetryJitter:          true,
		RetryTransportErrors: true,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
		RateLimit:            0,
		RateWindow:           0,
	}
	defaultReadLikePostPolicy = Policy{
		Timeout:          4 * time.Second,
		MaxAttempts:      2,
		RetryBaseBackoff: 120 * time.Millisecond,
		RetryableStatuses: []int{
			429,
			502,
			503,
			504,
		},
		RetryJitter:          true,
		RetryTransportErrors: true,
		Mutation:             false,
		IdempotencyTTL:       idempotencyTTL,
		RateLimit:            0,
		RateWindow:           0,
	}
	defaultPutPolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      2,
		RetryBaseBackoff: 120 * time.Millisecond,
		RetryableStatuses: []int{
			429,
			502,
			503,
			504,
		},
		RetryJitter:          true,
		RetryTransportErrors: true,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
	}
	defaultDeletePolicy = Policy{
		Timeout:          8 * time.Second,
		MaxAttempts:      2,
		RetryBaseBackoff: 120 * time.Millisecond,
		RetryableStatuses: []int{
			429,
			502,
			503,
			504,
		},
		RetryJitter:          true,
		RetryTransportErrors: true,
		Mutation:             true,
		IdempotencyTTL:       idempotencyTTL,
	}
)

var mcpToolPolicies = map[string]Policy{
	"koditon_search_products":               readSearchPolicy,
	"koditon_create_check_in":               requireIdempotency(checkInWritePolicy),
	"koditon_request_check_in_image_upload": checkInUploadRequestPolicy,
	"koditon_confirm_check_in_image_upload": requireIdempotency(checkInUploadConfirmPolicy),
}

var apiOperationPolicies = map[string]Policy{
	"config":                     withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache, Server: configServerCache}),
	"badge-get-all":              withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"category-get-all":           withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"category-get-by-id":         withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"check-in-create":            requireIdempotency(checkInWritePolicy),
	"flavor-get-all":             withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"flavor-get-by-id":           withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"serving-style-get-all":      withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"serving-style-get-by-id":    withCache(defaultReadLikePostPolicy, CachePolicy{HTTP: catalogHTTPCache}),
	"storage-request-upload-url": requireIdempotency(checkInUploadRequestPolicy),
	"storage-confirm-upload":     requireIdempotency(checkInUploadConfirmPolicy),
	"analytics-ingest": {
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		Mutation:             true,
		RetryTransportErrors: false,
		IdempotencyTTL:       idempotencyTTL,
	},
	"oauth-device-authorization-create": {
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		Mutation:             true,
		RetryTransportErrors: false,
		IdempotencyTTL:       idempotencyTTL,
	},
	"oauth-token-create": {
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		Mutation:             true,
		RetryTransportErrors: false,
		IdempotencyTTL:       idempotencyTTL,
	},
	"oauth-client-register-create": {
		Timeout:          8 * time.Second,
		MaxAttempts:      1,
		RetryBaseBackoff: 0,
		RetryableStatuses: []int{
			429,
			503,
		},
		Mutation:             true,
		RetryTransportErrors: false,
		IdempotencyTTL:       idempotencyTTL,
	},
	"oauth-revoke-create": defaultDeletePolicy,
}

// Explicit POST/PATCH operation IDs in backend/internal/api.
// Mutations require idempotency by default; read-like POST endpoints are excluded.
var readLikePostOperationIDs = map[string]struct{}{
	"auth-email-start":                  {},
	"auth-passkey-authenticate-options": {},
	"auth-passkey-register-options":     {},
	"product-bulk-get":                  {},
	"product-searchable-bulk-get":       {},
}

var mutationOperationIDs = map[string]struct{}{
	"admin-event-mark-reviewed":              {},
	"auth-email-confirm":                     {},
	"auth-email-request":                     {},
	"auth-passkey-register-finish":           {},
	"auth-sign-out":                          {},
	"auth-sign-out-all":                      {},
	"auth-session-delete":                    {},
	"brand-add-logo":                         {},
	"brand-create":                           {},
	"brand-edit-suggestion-create":           {},
	"brand-like":                             {},
	"category-add-serving-style":             {},
	"category-create":                        {},
	"check-in-comment-create":                {},
	"check-in-create":                        {},
	"check-in-image-create":                  {},
	"check-in-reaction-create":               {},
	"check-in-update":                        {},
	"company-add-logo":                       {},
	"company-create":                         {},
	"company-edit-suggestion-create":         {},
	"company-make-subsidiary":                {},
	"company-merge":                          {},
	"edit-suggestion-resolve":                {},
	"email-confirm-change":                   {},
	"flavor-create":                          {},
	"friend-create":                          {},
	"friend-update":                          {},
	"image-entity-update-metadata":           {},
	"location-create":                        {},
	"location-merge":                         {},
	"logo-create":                            {},
	"logo-update":                            {},
	"notification-mark-all-read":             {},
	"notification-mark-check-in-read":        {},
	"notification-mark-friend-requests-read": {},
	"notification-mark-read":                 {},
	"notification-mark-unread":               {},
	"notification-preferences-update":        {},
	"product-barcode-create":                 {},
	"product-create":                         {},
	"product-edit-suggestion-create":         {},
	"product-image-create":                   {},
	"product-image-update":                   {},
	"product-list-create":                    {},
	"product-list-item-add":                  {},
	"product-list-update":                    {},
	"product-logo-add":                       {},
	"product-merge":                          {},
	"product-update":                         {},
	"me-avatar-create":                       {},
	"me-avatar-delete":                       {},
	"me-avatar-update":                       {},
	"profile-request-email-change":           {},
	"profile-update":                         {},
	"report-create":                          {},
	"report-resolve":                         {},
	"serving-style-create":                   {},
	"storage-confirm-upload":                 {},
	"storage-request-upload-url":             {},
	"sub-brand-create":                       {},
	"sub-brand-edit-suggestion-create":       {},
	"subcategory-create":                     {},
	"verify":                                 {},
}

func ForMCPTool(name string) (Policy, bool) {
	policy, ok := mcpToolPolicies[strings.TrimSpace(name)]
	if !ok {
		return Policy{}, false
	}
	return policy, true
}

func ForAPIOperation(operationID string) (Policy, bool) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return Policy{}, false
	}
	if policy, ok := apiOperationPolicies[operationID]; ok {
		return policy, true
	}
	if _, ok := readLikePostOperationIDs[operationID]; ok {
		return defaultReadLikePostPolicy, true
	}
	if _, ok := mutationOperationIDs[operationID]; ok {
		return requireIdempotency(defaultMutationPolicy), true
	}
	return Policy{}, false
}

func ForHTTPMethod(method string) (Policy, bool) {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD":
		return defaultReadLikePostPolicy, true
	case "PUT":
		return defaultPutPolicy, true
	case "DELETE":
		return defaultDeletePolicy, true
	default:
		return Policy{}, false
	}
}

func requireIdempotency(policy Policy) Policy {
	policy.IdempotencyRequired = true
	return policy
}

func withCache(policy Policy, cache CachePolicy) Policy {
	policy.Cache = cache
	return policy
}
