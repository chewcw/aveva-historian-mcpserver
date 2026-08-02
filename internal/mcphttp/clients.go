package mcphttp

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// Client is one entry in the clients file: a credential pair allowed to mint
// JWTs via the token endpoint.
type Client struct {
	ClientID         string   `json:"client_id"`
	ClientSecretHash string   `json:"client_secret_hash"`
	Scopes           []string `json:"scopes"`
	Enabled          bool     `json:"enabled"`
}

// ClientsFile is the on-disk format of MCP_CLIENTS_FILE.
type ClientsFile struct {
	Clients []Client `json:"clients"`
}

// LoadClients reads the clients file and returns the enabled clients keyed by
// client_id. Disabled clients are dropped. Malformed entries are hard errors:
// a typo in a hash must not silently disable a credential.
func LoadClients(path string) (map[string]Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read clients file: %w", err)
	}
	var file ClientsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse clients file %s: %w", path, err)
	}
	clients := make(map[string]Client, len(file.Clients))
	for _, c := range file.Clients {
		if c.ClientID == "" {
			return nil, fmt.Errorf("clients file %s: client with empty client_id", path)
		}
		if _, dup := clients[c.ClientID]; dup {
			return nil, fmt.Errorf("clients file %s: duplicate client_id %q", path, c.ClientID)
		}
		if c.ClientSecretHash == "" {
			return nil, fmt.Errorf("clients file %s: client %q has empty client_secret_hash", path, c.ClientID)
		}
		if _, err := bcrypt.Cost([]byte(c.ClientSecretHash)); err != nil {
			return nil, fmt.Errorf("clients file %s: client %q has invalid bcrypt hash: %w", path, c.ClientID, err)
		}
		if !c.Enabled {
			continue
		}
		clients[c.ClientID] = c
	}
	return clients, nil
}

// ClientSecretValid reports whether secret matches the client's stored hash.
func ClientSecretValid(c Client, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(c.ClientSecretHash), []byte(secret)) == nil
}
