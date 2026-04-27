package operationpolicy

import "testing"

func TestForAPIOperation_PostPatchCoverage(t *testing.T) {
	t.Parallel()

	// All POST/PATCH operation IDs from backend/internal/transport/openapi (excluding OAuth and analytics APIs).
	expected := map[string]bool{
		"admin-event-mark-reviewed":              true,
		"auth-email-confirm":                     true,
		"auth-email-request":                     true,
		"auth-email-start":                       false,
		"auth-passkey-authenticate-options":      false,
		"auth-passkey-register-finish":           true,
		"auth-passkey-register-options":          false,
		"auth-sign-out":                          true,
		"auth-sign-out-all":                      true,
		"auth-session-delete":                    true,
		"brand-add-logo":                         true,
		"brand-create":                           true,
		"brand-edit-suggestion-create":           true,
		"brand-like":                             true,
		"category-add-serving-style":             true,
		"category-create":                        true,
		"check-in-comment-create":                true,
		"check-in-create":                        true,
		"check-in-image-create":                  true,
		"check-in-reaction-create":               true,
		"check-in-update":                        true,
		"company-add-logo":                       true,
		"company-create":                         true,
		"company-edit-suggestion-create":         true,
		"company-make-subsidiary":                true,
		"company-merge":                          true,
		"edit-suggestion-resolve":                true,
		"email-confirm-change":                   true,
		"flavor-create":                          true,
		"friend-create":                          true,
		"friend-update":                          true,
		"image-entity-update-metadata":           true,
		"location-create":                        true,
		"location-merge":                         true,
		"logo-create":                            true,
		"logo-update":                            true,
		"notification-mark-all-read":             true,
		"notification-mark-check-in-read":        true,
		"notification-mark-friend-requests-read": true,
		"notification-mark-read":                 true,
		"notification-mark-unread":               true,
		"notification-preferences-update":        true,
		"product-barcode-create":                 true,
		"product-bulk-get":                       false,
		"product-create":                         true,
		"product-edit-suggestion-create":         true,
		"product-image-create":                   true,
		"product-image-update":                   true,
		"product-list-create":                    true,
		"product-list-item-add":                  true,
		"product-list-update":                    true,
		"product-logo-add":                       true,
		"product-merge":                          true,
		"product-searchable-bulk-get":            false,
		"me-avatar-create":                       true,
		"me-avatar-delete":                       true,
		"me-avatar-update":                       true,
		"profile-request-email-change":           true,
		"profile-update":                         true,
		"report-create":                          true,
		"report-resolve":                         true,
		"serving-style-create":                   true,
		"storage-confirm-upload":                 true,
		"storage-request-upload-url":             true,
		"sub-brand-create":                       true,
		"sub-brand-edit-suggestion-create":       true,
		"subcategory-create":                     true,
		"verify":                                 true,
	}

	for operationID, idempotencyRequired := range expected {
		policy, ok := ForAPIOperation(operationID)
		if !ok {
			t.Fatalf("operation %q missing policy coverage", operationID)
		}
		if policy.IdempotencyRequired != idempotencyRequired {
			t.Fatalf("operation %q idempotency required mismatch: got=%v want=%v", operationID, policy.IdempotencyRequired, idempotencyRequired)
		}
		if policy.Mutation == idempotencyRequired {
			continue
		}
		// Read-like POST operations should not be marked as mutation nor require idempotency.
		if !idempotencyRequired && !policy.Mutation {
			continue
		}
		t.Fatalf("operation %q mutation flag mismatch: got=%v want=%v", operationID, policy.Mutation, idempotencyRequired)
	}
}
