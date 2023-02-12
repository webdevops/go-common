package azidentity

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	EnvAzureEnvironment                = "AZURE_ENVIRONMENT"
	EnvAzureAdditionallyAllowedTenants = "AZURE_ADDITIONALLY_ALLOWED_TENANTS"
	EnvAzureAuthorityHost              = "AZURE_AUTHORITY_HOST"
	EnvAzureClientCertificatePassword  = "AZURE_CLIENT_CERTIFICATE_PASSWORD"
	EnvAzureClientCertificatePath      = "AZURE_CLIENT_CERTIFICATE_PATH"
	EnvAzureClientID                   = "AZURE_CLIENT_ID"
	EnvAzureClientSecret               = "AZURE_CLIENT_SECRET"
	EnvAzureFederatedTokenFile         = "AZURE_FEDERATED_TOKEN_FILE"
	EnvAzurePassword                   = "AZURE_PASSWORD"
	EnvAzureRegionalAuthorityName      = "AZURE_REGIONAL_AUTHORITY_NAME"
	EnvAzureTenantID                   = "AZURE_TENANT_ID"
	EnvAzureUsername                   = "AZURE_USERNAME"
)

func NewAzDefaultCredential(clientOptions *azcore.ClientOptions) (azcore.TokenCredential, error) {
	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		// azurecli authentication
		return NewAzCliCredential()
	case "wi", "workload", "workloadidentity", "federation":
		var tokenFile, tenantID, clientID string
		var ok bool

		if _, ok = os.LookupEnv(EnvAzureAuthorityHost); !ok {
			panic(fmt.Sprintf(`missing environment variable "%s" for workload identity. Check webhook and pod configuration`, EnvAzureAuthorityHost))
		}

		if tokenFile, ok = os.LookupEnv(EnvAzureFederatedTokenFile); !ok {
			panic(fmt.Sprintf(`missing environment variable "%s" for workload identity. Check webhook and pod configuration`, EnvAzureFederatedTokenFile))
		}

		if tenantID, ok = os.LookupEnv(EnvAzureTenantID); !ok {
			panic(fmt.Sprintf(`missing environment variable "%s" for workload identity. Check webhook and pod configuration`, EnvAzureTenantID))
		}

		if clientID, ok = os.LookupEnv(EnvAzureClientID); !ok {
			panic(fmt.Sprintf(`missing environment variable "%s" for workload identity. Check webhook and pod configuration`, EnvAzureClientID))
		}

		opts := azidentity.WorkloadIdentityCredentialOptions{}
		if clientOptions != nil {
			opts.ClientOptions = *clientOptions
		}

		return azidentity.NewWorkloadIdentityCredential(tenantID, clientID, tokenFile, &opts)
	default:
		// general azure authentication (env vars, service principal, msi, ...)
		opts := azidentity.DefaultAzureCredentialOptions{}
		if clientOptions != nil {
			opts.ClientOptions = *clientOptions
		}

		return azidentity.NewDefaultAzureCredential(&opts)
	}
}

func NewAzCliCredential() (azcore.TokenCredential, error) {
	opts := azidentity.AzureCLICredentialOptions{}
	return azidentity.NewAzureCLICredential(&opts)
}
