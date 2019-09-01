package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

var (
	accountKeysCache        = map[string]string{}
	resourceGroupNamesCache = map[string]string{}
	writeLock               = sync.RWMutex{}
)

func (client Client) FindResourceGroup(ctx context.Context, accountName string) (*string, error) {
	cacheKey := accountName
	if v, ok := resourceGroupNamesCache[cacheKey]; ok {
		return &v, nil
	}

	accounts, err := client.AccountsClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error listing Storage Accounts (to find Resource Group for %q): %s", accountName, err)
	}

	if accounts.Value == nil {
		return nil, nil
	}

	var resourceGroup *string
	for _, account := range *accounts.Value {
		if account.Name == nil || account.ID == nil {
			continue
		}

		if strings.EqualFold(accountName, *account.Name) {
			id, err := azure.ParseAzureResourceID(*account.ID)
			if err != nil {
				return nil, fmt.Errorf("Error parsing ID for Storage Account %q: %s", accountName, err)
			}

			resourceGroup = &id.ResourceGroup
			break
		}
	}

	if resourceGroup != nil {
		writeLock.Lock()
		resourceGroupNamesCache[cacheKey] = *resourceGroup
		writeLock.Unlock()
	}

	return resourceGroup, nil
}

func (client Client) findAccountKey(ctx context.Context, resourceGroup, accountName string) (*string, error) {
	cacheKey := fmt.Sprintf("%s-%s", resourceGroup, accountName)
	if v, ok := accountKeysCache[cacheKey]; ok {
		return &v, nil
	}

	props, err := client.AccountsClient.ListKeys(ctx, resourceGroup, accountName)
	if err != nil {
		return nil, fmt.Errorf("Error Listing Keys for Storage Account %q (Resource Group %q): %+v", accountName, resourceGroup, err)
	}

	if props.Keys == nil || len(*props.Keys) == 0 {
		return nil, fmt.Errorf("Keys were nil for Storage Account %q (Resource Group %q): %+v", accountName, resourceGroup, err)
	}

	keys := *props.Keys
	firstKey := keys[0].Value

	writeLock.Lock()
	accountKeysCache[cacheKey] = *firstKey
	writeLock.Unlock()

	return firstKey, nil
}
