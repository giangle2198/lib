package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	vault "github.com/hashicorp/vault/api"
)

type VaultParameters struct {
	// connection parameters
	Address  string
	TokenKey string

	// the locations / field names of our two secrets
	ApiKeyPath              string
	ApiKeyMountPath         string
	DatabaseCredentialsPath string
}

type Vault struct {
	client     *vault.Client
	parameters VaultParameters
}

// DatabaseCredentials is a set of dynamic credentials retrieved from Vault
type DatabaseCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewVaultAppRoleClient logs in to Vault using the AppRole authentication
// method, returning an authenticated client and the auth token itself, which
// can be periodically renewed.
func NewVaultAppRoleClient(ctx context.Context, parameters VaultParameters) (*Vault, error) {
	log.Printf("connecting to vault @ %s", parameters.Address)

	config := vault.DefaultConfig() // modify for more granular configuration
	config.Address = parameters.Address

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize vault client: %w", err)
	}
	// authorization
	client.SetToken(parameters.TokenKey)

	vault := &Vault{
		client:     client,
		parameters: parameters,
	}

	log.Println("connecting to vault: success!")

	return vault, nil
}

// GetSecretAPIKey fetches the latest version of secret api key from kv-v2
func (v *Vault) GetSecretAPIKeys(ctx context.Context) (map[string]interface{}, error) {
	log.Println("getting secret api key from vault")

	secret, err := v.client.KVv2(v.parameters.ApiKeyMountPath).Get(ctx, v.parameters.ApiKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret: %w", err)
	}

	log.Println("getting secret api key from vault: success!")

	return secret.Data, nil
}

// GetDatabaseCredentials retrieves a new set of temporary database credentials
func (v *Vault) GetDatabaseCredentials(ctx context.Context) (DatabaseCredentials, *vault.Secret, error) {
	log.Println("getting temporary database credentials from vault")

	lease, err := v.client.Logical().ReadWithContext(ctx, v.parameters.DatabaseCredentialsPath)
	if err != nil {
		return DatabaseCredentials{}, nil, fmt.Errorf("unable to read secret: %w", err)
	}

	b, err := json.Marshal(lease.Data)
	if err != nil {
		return DatabaseCredentials{}, nil, fmt.Errorf("malformed credentials returned: %w", err)
	}

	var credentials DatabaseCredentials

	if err := json.Unmarshal(b, &credentials); err != nil {
		return DatabaseCredentials{}, nil, fmt.Errorf("unable to unmarshal credentials: %w", err)
	}

	log.Println("getting temporary database credentials from vault: success!")

	// raw secret is included to renew database credentials
	return credentials, lease, nil
}
