package apple

type Config struct {
	BundleID     string
	TeamID       string
	PrivateKeyID string
	PrivateKey   string // PEM format
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type IdentityToken struct {
	Issuer         string `json:"iss"`
	Subject        string `json:"sub"`
	Audience       string `json:"aud"`
	IssuedAt       int64  `json:"iat"`
	ExpiresAt      int64  `json:"exp"`
	Nonce          string `json:"nonce,omitempty"`
	NonceSupported bool   `json:"nonce_supported,omitempty"`
	Email          string `json:"email,omitempty"`
	EmailVerified  string `json:"email_verified,omitempty"`
	IsPrivateEmail string `json:"is_private_email,omitempty"`
	RealUserStatus int    `json:"real_user_status,omitempty"`
	AuthTime       int64  `json:"auth_time,omitempty"`
	TransferSub    string `json:"transfer_sub,omitempty"`
}
