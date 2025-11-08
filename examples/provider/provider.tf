# Example: Configure the msgraph-entra provider with service principal authentication
provider "msgraph-entra" {
  tenant_id     = "00000000-0000-0000-0000-000000000000" # Your Azure AD tenant ID
  client_id     = "00000000-0000-0000-0000-000000000000" # Service principal client ID
  client_secret = var.client_secret                      # Service principal secret (use variable for security)
}

# Alternative: Use OIDC for GitHub Actions (no secrets!)
provider "msgraph-entra" {
  tenant_id  = var.tenant_id
  client_id  = var.client_id
  oidc_token = var.oidc_token # Automatically provided by GitHub Actions with ARM_USE_OIDC=true
}

# Alternative: Use Azure CLI authentication
provider "msgraph-entra" {
  use_cli = true
}

# Alternative: Use environment variables
# Set ARM_TENANT_ID, ARM_CLIENT_ID, ARM_CLIENT_SECRET or ARM_USE_OIDC=true
provider "msgraph-entra" {
  # Configuration will be read from environment variables
}
