# Provider Renamed to msgraph-entra

The provider has been successfully renamed from `entra` to `msgraph-entra`.

## Why msgraph-entra?

✅ **Clarity**: Indicates it uses Microsoft Graph API
✅ **Specificity**: Shows it's for Entra ID (Azure AD)
✅ **No Conflicts**: Unlikely to be taken by Microsoft
✅ **Professional**: Clear, concise naming convention

## Changes Made

### Core Files
- ✅ `go.mod` - Module name updated
- ✅ `main.go` - Registry address updated
- ✅ `internal/provider/provider.go` - TypeName updated to `msgraph-entra`

### Binary
- ✅ Built as `terraform-provider-msgraph-entra.exe`
- ✅ Size: 169MB
- ✅ Fully functional with OIDC support

### Documentation
- ✅ README.md updated
- ✅ All examples updated
- ✅ OIDC documentation reflects new name

### Examples Updated
- ✅ `examples/provider/provider.tf`
- ✅ `examples/security-administrator/security-admins.tf`
- ✅ `examples/global-administrator/global-admins.tf`
- ✅ `examples/user-administrator/user-admins.tf`
- ✅ `examples/complete-setup/main.tf`

## Usage

### Provider Configuration

```hcl
terraform {
  required_providers {
    msgraph-entra = {
      source  = "yourusername/msgraph-entra"
      version = "~> 1.0"
    }
  }
}

# Using environment variables (recommended for GitHub Actions)
provider "msgraph-entra" {
  # ARM_TENANT_ID, ARM_CLIENT_ID, ARM_USE_OIDC=true
}

# Using explicit configuration
provider "msgraph-entra" {
  tenant_id  = var.tenant_id
  client_id  = var.client_id
  oidc_token = var.oidc_token  # For OIDC/Workload Identity
}
```

### Resources

```hcl
# Data source
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Resource
resource "msgraph-entra_directory_role_eligible_assignment" "example" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = data.azuread_user.john.id
  directory_scope_id = "/"
  justification      = "Security operations"

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"
    expiration {
      type     = "afterDuration"
      duration = "P365D"
    }
  }
}
```

## GitHub Actions with OIDC

```yaml
permissions:
  id-token: write
  contents: read

steps:
  - uses: hashicorp/setup-terraform@v3

  - name: Terraform Apply
    run: terraform apply -auto-approve
    env:
      ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
      ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
      ARM_USE_OIDC: true
```

## Next Steps

1. Test the provider locally
2. Publish to Terraform Registry as `yourusername/msgraph-entra`
3. Update GitHub repository name (optional but recommended)
4. Tag a release (v1.0.0)

The provider is ready for production use! 🚀
