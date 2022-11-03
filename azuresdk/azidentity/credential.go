package azidentity

import (
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func NewAzCredential(clientOptions *azcore.ClientOptions) (azcore.TokenCredential, error) {
	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		// azurecli authentication
		opts := azidentity.AzureCLICredentialOptions{}
		return azidentity.NewAzureCLICredential(&opts)
	default:
		// general azure authentication (env vars, service principal, msi, ...)
		opts := azidentity.DefaultAzureCredentialOptions{}
		if clientOptions != nil {
			opts.ClientOptions = *clientOptions
		}

		return azidentity.NewDefaultAzureCredential(&opts)
	}
}
