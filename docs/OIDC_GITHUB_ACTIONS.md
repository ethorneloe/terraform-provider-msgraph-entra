# Using OIDC Authentication with GitHub Actions

This guide shows how to use the provider with OIDC/Workload Identity Federation in GitHub Actions, which is more secure than storing client secrets.

## Why OIDC?

✅ **No secrets stored in GitHub** - Uses short-lived tokens
✅ **Automatic credential rotation** - Tokens expire quickly
✅ **Better security** - Federated trust instead of long-lived secrets
✅ **Audit trail** - Azure logs show which GitHub workflow authenticated

## Setup Steps

### 1. Configure Azure App Registration for OIDC

```powershell
# Using Azure CLI or PowerShell
$appId = "your-app-registration-client-id"
$tenantId = "your-tenant-id"

# Add federated credential for GitHub Actions
az ad app federated-credential create \
  --id $appId \
  --parameters '{
    "name": "github-actions-main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yourorg/yourrepo:ref:refs/heads/main",
    "description": "GitHub Actions for main branch",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

**Subject patterns:**
- Main branch: `repo:yourorg/yourrepo:ref:refs/heads/main`
- Any branch: `repo:yourorg/yourrepo:ref:refs/heads/*`
- Pull requests: `repo:yourorg/yourrepo:pull_request`
- Specific environment: `repo:yourorg/yourrepo:environment:production`

### 2. Assign Required Permissions

Your app registration needs:
- `RoleManagement.ReadWrite.Directory` (application permission)
- `Directory.Read.All` (application permission)

Grant admin consent for these permissions.

### 3. Configure GitHub Repository Secrets

Add these secrets to your GitHub repository (Settings > Secrets and variables > Actions):

- `AZURE_TENANT_ID` - Your Azure AD tenant ID
- `AZURE_CLIENT_ID` - Your app registration client ID

**Note:** You do NOT need to store `AZURE_CLIENT_SECRET` when using OIDC!

### 4. GitHub Actions Workflow

Create `.github/workflows/terraform.yml`:

```yaml
name: Terraform with OIDC

on:
  push:
    branches: [main]
  pull_request:

permissions:
  id-token: write  # Required for OIDC
  contents: read

jobs:
  terraform:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: 1.6.0

      - name: Terraform Init
        run: terraform init
        env:
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: true

      - name: Terraform Plan
        run: terraform plan
        env:
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: true

      - name: Terraform Apply
        if: github.ref == 'refs/heads/main'
        run: terraform apply -auto-approve
        env:
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: true
```

### 5. Terraform Configuration

Your Terraform configuration doesn't need changes - the provider automatically detects OIDC:

```hcl
terraform {
  required_providers {
    entra = {
      source  = "yourusername/entra"
      version = "~> 1.0"
    }
  }
}

# Provider configuration is read from environment variables
provider "entra" {
  # ARM_TENANT_ID, ARM_CLIENT_ID, and ARM_USE_OIDC from environment
}

# Your resources here...
```

## How It Works

1. GitHub Actions generates a JWT token for your workflow
2. The workflow provides this token to Terraform via `ARM_USE_OIDC=true`
3. The provider exchanges the GitHub token for an Azure AD access token
4. Azure validates the token against the federated credential you configured
5. The provider uses the access token to call Microsoft Graph APIs

## Environment Variable Options

The provider supports multiple environment variable naming conventions:

| Purpose | Option 1 | Option 2 |
|---------|----------|----------|
| Tenant ID | `ENTRA_TENANT_ID` | `ARM_TENANT_ID` |
| Client ID | `ENTRA_CLIENT_ID` | `ARM_CLIENT_ID` |
| OIDC Token | `ENTRA_OIDC_TOKEN` | `ARM_OIDC_TOKEN` |
| Use OIDC | `ENTRA_USE_OIDC=true` | `ARM_USE_OIDC=true` |

Using `ARM_*` variables provides compatibility with the AzureRM provider.

## Troubleshooting

### "AADSTS70021: No matching federated identity record found"

**Cause:** The federated credential subject doesn't match your workflow.

**Solution:** Check your federated credential subject matches exactly:
```bash
az ad app federated-credential list --id <app-id>
```

Common issues:
- Wrong repository name
- Wrong branch name
- Missing `repo:` prefix
- Typo in organization or repository name

### "AADSTS700024: Client assertion audience claim does not match Realm issuer"

**Cause:** Wrong audience in federated credential.

**Solution:** Ensure audience is `["api://AzureADTokenExchange"]`

### "Missing OIDC token"

**Cause:** GitHub Actions doesn't have `id-token: write` permission.

**Solution:** Add to your workflow:
```yaml
permissions:
  id-token: write
  contents: read
```

### Testing OIDC Locally

You cannot easily test OIDC locally since it requires GitHub Actions infrastructure. For local development, use:

- Azure CLI authentication (`use_cli = true`)
- Client secret authentication (for testing only, not production)

## Security Best Practices

1. ✅ **Use environment-specific credentials** - Different app registrations for prod/staging
2. ✅ **Scope federated credentials** - Use specific branches/environments in subject
3. ✅ **Enable conditional access** - Require compliant devices, locations
4. ✅ **Monitor sign-ins** - Review Azure AD sign-in logs regularly
5. ✅ **Least privilege** - Only grant permissions the provider actually needs
6. ✅ **Use environments** - GitHub environments with required reviewers for prod

## Advanced: Multiple Environments

```yaml
jobs:
  terraform-prod:
    runs-on: ubuntu-latest
    environment: production  # Requires approval

    steps:
      - uses: actions/checkout@v4

      - uses: hashicorp/setup-terraform@v3

      - name: Terraform Apply Production
        run: terraform apply -auto-approve
        working-directory: ./environments/production
        env:
          ARM_TENANT_ID: ${{ secrets.PROD_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.PROD_CLIENT_ID }}
          ARM_USE_OIDC: true
```

Federated credential for production environment:
```
Subject: repo:yourorg/yourrepo:environment:production
```

## Comparison with Client Secrets

| Feature | OIDC | Client Secret |
|---------|------|---------------|
| Security | ⭐⭐⭐⭐⭐ Excellent | ⭐⭐⭐ Good |
| Secret Storage | None required | Requires secure storage |
| Token Lifetime | Minutes | Years (unless rotated) |
| Rotation | Automatic | Manual |
| GitHub Actions | Native support | Requires secret |
| Local Development | Not supported | Supported |
| Setup Complexity | Medium | Easy |
| Azure Logs | Shows GitHub workflow | Shows app name only |

**Recommendation:** Use OIDC for all CI/CD pipelines, use client secrets only for local development.

## References

- [Azure Workload Identity Federation](https://learn.microsoft.com/en-us/azure/active-directory/workload-identities/workload-identity-federation)
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [Configure Azure AD for GitHub Actions](https://learn.microsoft.com/en-us/azure/developer/github/connect-from-azure)
