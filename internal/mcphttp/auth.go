package mcphttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenTTL is how long minted JWTs remain valid.
const tokenTTL = time.Hour

// Claims is the JWT payload for MCP HTTP access tokens.
type Claims struct {
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

type claimsCtxKey struct{}

// IssueToken mints a signed HS256 JWT for a client.
func IssueToken(c Client, secret, issuer, audience string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		Scopes: c.Scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   c.ClientID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// validateToken verifies signature, issuer, audience, and expiry.
func validateToken(tokenStr, secret, issuer, audience string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}
	if !parsed.Valid {
		return nil, errors.New("validate token: invalid token")
	}
	return claims, nil
}

// requireAuth rejects requests without a valid Bearer JWT and exposes the
// parsed claims via the request context.
func requireAuth(issuer, audience, secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth schemes are case-insensitive per RFC 7235; match "bearer" too.
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(w)
			return
		}
		claims, err := validateToken(parts[1], secret, issuer, audience)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}
