package auth

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

type AuthTokensResponse struct {
	AccessToken           string `json:"access_token" doc:"JWT access token for API authentication"`
	AccessTokenExpiresAt  int64  `json:"access_token_expires_at" doc:"Unix timestamp when access token expires"`
	RefreshToken          string `json:"refresh_token" doc:"JWT refresh token for obtaining new access tokens"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at" doc:"Unix timestamp when refresh token expires"`
	UserID                string `json:"user_id" format:"uuid" doc:"User ID"`
	IsNewUser             bool   `json:"is_new_user" doc:"Whether this is a newly created user"`
}
