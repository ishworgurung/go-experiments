package vault_client

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

// HashiVault is Hashicorp Vault client
type HashiVault struct {
	path   string
	Client *vault.Client
	Config *vault.Config
}

func NewClient(token string) (*HashiVault, error) {
	vConfig := vault.DefaultConfig()
	vClient, err := vault.NewClient(vConfig)
	if err != nil {
		return nil, fmt.Errorf("err: %w", err)
	}
	vClient.SetToken(token)
	vl := &HashiVault{
		Client: vClient,
		Config: vConfig,
	}
	return vl, nil
}
