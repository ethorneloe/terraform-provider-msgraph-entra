# GitHub Actions with OIDC Example

This example demonstrates how to use the `msgraph-entra` provider with OIDC (OpenID Connect) authentication in GitHub Actions. This is the **most secure** way to authenticate because **no secrets are stored in GitHub**.

## Why OIDC?

✅ **No Secrets** - No client secret stored in GitHub Secrets
✅ **Short-Lived Tokens** - Tokens expire after minutes
✅ **Automatic Rotation** - New token for each workflow run
✅ **Better Auditing** - Azure logs show which GitHub workflow authenticated
✅ **Least Privilege** - Tokens are scoped to specific repos/branches

## Prerequisites

### 1. Azure App Registration

You need an Azure App Registration with:
- **Application (client) ID**
- **Tenant ID**
- **Required Permissions:**
  - `RoleManagement.ReadWrite.Directory` (Application)
  - `Directory.Read.All` (Application)
  - Admin consent granted

### 2. Federated Credential Configuration

Configure a federated credential in your Azure App Registration:

**Using Azure Portal:**
1. Go to Azure Portal → Azure Active Directory → App registrations
2. Select your app registration
3. Go to "Certificates & secrets" → "Federated credentials"
4. Click "Add credential"
5. Select "GitHub Actions deploying Azure resources"
6. Fill in:
   - **Organization:** Your GitHub org (e.g., `myorg`)
   - **Repository:** Your repo name (e.g., `terraform-entra-pim`)
   - **Entity type:** Branch
   - **Branch name:** `main`
   - **Name:** `github-actions-main`
7. Click "Add"

**Using Azure CLI:**
```bash
# For main branch
az ad app federated-credential create \
  --id <YOUR_APP_ID> \
  --parameters '{
    "name": "github-actions-main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yourorg/yourrepo:ref:refs/heads/main",
    "description": "GitHub Actions for main branch",
    "audiences": ["api://AzureADTokenExchange"]
  }'

# For pull requests (optional)
az ad app federated-credential create \
  --id <YOUR_APP_ID> \
  --parameters '{
    "name": "github-actions-pr",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yourorg/yourrepo:pull_request",
    "description": "GitHub Actions for pull requests",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

**Subject Patterns:**
- Main branch: `repo:yourorg/yourrepo:ref:refs/heads/main`
- Any branch: `repo:yourorg/yourrepo:ref:refs/heads/*`
- Pull requests: `repo:yourorg/yourrepo:pull_request`
- Specific environment: `repo:yourorg/yourrepo:environment:production`
- Specific tag: `repo:yourorg/yourrepo:ref:refs/tags/v1.0.0`

### 3. GitHub Repository Secrets

Add these secrets to your GitHub repository (Settings → Secrets and variables → Actions):

- `AZURE_TENANT_ID` - Your Azure AD tenant ID
- `AZURE_CLIENT_ID` - Your app registration client ID

**Note:** You do NOT need `AZURE_CLIENT_SECRET` when using OIDC! 🎉

## File Structure

```
your-terraform-repo/
├── .github/
│   └── workflows/
│       └── terraform.yml          ← GitHub Actions workflow (copy from .github-workflows-terraform.yml)
├── main.tf                        ← Terraform configuration (this file)
├── terraform.tfvars               ← Optional: Variable values
└── README.md
```

## Usage

### 1. Copy Files to Your Repository

Copy these files to your repository:

```bash
# Copy the Terraform configuration
cp examples/github-actions-oidc/main.tf your-repo/

# Copy the GitHub Actions workflow
mkdir -p your-repo/.github/workflows
cp examples/github-actions-oidc/.github-workflows-terraform.yml your-repo/.github/workflows/terraform.yml
```

### 2. Customize the Configuration

Edit `main.tf` to match your users and roles:

```hcl
locals {
  security_admins = [
    {
      upn           = "your.user@yourcompany.com"  # Change this
      justification = "Your justification"
      duration      = "P365D"
    },
  ]
}
```

### 3. Commit and Push

```bash
git add .
git commit -m "Add Entra PIM role assignments with OIDC"
git push origin main
```

### 4. Watch It Run

Go to your GitHub repository → Actions tab to see the workflow run.

The workflow will:
1. ✅ Authenticate using OIDC (no secrets!)
2. ✅ Run `terraform init`
3. ✅ Run `terraform plan`
4. ✅ On main branch: Run `terraform apply`

## How OIDC Works

```
┌─────────────────┐
│ GitHub Actions  │
│   Workflow      │
└────────┬────────┘
         │ 1. Request OIDC token
         ├─────────────────────────────┐
         │                             │
         ▼                             ▼
┌─────────────────┐         ┌──────────────────────┐
│ GitHub OIDC     │         │  Azure AD            │
│ Token Service   │──────────▶ Federated Cred     │
└─────────────────┘         │  Validation          │
         │                  └──────────┬───────────┘
         │ 2. JWT Token               │
         ▼                            │ 3. Validate
┌─────────────────┐                  │    - Issuer
│ Terraform       │                  │    - Subject
│ Provider        │                  │    - Audience
└────────┬────────┘                  │
         │ 4. Exchange JWT            │
         │    for Azure token         │
         └────────────────────────────▶
                                      │
                                      ▼
                            ┌──────────────────────┐
                            │ Azure AD Access      │
                            │ Token (short-lived)  │
                            └──────────┬───────────┘
                                       │
                                       ▼
                            ┌──────────────────────┐
                            │ Microsoft Graph API  │
                            │ (Create PIM roles)   │
                            └──────────────────────┘
```

## Environment Variables

The workflow sets these environment variables automatically:

```yaml
env:
  ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}   # From GitHub Secrets
  ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}   # From GitHub Secrets
  ARM_USE_OIDC: true                               # Enables OIDC
```

The provider automatically:
1. Detects `ARM_USE_OIDC=true`
2. Requests an OIDC token from GitHub
3. Exchanges it for an Azure AD access token
4. Uses the token to call Microsoft Graph API

## Testing Locally

You cannot test OIDC locally (it requires GitHub Actions infrastructure). For local testing, use:

### Option 1: Azure CLI
```hcl
provider "msgraph-entra" {
  use_cli = true
}
```

Then run:
```bash
az login
terraform plan
```

### Option 2: Client Secret (Local Only)
```hcl
provider "msgraph-entra" {
  tenant_id     = var.tenant_id
  client_id     = var.client_id
  client_secret = var.client_secret  # From environment or tfvars
}
```

## Troubleshooting

### Error: "AADSTS70021: No matching federated identity record found"

**Cause:** The federated credential subject doesn't match your workflow.

**Solution:**
1. Check your federated credential subject in Azure Portal
2. Ensure it matches exactly:
   ```
   repo:yourorg/yourrepo:ref:refs/heads/main
   ```
3. No typos in org name, repo name, or branch name
4. Include the `repo:` prefix

### Error: "AADSTS700024: Client assertion audience claim does not match"

**Cause:** Wrong audience in federated credential.

**Solution:** Ensure audience is `["api://AzureADTokenExchange"]`

### Error: "Missing permissions"

**Cause:** App registration doesn't have required permissions or admin consent not granted.

**Solution:**
1. Add `RoleManagement.ReadWrite.Directory` permission
2. Add `Directory.Read.All` permission
3. Grant admin consent
4. Wait 5-10 minutes for propagation

### Error: "Provider not found"

**Cause:** Provider binary not available in the GitHub Actions runner.

**Solution:**
- For published providers: Ensure `source` in `required_providers` is correct
- For local/unpublished: You need to publish to Terraform Registry or use a private registry

## Security Best Practices

1. ✅ **Use branch protection** - Require PR reviews before merging to main
2. ✅ **Use environments** - Add approval gates for production
3. ✅ **Scope federated credentials** - Use specific branches/environments
4. ✅ **Enable audit logs** - Monitor Azure AD sign-in logs
5. ✅ **Use least privilege** - Only grant necessary permissions
6. ✅ **Review regularly** - Check role assignments quarterly

## Advanced: Multiple Environments

```yaml
jobs:
  terraform-dev:
    runs-on: ubuntu-latest
    environment: development
    steps:
      - name: Terraform Apply
        run: terraform apply -auto-approve
        env:
          ARM_TENANT_ID: ${{ secrets.DEV_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.DEV_CLIENT_ID }}
          ARM_USE_OIDC: true

  terraform-prod:
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval
    needs: terraform-dev
    steps:
      - name: Terraform Apply
        run: terraform apply -auto-approve
        env:
          ARM_TENANT_ID: ${{ secrets.PROD_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.PROD_CLIENT_ID }}
          ARM_USE_OIDC: true
```

Then create environment-specific federated credentials:
```
Subject: repo:yourorg/yourrepo:environment:production
Subject: repo:yourorg/yourrepo:environment:development
```

## Next Steps

1. ✅ Set up federated credential in Azure
2. ✅ Add GitHub Secrets (AZURE_TENANT_ID, AZURE_CLIENT_ID)
3. ✅ Copy and customize main.tf
4. ✅ Copy workflow file
5. ✅ Push to GitHub
6. ✅ Watch it work! 🚀

No secrets stored, maximum security! 🔒
