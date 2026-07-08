// Package auth provides the primitives every session in this app is built
// from: random opaque tokens, their storage-safe hash, and an HMAC signer
// so a tampered cookie is rejected before it ever reaches the DB. Used by
// both guest cart sessions and admin login sessions.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrInvalidToken = errors.New("auth: invalid token")

// GenerateToken returns a new random 256-bit token, base64url-encoded.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the token's sha256 digest, base64url-encoded, for DB
// storage/lookup. The raw token is never persisted — only its hash — so a
// DB leak alone can't be used to hijack a session.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Signer HMAC-signs tokens for cookie storage.
type Signer struct {
	key []byte
}

func NewSigner(key []byte) Signer {
	return Signer{key: key}
}

func (s Signer) Sign(token string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(token))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token + "." + sig
}

// Verify checks a signed cookie value and returns the raw token.
func (s Signer) Verify(cookieValue string) (string, error) {
	token, sig, found := strings.CutLast(cookieValue, ".")
	if !found {
		return "", ErrInvalidToken
	}

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(token))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return "", ErrInvalidToken
	}
	return token, nil
}
