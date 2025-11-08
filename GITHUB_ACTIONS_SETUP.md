# GitHub Actions OIDC Setup Guide

This guide explains how to configure GitHub Actions to run acceptance tests using OIDC (OpenID Connect) authentication with Azure AD.

## What is OIDC Authentication?

OIDC allows GitHub Actions to authenticate to Azure AD without storing long-lived secrets. Instead:

1. GitHub Actions requests a short-lived OIDC token from GitHub's identity provider
2. The token contains claims about the workflow (repository, branch, commit SHA, etc.)
3. Azure AD validates the token against a federated credential configuration
4. Azure AD issues an access token for Microsoft Graph API
5. The provider uses this token to make API calls

## Benefits

- ✅ No client secrets to rotate or manage
- ✅ Tokens are automatically rotated (valid for ~10 minutes)
- ✅ Scoped to specific repositories and branches
- ✅ Audit trail shows which workflow requested tokens
- ✅ Aligns with security best practices (no long-lived credentials)

## Prerequisites

- Azure AD tenant with Premium P2 license (for PIM features)
- Global Administrator or Application Administrator role
- GitHub repository admin access
- Azure CLI installed (or use Azure Portal)

## Step-by-Step Setup

### Step 1: Create Azure AD App Registration

This app registration represents the GitHub Actions runner and needs permissions to manage PIM role assignments.

**Using Azure CLI:**

```bash
# Login to Azure
az login

# Create the app registration
APP_ID=$(az ad app create \
  --display-name "terraform-provider-msgraph-entra-tests" \
  --query appId -o tsv)

echo "Application (Client) ID: $APP_ID"

# Get your tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)
echo "Tenant ID: $TENANT_ID"
```

**Using Azure Portal:**

1. Go to **Azure AD** → **App registrations** → **New registration**
2. Name: `terraform-provider-msgraph-entra-tests`
3. Supported account types: **Accounts in this organizational directory only**
4. Click **Register**
5. Note the **Application (client) ID** and **Directory (tenant) ID**

### Step 2: Configure Federated Credential

This tells Azure AD to trust OIDC tokens from your GitHub repository.

**Using Azure CLI:**

```bash
# Replace with your GitHub username/org
GITHUB_ORG="ethorneloe"
REPO_NAME="terraform-provider-msgraph-entra"

# Create federated credential for main branch
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "GitHubActions-Main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'"$GITHUB_ORG"'/'"$REPO_NAME"':ref:refs/heads/main",
    "description": "GitHub Actions on main branch",
    "audiences": ["api://AzureADTokenExchange"]
  }'

# Optional: Add credential for pull requests
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "GitHubActions-PR",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'"$GITHUB_ORG"'/'"$REPO_NAME"':pull_request",
    "description": "GitHub Actions on pull requests",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

**Using Azure Portal:**

1. Go to your app registration → **Certificates & secrets** → **Federated credentials**
2. Click **Add credential**
3. Select **GitHub Actions deploying Azure resources**
4. Fill in:
   - **Organization**: `ethorneloe`
   - **Repository**: `terraform-provider-msgraph-entra`
   - **Entity type**: `Branch`
   - **GitHub branch name**: `main`
   - **Name**: `GitHubActions-Main`
5. Click **Add**
6. Repeat for pull requests (Entity type: **Pull request**)

### Step 3: Grant API Permissions

The app needs permissions to manage PIM role assignments.

**Using Azure CLI:**

```bash
# Microsoft Graph API ID
GRAPH_API="00000003-0000-0000-c000-000000000000"

# RoleManagement.ReadWrite.Directory permission ID
ROLE_MGMT_PERMISSION="9e3f62cf-ca93-4989-b6ce-bf83c28f9fe8"

# Add the permission
az ad app permission add \
  --id $APP_ID \
  --api $GRAPH_API \
  --api-permissions $ROLE_MGMT_PERMISSION=Role

# Grant admin consent
az ad app permission admin-consent --id $APP_ID

echo "Permissions granted and consented"
```

**Using Azure Portal:**

1. Go to your app registration → **API permissions**
2. Click **Add a permission** → **Microsoft Graph** → **Application permissions**
3. Search for and select:
   - `RoleManagement.ReadWrite.Directory`
   - `Directory.Read.All` (usually already included)
4. Click **Add permissions**
5. Click **Grant admin consent for [Tenant Name]**
6. Confirm by clicking **Yes**

### Step 4: Get Test Principal ID

You need a user or service principal to test role assignments against.

**For a user:**

```bash
# Using Azure CLI
az ad user show \
  --id user@yourcompany.com \
  --query id -o tsv
```

**For a service principal:**

```bash
# Create a test service principal
TEST_SP_ID=$(az ad sp create-for-rbac \
  --name "test-pim-principal" \
  --skip-assignment \
  --query id -o tsv)

echo "Test Principal ID: $TEST_SP_ID"
```

**Or use an existing principal:**

```bash
# Get existing service principal
az ad sp show \
  --id <app-id> \
  --query id -o tsv
```

### Step 5: Configure GitHub Secrets

Add these secrets to your GitHub repository:

1. Go to your repository → **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret** for each:

| Secret Name | Value | Description |
|------------|-------|-------------|
| `AZURE_TENANT_ID` | Your tenant ID | Azure AD tenant identifier |
| `AZURE_CLIENT_ID` | App registration client ID | The app ID from Step 1 |
| `TEST_PRINCIPAL_ID` | User/SP object ID | Target principal for test assignments |

**Using GitHub CLI:**

```bash
# Set the secrets (replace with your actual values)
gh secret set AZURE_TENANT_ID --body "$TENANT_ID"
gh secret set AZURE_CLIENT_ID --body "$APP_ID"
gh secret set TEST_PRINCIPAL_ID --body "$TEST_SP_ID"
```

## How It Works in the Workflow

The updated [.github/workflows/test.yml](.github/workflows/test.yml#L61-L91) now includes:

```yaml
test:
  name: Terraform Provider Acceptance Tests
  needs: build
  runs-on: ubuntu-latest
  timeout-minutes: 15
  # Required for OIDC authentication to Azure
  permissions:
    id-token: write      # ← Allows requesting OIDC token
    contents: read
  steps:
    # ... setup steps ...
    - name: Run Acceptance Tests
      env:
        TF_ACC: "1"
        ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
        ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
        ARM_USE_OIDC: "true"
        TEST_PRINCIPAL_ID: ${{ secrets.TEST_PRINCIPAL_ID }}
      run: go test -v -cover ./internal/provider/
      timeout-minutes: 10
```

### What happens during execution:

1. **GitHub generates OIDC token**: Because `permissions.id-token: write` is set, GitHub's OIDC provider creates a JWT token with claims like:
   ```json
   {
     "iss": "https://token.actions.githubusercontent.com",
     "sub": "repo:ethorneloe/terraform-provider-msgraph-entra:ref:refs/heads/main",
     "aud": "api://AzureADTokenExchange",
     "repository": "ethorneloe/terraform-provider-msgraph-entra",
     "ref": "refs/heads/main",
     "sha": "abc123...",
     "workflow": "Tests"
   }
   ```

2. **Token is available in environment**: The Azure SDK automatically retrieves this token when `ARM_USE_OIDC=true`

3. **Provider authenticates**: The provider's client.go code ([lines 36-40](internal/provider/client.go#L36-L40)) detects OIDC mode:
   ```go
   } else if oidcToken := getEnvWithFallback("ENTRA_OIDC_TOKEN", "ARM_OIDC_TOKEN"); oidcToken != "" {
       // Use OIDC token authentication
       cred, err = azidentity.NewClientAssertionCredential(tenantID, clientID, func(ctx context.Context) (string, error) {
           return oidcToken, nil
       }, nil)
   }
   ```

4. **Azure validates token**: Azure AD checks:
   - Token signature is valid (signed by GitHub)
   - Issuer is `https://token.actions.githubusercontent.com`
   - Subject matches federated credential configuration
   - Audience is `api://AzureADTokenExchange`

5. **Access token issued**: If validation succeeds, Azure AD returns a Graph API access token

6. **Tests run**: All API calls use this token until it expires (~10 minutes)

## Verification

After setup, test the configuration:

1. **Trigger the workflow**:
   ```bash
   # Make a small change and push
   git commit --allow-empty -m "Test OIDC authentication"
   git push
   ```

2. **Check workflow logs**:
   - Go to **Actions** tab in GitHub
   - Click on the **Tests** workflow run
   - Check the **Terraform Provider Acceptance Tests** job
   - Look for authentication success in logs

3. **Expected output**:
   ```
   === RUN   TestAccDirectoryRoleDataSource
   provider_test.go:88: Using authentication method: CLI=false, ClientCreds=false, OIDC=true
   --- PASS: TestAccDirectoryRoleDataSource (2.34s)
   ```

## Troubleshooting

### Error: "Failed to obtain a token from the tenant"

**Cause**: Federated credential subject doesn't match the workflow.

**Fix**: Verify the subject in your federated credential matches exactly:
- For main branch: `repo:ethorneloe/terraform-provider-msgraph-entra:ref:refs/heads/main`
- For PRs: `repo:ethorneloe/terraform-provider-msgraph-entra:pull_request`

### Error: "permission id-token: write is required"

**Cause**: Missing permissions in workflow file.

**Fix**: Ensure test.yml has:
```yaml
permissions:
  id-token: write
  contents: read
```

### Error: "RoleManagement.ReadWrite.Directory permission required"

**Cause**: API permission not granted or admin consent missing.

**Fix**:
1. Go to app registration → API permissions
2. Verify `RoleManagement.ReadWrite.Directory` shows status **Granted for [Tenant]**
3. If not, click **Grant admin consent**

### Error: "Context access might be invalid: AZURE_TENANT_ID"

**Cause**: GitHub secret doesn't exist yet.

**Fix**: Add the secret in repository settings (see Step 5 above).

### Tests skip with "No authentication method available"

**Cause**: Secrets not configured or workflow not passing them.

**Fix**: Verify all three secrets are set:
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `TEST_PRINCIPAL_ID`

## Security Considerations

1. **Scope federated credentials narrowly**: Use specific branches, not wildcards
2. **Use separate app registrations**: Don't reuse production app registrations for tests
3. **Least privilege**: Only grant necessary API permissions
4. **Audit regularly**: Review Azure AD sign-in logs for unusual activity
5. **Protect secrets**: Ensure GitHub repository secrets are only accessible to administrators

## Alternative: Client Secret Authentication

If OIDC is not feasible, you can use client secret authentication:

1. Create a client secret in the app registration
2. Add `AZURE_CLIENT_SECRET` to GitHub secrets
3. Remove `ARM_USE_OIDC: "true"` from workflow
4. Provider will automatically use client secret flow

**However, OIDC is recommended because:**
- No secret rotation needed
- Shorter validity period (better security)
- Better audit trail with workflow context

## Local Testing

For local development, you can use Azure CLI authentication:

```bash
# Login with Azure CLI
az login

# Set test principal
export TEST_PRINCIPAL_ID="00000000-0000-0000-0000-000000000000"

# Run tests
TF_ACC=1 go test -v ./internal/provider/ -timeout 30m
```

See [TESTING.md](TESTING.md) for comprehensive local testing instructions.

## Support

For issues with OIDC setup:
- Review [GitHub OIDC documentation](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- Review [Azure federated credentials documentation](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation)
- Open an issue at https://github.com/ethorneloe/terraform-provider-msgraph-entra/issues
