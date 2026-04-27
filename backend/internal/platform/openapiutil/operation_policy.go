package openapiutil

import (
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon/internal/platform/operationpolicy"
)

func ApplyOperationPolicyDocumentation(op *huma.Operation) {
	if op == nil || strings.TrimSpace(op.OperationID) == "" {
		return
	}
	policy, ok := operationpolicy.ForAPIOperation(op.OperationID)
	if !ok {
		policy, ok = operationpolicy.ForHTTPMethod(op.Method)
	}
	if !ok {
		return
	}

	policyDoc := map[string]any{
		"retry": map[string]any{
			"max_attempts": policy.MaxAttempts,
			"retry_on": map[string]any{
				"http_status_codes": policy.RetryableStatuses,
				"transport_errors":  policy.RetryTransportErrors,
			},
			"backoff": map[string]any{
				"strategy":      "exponential",
				"base_delay_ms": int(policy.RetryBaseBackoff / time.Millisecond),
				"jitter":        retryJitterMode(policy.RetryJitter),
			},
			"respect_retry_after_header": true,
		},
		"mutation": policy.Mutation,
	}

	descriptionNotes := make([]string, 0, 2)
	if policy.RateLimit > 0 && policy.RateWindow > 0 {
		policyDoc["rate_limit"] = map[string]any{
			"limit":          policy.RateLimit,
			"window_seconds": int(policy.RateWindow / time.Second),
			"response_headers": []string{
				"RateLimit-Limit",
				"RateLimit-Remaining",
				"RateLimit-Reset",
				"Retry-After",
			},
			"status_code": 429,
		}
		descriptionNotes = append(descriptionNotes, "This operation is rate-limited and may return 429 with Retry-After and RateLimit-* headers.")
	}
	if policy.Cache.HTTPEnabled() {
		policyDoc["cache"] = map[string]any{
			"scope":                          string(policy.Cache.HTTP.Scope),
			"max_age_seconds":                int(policy.Cache.HTTP.MaxAge / time.Second),
			"stale_while_revalidate_seconds": int(policy.Cache.HTTP.StaleWhileRevalidate / time.Second),
			"stale_if_error_seconds":         int(policy.Cache.HTTP.StaleIfError / time.Second),
			"must_revalidate":                policy.Cache.HTTP.MustRevalidate,
			"etag":                           policy.Cache.HTTP.UseETag,
			"last_modified":                  policy.Cache.HTTP.UseLastModified,
			"request_headers": []string{
				"If-None-Match",
				"If-Modified-Since",
			},
			"response_headers": []string{
				"Cache-Control",
				"Date",
				"ETag",
				"Expires",
				"Last-Modified",
				"Vary",
			},
		}
		descriptionNotes = append(
			descriptionNotes,
			"This operation may be cached by private clients and supports standard conditional requests via ETag and Last-Modified.",
		)
	}
	if policy.IdempotencyRequired {
		policyDoc["idempotency"] = map[string]any{
			"required":    true,
			"header":      "Idempotency-Key",
			"ttl_seconds": int(policy.IdempotencyTTL / time.Second),
		}
		ensureIdempotencyHeaderParameter(op)
	}

	if op.Extensions == nil {
		op.Extensions = map[string]any{}
	}
	op.Extensions["x-koditon-operation-policy"] = policyDoc

	if len(descriptionNotes) == 0 {
		return
	}
	note := strings.Join(descriptionNotes, " ")
	if strings.TrimSpace(op.Description) == "" {
		op.Description = note
		return
	}
	op.Description = strings.TrimSpace(op.Description) + " " + note
}

func retryJitterMode(enabled bool) string {
	if enabled {
		return "full"
	}
	return "none"
}

func ensureIdempotencyHeaderParameter(op *huma.Operation) {
	if op == nil {
		return
	}
	for _, parameter := range op.Parameters {
		if parameter == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parameter.In), "header") && strings.EqualFold(strings.TrimSpace(parameter.Name), "Idempotency-Key") {
			parameter.Required = true
			return
		}
	}
	op.Parameters = append(op.Parameters, &huma.Param{
		Name:        "Idempotency-Key",
		In:          "header",
		Description: "Unique key used to make mutation retries safe and replay the original response.",
		Required:    true,
		Schema: &huma.Schema{
			Type:      "string",
			MinLength: new(1),
		},
	})
}
