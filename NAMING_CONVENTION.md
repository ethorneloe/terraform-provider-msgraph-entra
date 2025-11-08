# Terraform Provider Naming Convention

This document explains the naming conventions used in this provider.

## Summary

**Provider Name**: Uses **dashes** (`msgraph-entra`)
**Resource/Data Source Names**: Use **dashes from provider + underscore + resource type** (`msgraph-entra_directory_role`)

## Why This Matters

Terraform constructs resource type names by concatenating the provider name with the resource type:

1. **Provider names** (in the `terraform` block and `provider` block):
   - ✅ **Can** use dashes: `msgraph-entra`
   - ❌ **Cannot** use underscores: `msgraph_entra` (invalid)

2. **Resource and data source type names**:
   - Terraform automatically creates: `{provider_name}_{resource_type}`
   - Example: `msgraph-entra` + `_directory_role` = `msgraph-entra_directory_role`
   - This is the **correct and expected** format, matching the pattern used by other multi-word providers like `grafana-adaptive-metrics`

## Examples

### ✅ Correct Usage

```hcl
# Provider block - uses dashes
provider "msgraph-entra" {
  tenant_id = "00000000-0000-0000-0000-000000000000"
}

# Data source - provider name (with dashes) + underscore + resource type
data "msgraph-entra_directory_role" "global_admin" {
  display_name = "Global Administrator"
}

# Resource - provider name (with dashes) + underscore + resource type
resource "msgraph-entra_directory_role_eligible_assignment" "example" {
  role_definition_id = data.msgraph-entra_directory_role.global_admin.template_id
  principal_id       = "00000000-0000-0000-0000-000000000000"
  directory_scope_id = "/"
  justification      = "Example assignment"

  schedule_info {
    start_date_time = "2025-01-01T00:00:00Z"
    expiration {
      type     = "afterDuration"
      duration = "P365D"
    }
  }
}
```

### ❌ Incorrect Usage

```hcl
# WRONG - provider with underscore
provider "msgraph_entra" {  # ❌ Invalid - provider names cannot use underscores
  tenant_id = "..."
}

# WRONG - trying to use underscores throughout
data "msgraph_entra_directory_role" "example" {  # ❌ Invalid - doesn't match provider name
  display_name = "..."
}
```

## Real-World Example: Grafana Adaptive Metrics

This naming pattern matches other Terraform providers with multi-word names. For example, the [`grafana-adaptive-metrics` provider](https://github.com/grafana/terraform-provider-grafana-adaptive-metrics):

- **Provider**: `grafana-adaptive-metrics` (uses dashes)
- **Resources**: `grafana-adaptive-metrics_exemption`, `grafana-adaptive-metrics_policy`, etc.

This is the standard Terraform pattern for providers with multi-word names.

## Technical Implementation

### In Go Code

**Provider registration** ([provider.go:36](internal/provider/provider.go#L36)):
```go
func (p *EntraProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    // Provider name is set by Terraform based on registry/local name
    // We don't set it here
}
```

**Data source type names** ([directory_role_data_source.go:37](internal/provider/directory_role_data_source.go#L37)):
```go
func (d *DirectoryRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    // Uses underscore: msgraph_entra_directory_role
    resp.TypeName = req.ProviderTypeName + "_directory_role"
}
```

**Resource type names** ([directory_role_eligible_assignment_resource.go:60](internal/provider/directory_role_eligible_assignment_resource.go#L60)):
```go
func (r *DirectoryRoleEligibleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    // Uses underscore: msgraph_entra_directory_role_eligible_assignment
    resp.TypeName = req.ProviderTypeName + "_directory_role_eligible_assignment"
}
```

### In Test Files

**Provider factory map** ([provider_test.go:19-21](internal/provider/provider_test.go#L19-L21)):
```go
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "msgraph-entra": providerserver.NewProtocol6WithError(New("test")()), // Uses dash
}
```

**Test assertions** ([directory_role_data_source_test.go:21](internal/provider/directory_role_data_source_test.go#L21)):
```go
resource.TestCheckResourceAttr("data.msgraph-entra_directory_role.test", "display_name", "Security Administrator"),
//                                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                                    Uses dashes in provider name, underscore separator
```

## Registry Publishing

When publishing to the Terraform Registry:

- **Namespace**: `ethorneloe`
- **Provider Name**: `msgraph-entra` (with dash)
- **Full Address**: `registry.terraform.io/ethorneloe/msgraph-entra`

Users install it as:
```hcl
terraform {
  required_providers {
    msgraph-entra = {
      source  = "ethorneloe/msgraph-entra"
      version = "~> 1.0"
    }
  }
}
```

## Local Development Override

For local development ([.terraformrc or terraform.rc](README.md)):

```hcl
provider_installation {
  dev_overrides {
    "ethorneloe/msgraph-entra" = "C:\\path\\to\\terraform-provider-msgraph-entra"
  }
  direct {}
}
```

Note the dash in the provider name.

## Summary Table

| Component | Naming Rule | Example |
|-----------|-------------|---------|
| **Provider Name** | Use **dashes** | `msgraph-entra` |
| **Data Sources** | `{provider-name}_{resource_type}` | `msgraph-entra_directory_role` |
| **Resources** | `{provider-name}_{resource_type}` | `msgraph-entra_directory_role_eligible_assignment` |
| **Go Package** | Use **dashes** | `terraform-provider-msgraph-entra` |
| **Registry Path** | Use **dashes** | `registry.terraform.io/ethorneloe/msgraph-entra` |

## Key Takeaway

**Provider name = dashes, Type separator = underscore, Full format = `provider-name_resource_type`**

The Go code uses `req.ProviderTypeName + "_resource_type"` which automatically creates the correct format (`msgraph-entra_directory_role`) since `req.ProviderTypeName` contains the provider name with dashes.

This is the standard Terraform pattern for multi-word provider names, as seen in providers like `grafana-adaptive-metrics`.
