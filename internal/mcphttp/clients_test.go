package mcphttp

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func mustHash(t *testing.T, secret string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

func writeClientsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clients.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadClientsFiltersDisabledAndVerifiesSecret(t *testing.T) {
	hash := mustHash(t, "s3cret")
	path := writeClientsFile(t, `{
		"clients": [
			{"client_id": "web", "client_secret_hash": "`+hash+`", "scopes": ["read"], "enabled": true},
			{"client_id": "disabled", "client_secret_hash": "`+hash+`", "scopes": ["read"], "enabled": false}
		]
	}`)
	clients, err := LoadClients(path)
	if err != nil {
		t.Fatalf("LoadClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("got %d clients, want 1 (disabled filtered)", len(clients))
	}
	web, ok := clients["web"]
	if !ok {
		t.Fatal("client web missing")
	}
	if !ClientSecretValid(web, "s3cret") {
		t.Error("ClientSecretValid(web, s3cret) = false, want true")
	}
	if ClientSecretValid(web, "wrong") {
		t.Error("ClientSecretValid(web, wrong) = true, want false")
	}
}

func TestLoadClientsDuplicateID(t *testing.T) {
	hash := mustHash(t, "s3cret")
	path := writeClientsFile(t, `{
		"clients": [
			{"client_id": "dup", "client_secret_hash": "`+hash+`", "enabled": true},
			{"client_id": "dup", "client_secret_hash": "`+hash+`", "enabled": true}
		]
	}`)
	if _, err := LoadClients(path); err == nil {
		t.Fatal("expected error for duplicate client_id")
	}
}

func TestLoadClientsMissingFile(t *testing.T) {
	if _, err := LoadClients(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadClientsBadHash(t *testing.T) {
	path := writeClientsFile(t, `{
		"clients": [
			{"client_id": "bad", "client_secret_hash": "not-a-bcrypt-hash", "enabled": true}
		]
	}`)
	if _, err := LoadClients(path); err == nil {
		t.Fatal("expected error for invalid bcrypt hash")
	}
}
