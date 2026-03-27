package auth

type AuthProvider = string

const (
	AuthProviderEmail   AuthProvider = "email"
	AuthProviderApple   AuthProvider = "apple"
	AuthProviderPasskey AuthProvider = "passkey"
)

type UserInfo struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}
