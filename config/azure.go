// Package config provides Azure Key Vault integration for secrets management.
package config

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// KeyVaultClient wraps Azure Key Vault secret operations.
type KeyVaultClient struct {
	client *azsecrets.Client
}

// NewKeyVaultClient creates a new Key Vault client using DefaultAzureCredential.
func NewKeyVaultClient(vaultName string) (*KeyVaultClient, error) {
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	return &KeyVaultClient{client: client}, nil
}

// GetSecret retrieves a secret value from Key Vault.
func (kv *KeyVaultClient) GetSecret(ctx context.Context, name string) (string, error) {
	resp, err := kv.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", name, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has no value", name)
	}

	return *resp.Value, nil
}

// SetSecret stores a secret value in Key Vault.
func (kv *KeyVaultClient) SetSecret(ctx context.Context, name, value string) error {
	params := azsecrets.SetSecretParameters{
		Value: &value,
	}

	_, err := kv.client.SetSecret(ctx, name, params, nil)
	if err != nil {
		return fmt.Errorf("failed to set secret %s: %w", name, err)
	}

	return nil
}
