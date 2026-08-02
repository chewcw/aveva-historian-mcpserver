package mcphttp

import (
	"encoding/json"
	"net/http"
	"time"
)

// tokenHandler implements POST /mcp/token with an OAuth2 client-credentials
// style exchange: valid credentials yield a signed JWT.
type tokenHandler struct {
	clients  map[string]Client
	secret   string
	issuer   string
	audience string
}

func newTokenHandler(clients map[string]Client, secret, issuer, audience string) *tokenHandler {
	return &tokenHandler{clients: clients, secret: secret, issuer: issuer, audience: audience}
}

func (h *tokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	clientID, secret := h.credentials(r)
	if clientID == "" || secret == "" {
		writeTokenError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	client, ok := h.clients[clientID]
	if !ok || !ClientSecretValid(client, secret) {
		writeTokenError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	token, err := IssueToken(client, h.secret, h.issuer, h.audience, tokenTTL)
	if err != nil {
		writeTokenError(w, http.StatusInternalServerError, "server_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int64(tokenTTL / time.Second),
	})
}

// credentials extracts client_id/client_secret from Basic auth or form body.
// Basic auth wins when present.
func (h *tokenHandler) credentials(r *http.Request) (string, string) {
	if id, secret, ok := r.BasicAuth(); ok {
		return id, secret
	}
	if err := r.ParseForm(); err == nil {
		return r.FormValue("client_id"), r.FormValue("client_secret")
	}
	return "", ""
}

func writeTokenError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code})
}
