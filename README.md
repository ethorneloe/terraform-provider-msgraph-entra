# Terraform Provider msgraph_entra (Azure AD)

A Terraform provider for managing Microsoft Entra ID (formerly Azure AD) resources with a focus on Privileged Identity Management (PIM) features.

## Features

- **Eligible Role Assignments**: Create time-bound eligible assignments for Entra ID directory roles using PIM
- **Flexible Authentication**: Support for service principal (client credentials), Azure CLI authentication
- **Scalable Management**: Design patterns for managing role assignments at scale across multiple roles

## Why This Provider?

The existing Terraform providers have gaps when it comes to PIM management:

- **hashicorp/azuread**: Doesn't support PIM eligible role assignments for Entra ID directory roles
- **hashicorp/azurerm**: Only supports Azure RBAC roles (subscription/resource-level), not Entra ID directory roles
- **microsoft/msgraph**: Still in preview with limited PIM support

This provider fills those gaps by providing direct support for Entra ID directory role eligible assignments through the Microsoft Graph API.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)
- Azure AD tenant with appropriate licenses (P2 or equivalent for PIM features)
- Service Principal with required Graph API permissions:
  - `RoleManagement.ReadWrite.Directory` (for managing role assignments)
  - `Directory.Read.All` (for reading directory objects)

## Building The Provider

```shell
git clone <repository-url>
cd terraform-provider-msgraph_entra
go build -o terraform-provider-msgraph_entra.exe
```

## Using the Provider

### Authentication

The provider supports multiple authentication methods:

#### Service Principal (Client Credentials)

```hcl
provider "msgraph_entra" {
  tenant_id     = "00000000-0000-0000-0000-000000000000"
  client_id     = "00000000-0000-0000-0000-000000000000"
  client_secret = var.client_secret
}
```

#### Environment Variables

```shell
export ENTRA_TENANT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_CLIENT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_CLIENT_SECRET="your-secret-here"
```

```hcl
provider "msgraph_entra" {
  # Configuration read from environment variables
}
```

#### Azure CLI

```hcl
provider "msgraph_entra" {
  use_cli = true
}
```

### Resources

#### `entra_directory_role_eligible_assignment`

Creates an eligible assignment for an Entra ID directory role. Users must activate this assignment through PIM to use the role.

```hcl
# Look up the Security Administrator role
data "msgraph_entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Look up the user
data "azuread_user" "john" {
  user_principal_name = "john.doe@contoso.com"
}

# Create eligible assignment
resource "msgraph_entra_directory_role_eligible_assignment" "john_security_admin" {
  role_definition_id = data.entra_directory_role.security_admin.template_id
  principal_id       = data.azuread_user.john.id
  directory_scope_id = "/"
  justification      = "Security team member - requires elevated access for incident response"

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"

    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-12-31T23:59:59Z"
    }
  }
}
```

**Arguments:**

- `role_definition_id` (Required) - The template ID of the directory role (get from `entra_directory_role` data source)
- `principal_id` (Required) - The object ID of the principal (user, group, or service principal)
- `directory_scope_id` (Optional) - The scope of the assignment (default: "/" for tenant-wide)
- `justification` (Optional) - Justification for the assignment
- `schedule_info` (Optional) - Schedule configuration block:
  - `start_date_time` (Optional) - When the eligibility starts (RFC3339 format, defaults to now)
  - `expiration` (Optional) - Expiration configuration:
    - `type` (Optional) - "noExpiration", "afterDateTime", or "afterDuration"
    - `end_date_time` (Optional) - End date/time when type is "afterDateTime" (RFC3339 format)
    - `duration` (Optional) - Duration when type is "afterDuration" (ISO 8601 format, e.g., "PT8H", "P365D")

**Attributes:**

- `id` - The ID of the role eligibility schedule request
- `schedule_id` - The ID of the created role eligibility schedule

### Data Sources

#### `entra_directory_role`

Retrieves information about an Entra ID directory role.

```hcl
data "msgraph_entra_directory_role" "global_admin" {
  display_name = "Global Administrator"
}

output "global_admin_template_id" {
  value = data.entra_directory_role.global_admin.template_id
}
```

**Arguments:**

- `display_name` (Required) - The display name of the directory role

**Attributes:**

- `id` - The object ID of the directory role instance
- `template_id` - The template ID (used for role assignments)
- `description` - The role description

## Scalable Patterns

### Pattern 1: One File Per Role

Create a separate file for each role with all assignments defined in a local variable:

```hcl
# security-admins.tf
locals {
  security_admins = [
    {
      upn           = "john.doe@contoso.com"
      justification = "Security team lead"
      duration      = "P365D"
    },
    {
      upn           = "jane.smith@contoso.com"
      justification = "Security analyst"
      duration      = "P180D"
    },
  ]
}

data "msgraph_entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

data "azuread_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

resource "msgraph_entra_directory_role_eligible_assignment" "security_admins" {
  for_each = { for admin in local.security_admins : admin.upn => admin }

  role_definition_id = data.entra_directory_role.security_administrator.template_id
  principal_id       = data.azuread_user.security_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"
    expiration {
      type     = "afterDuration"
      duration = each.value.duration
    }
  }
}
```

### Pattern 2: Centralized Configuration

Manage all roles in a single configuration file:

```hcl
locals {
  role_assignments = {
    "Global Administrator" = [
      { upn = "breakglass1@contoso.com", justification = "Break-glass", duration = "P365D" },
    ]
    "Security Administrator" = [
      { upn = "security.lead@contoso.com", justification = "SOC lead", duration = "P365D" },
      { upn = "security.analyst@contoso.com", justification = "SOC analyst", duration = "P180D" },
    ]
  }

  all_assignments = flatten([
    for role_name, assignments in local.role_assignments : [
      for assignment in assignments : {
        key           = "${role_name}|${assignment.upn}"
        role_name     = role_name
        upn           = assignment.upn
        justification = assignment.justification
        duration      = assignment.duration
      }
    ]
  ])
}

data "msgraph_entra_directory_role" "roles" {
  for_each     = toset(keys(local.role_assignments))
  display_name = each.key
}

data "azuread_user" "users" {
  for_each            = toset([for assignment in local.all_assignments : assignment.upn])
  user_principal_name = each.key
}

resource "msgraph_entra_directory_role_eligible_assignment" "assignments" {
  for_each = { for assignment in local.all_assignments : assignment.key => assignment }

  role_definition_id = data.entra_directory_role.roles[each.value.role_name].template_id
  principal_id       = data.azuread_user.users[each.value.upn].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"
    expiration {
      type     = "afterDuration"
      duration = each.value.duration
    }
  }
}
```

## Common Directory Roles

Here are some commonly used Entra ID directory roles:

- **Global Administrator** - Highest privilege level, use sparingly
- **Security Administrator** - Security operations and monitoring
- **User Administrator** - User and group management
- **Privileged Role Administrator** - Manage role assignments
- **Application Administrator** - Manage enterprise applications
- **Cloud Application Administrator** - Manage cloud applications
- **Authentication Administrator** - Manage authentication methods
- **Conditional Access Administrator** - Manage conditional access policies

## Best Practices

1. **Use Eligible Assignments**: Always prefer eligible over active assignments for privileged roles
2. **Time-Bound Access**: Use expiration dates or durations, avoid "noExpiration"
3. **Justifications**: Always provide clear justifications for audit purposes
4. **Break-Glass Accounts**: Use eligible assignments even for emergency access accounts
5. **Regular Reviews**: Periodically review and rotate role assignments
6. **Least Privilege**: Only assign the minimum required roles
7. **Separate Roles**: Use one file per role for better organization and git history
8. **Version Control**: Track all changes in git for audit trails

## Architecture with Access Packages

This provider works excellently with the pattern of **role-assignable groups + access packages**:

1. Create a role-assignable PIM-enabled group (using `hashicorp/azuread` provider)
2. Assign the Entra role to the group using this provider (eligible assignment)
3. Create access packages (using `hashicorp/azuread` provider) that grant group membership
4. Users request access through access packages, get group membership, then activate the role

This architecture provides:
- Self-service access requests
- Approval workflows
- Automated expiration
- Just-in-time elevation
- Full audit trails

## Troubleshooting

### Permission Errors

Ensure your service principal has the required Graph API permissions:

```powershell
# Using Microsoft Graph PowerShell
Connect-MgGraph -Scopes "Application.ReadWrite.All"

# Get your app
$app = Get-MgApplication -Filter "appId eq 'your-client-id'"

# Add required permissions
$params = @{
    RequiredResourceAccess = @(
        @{
            ResourceAppId = "00000003-0000-0000-c000-000000000000" # Microsoft Graph
            ResourceAccess = @(
                @{
                    Id = "9e3f62cf-ca93-4989-b6ce-bf83c28f9fe8" # RoleManagement.ReadWrite.Directory
                    Type = "Role"
                },
                @{
                    Id = "7ab1d382-f21e-4acd-a863-ba3e13f7da61" # Directory.Read.All
                    Type = "Role"
                }
            )
        }
    )
}

Update-MgApplication -ApplicationId $app.Id -BodyParameter $params

# Admin consent required
```

### Common Issues

1. **"Role not found"**: Ensure the role display name is exact (case-sensitive)
2. **"Invalid duration"**: Use ISO 8601 format (e.g., "PT8H" for 8 hours, "P30D" for 30 days)
3. **"Invalid date/time"**: Use RFC3339 format (e.g., "2025-01-08T00:00:00Z")

## Examples

See the [examples/](./examples/) directory for complete examples:

- [examples/provider/](./examples/provider/) - Provider configuration examples
- [examples/security-administrator/](./examples/security-administrator/) - Security Administrator role example
- [examples/global-administrator/](./examples/global-administrator/) - Global Administrator role example
- [examples/user-administrator/](./examples/user-administrator/) - User Administrator role example
- [examples/complete-setup/](./examples/complete-setup/) - Complete multi-role setup

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MPL-2.0
