package auth

import (
	"context"
)

type CurrentUserInput struct{}

type CurrentUserOutput struct {
	Body struct {
		FeatureFlags []string `json:"feature_flags" doc:"List of enabled feature flags for the user"`
	}
}

func (h *Handlers) GetCurrentUserInfo(ctx context.Context, _ *CurrentUserInput) (*CurrentUserOutput, error) {
	flags := GetFeatureFlags(ctx)
	if flags == nil {
		flags = []string{}
	}
	return &CurrentUserOutput{
		Body: struct {
			FeatureFlags []string `json:"feature_flags" doc:"List of enabled feature flags for the user"`
		}{
			FeatureFlags: flags,
		},
	}, nil
}
