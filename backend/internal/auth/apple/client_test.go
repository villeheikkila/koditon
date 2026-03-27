package apple

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func TestVerifyIDToken_RejectsNonceMismatch(t *testing.T) {
	t.Parallel()

	client, signer := newAppleClientForTokenTests(t)
	idToken := signAppleIDToken(t, signer, "com.example.maku", "expected-nonce")

	if _, err := client.VerifyIDToken(context.Background(), idToken, "different-nonce"); err != ErrInvalidNonce {
		t.Fatalf("expected ErrInvalidNonce, got %v", err)
	}
}

func TestVerifyIDToken_AcceptsExactNonce(t *testing.T) {
	t.Parallel()

	client, signer := newAppleClientForTokenTests(t)
	idToken := signAppleIDToken(t, signer, "com.example.maku", "expected-nonce")

	identity, err := client.VerifyIDToken(context.Background(), idToken, "expected-nonce")
	if err != nil {
		t.Fatalf("verify id token: %v", err)
	}
	if identity.Subject != "apple-user-subject" {
		t.Fatalf("expected subject to round-trip, got %q", identity.Subject)
	}
}

func newAppleClientForTokenTests(t *testing.T) (*Client, jwk.Key) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	privateJWK, err := jwk.Import(privateKey)
	if err != nil {
		t.Fatalf("create private jwk: %v", err)
	}
	if err := privateJWK.Set(jwk.AlgorithmKey, jwa.ES256()); err != nil {
		t.Fatalf("set signing algorithm: %v", err)
	}
	if err := privateJWK.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatalf("set key id: %v", err)
	}
	publicJWK, err := privateJWK.PublicKey()
	if err != nil {
		t.Fatalf("derive public jwk: %v", err)
	}
	keySet := jwk.NewSet()
	if err := keySet.AddKey(publicJWK); err != nil {
		t.Fatalf("add public key: %v", err)
	}

	return &Client{
		bundleID: "com.example.maku",
		keySet:   keySet,
	}, privateJWK
}

func signAppleIDToken(t *testing.T, signer jwk.Key, audience string, nonce string) string {
	t.Helper()

	now := time.Now()
	token, err := jwt.NewBuilder().
		Issuer(issuer).
		Subject("apple-user-subject").
		Audience([]string{audience}).
		IssuedAt(now).
		Expiration(now.Add(time.Hour)).
		Claim("nonce", nonce).
		Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), signer))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return string(signed)
}
