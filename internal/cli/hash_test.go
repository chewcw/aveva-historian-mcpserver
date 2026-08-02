package cli

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashSecretOutputIsBcrypt(t *testing.T) {
	cmd := NewHashCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"super-secret-value"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	hash := strings.TrimSpace(out.String())
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("super-secret-value")); err != nil {
		t.Fatalf("output %q is not a valid bcrypt hash of the secret: %v", hash, err)
	}
}

func TestHashSecretNoArgs(t *testing.T) {
	cmd := NewHashCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when secret arg missing")
	}
}
