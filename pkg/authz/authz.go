package authz

import "github.com/openfga/go-sdk/client"

func NewFGAClient(apiUrl string, storeId string) (*client.OpenFgaClient, error) {
	cfg := client.ClientConfiguration{
		ApiUrl:  apiUrl,
		StoreId: storeId,
	}

	fga, err := client.NewSdkClient(&cfg)
	if err != nil {
		return nil, err
	}

	return fga, nil
}
