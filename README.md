# Terraform Provider msgraph-entra

A Terraform provider for managing Microsoft Entra ID (formerly Azure AD) resources with a current focus on Privileged Identity Management (PIM) features.

## Features

- **Eligible Role Assignments**: Create time-bound eligible assignments for Entra ID directory roles using PIM
- **Scalable Management**: Design patterns for managing role assignments at scale across multiple roles

## Purpose

To see what it was like to write a provider and how much effort was needed to implement time-bound eligible assignments with Terraform in this fashion as opposed to making use of local-exec and inline scripts.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)
- Azure AD tenant with appropriate licenses (P2 or equivalent for PIM features)
- Service Principal with required Graph API permissions:
  - `RoleManagement.ReadWrite.Directory` (for managing role assignments)
  - `Directory.Read.All` (for reading directory objects)

## Building The Provider

```shell
git clone https://github.com/ethorneloe/terraform-provider-msgraph-entra
cd terraform-provider-msgraph-entra
go build -o terraform-provider-msgraph-entra.exe
```

## Using the Provider

### Authentication

The provider supports multiple authentication methods:

#### Service Principal (Client Credentials)

```hcl
provider "msgraph-entra" {
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
provider "msgraph-entra" {
  # Configuration read from environment variables
}
```

#### Azure CLI

```hcl
provider "msgraph-entra" {
  use_cli = true
}
```

### Resources

#### `msgraph-entra_directory_role_eligible_assignment`

Creates an eligible assignment for an Entra ID directory role. Users must activate this assignment through PIM to use the role.

```hcl
# Look up the Security Administrator role
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Look up users by their UPN
data "msgraph-entra_user" "john" {
  user_principal_name = "john.doe@contoso.com"
}

data "msgraph-entra_user" "jane" {
  user_principal_name = "jane.smith@contoso.com"
}

# Create eligible assignment with fixed expiration date using UPN lookup
# Using afterDateTime provides strict enforcement - if someone manually changes
# the end date in the Azure Portal, Terraform will detect drift and revert it
# on the next apply to match your configured end_date_time.
resource "msgraph-entra_directory_role_eligible_assignment" "john_security_admin" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = data.msgraph-entra_user.john.id
  directory_scope_id = "/"
  justification      = "Security team member - requires elevated access for incident response"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-12-31T23:59:59Z"
    }
  }
}

# Example with duration-based expiration (8 hours) using UPN lookup
# IMPORTANT: Using afterDuration protects against configuration drift from manual changes.
# If someone modifies the assignment in the Azure Portal/Entra admin center,
# Terraform will preserve your configured type and duration values and won't
# try to revert the change. This is different from afterDateTime, which will
# cause Terraform to detect drift and revert manual portal changes on next apply.
resource "msgraph-entra_directory_role_eligible_assignment" "jane_temp_admin" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = data.msgraph-entra_user.jane.id
  directory_scope_id = "/"
  justification      = "Temporary access for system maintenance"

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = "PT8H" # 8 hours in ISO 8601 duration format
    }
  }
}
```

**Arguments:**

- `role_definition_id` (Required) - The template ID of the directory role (get from `msgraph-entra_directory_role` data source)
- `principal_id` (Required) - The object ID of the principal (user, group, or service principal)
- `directory_scope_id` (Optional) - The scope of the assignment (default: "/" for tenant-wide)
- `justification` (Optional) - Justification for the assignment
- `schedule_info` (Optional) - Schedule configuration block:
  - `expiration` (Optional) - Expiration configuration:
    - `type` (Optional) - "noExpiration", "afterDateTime", or "afterDuration"
    - `end_date_time` (Optional) - End date/time when type is "afterDateTime" (RFC3339 format)
    - `duration` (Optional) - Duration when type is "afterDuration" (ISO 8601 format, e.g., "PT8H", "P365D")

**Read-Only Attributes:**

- `id` - The ID of the role eligibility schedule
- `schedule_id` - The ID of the created role eligibility schedule
- `schedule_info.start_date_time` - The actual time the eligibility started (set by Microsoft Graph)

### Data Sources

#### `msgraph-entra_directory_role`

Retrieves information about an Entra ID directory role.

```hcl
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

output "security_admin_template_id" {
  value = data.msgraph-entra_directory_role.security_admin.template_id
}
```

**Arguments:**

- `display_name` (Required) - The display name of the directory role

**Attributes:**

- `id` - The object ID of the directory role instance
- `template_id` - The template ID (used for role assignments)
- `description` - The role description

#### `msgraph-entra_user`

Retrieves information about an Entra ID user by their User Principal Name (UPN).

```hcl
data "msgraph-entra_user" "john" {
  user_principal_name = "john.doe@contoso.com"
}

output "john_object_id" {
  value = data.msgraph-entra_user.john.id
}

# Use in role assignment
resource "msgraph-entra_directory_role_eligible_assignment" "john_security_admin" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = data.msgraph-entra_user.john.id
  directory_scope_id = "/"
  justification      = "Security team member"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-12-31T23:59:59Z"
    }
  }
}
```

**Arguments:**

- `user_principal_name` (Required) - The UPN of the user (typically their email)

**Attributes:**

- `id` - The object ID of the user
- `display_name` - The display name
- `mail` - Primary email address
- `given_name` - First name
- `surname` - Last name
- `job_title` - Job title
- `department` - Department
- `account_enabled` - Whether the account is enabled

## Expiration Strategies

### Choosing Between afterDateTime and afterDuration

Understanding the drift detection behavior is crucial for choosing the right expiration type:

| Aspect | `afterDateTime` | `afterDuration` |
|--------|----------------|-----------------|
| **Configuration** | Explicit end date in RFC3339 | Duration in ISO 8601 (e.g., PT8H) |
| **Graph Storage** | Stored as `afterDateTime` + end date | Converted to `afterDateTime` + computed end |
| **Terraform State** | Stores exact `end_date_time` | Stores `type` + `duration` (not end date) |
| **Portal Changes** | **Detected as drift** - reverted on apply | **NOT detected** - manual changes preserved |
| **Use When** | You need strict enforcement of dates | You want flexibility for manual adjustments |
| **Best For** | Compliance, audits, contractors | Break-glass, temporary access, emergencies |

#### Example Scenario

**With afterDateTime:**
- Admin changes end date to 2026-06-30 in Azure Portal
- Next `terraform plan` shows drift: wants to change back to 2025-12-31
- `terraform apply` reverts the manual change ✅ Strict enforcement

**With afterDuration:**
- Admin changes end date in Azure Portal (Graph shows new afterDateTime)
- Next `terraform plan` shows NO drift (compares `duration` not `end_date_time`)
- `terraform apply` does nothing ✅ Manual change preserved

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
      end_date      = "2025-12-31T23:59:59Z"
    },
    {
      upn           = "jane.smith@contoso.com"
      justification = "Security analyst"
      end_date      = "2025-06-30T23:59:59Z"
    },
  ]
}

data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

# Look up users by UPN
data "msgraph-entra_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

resource "msgraph-entra_directory_role_eligible_assignment" "security_admins" {
  for_each = { for admin in local.security_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.security_administrator.template_id
  principal_id       = data.msgraph-entra_user.security_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = each.value.end_date
    }
  }
}
```

### Pattern 2: Centralized Configuration

Manage multiple roles with different expiration strategies in a single configuration file:

```hcl
locals {
  # Security Administrators with fixed expiration dates (strict enforcement)
  security_admins = [
    {
      upn           = "security.lead@contoso.com"
      justification = "Security team lead - annual review cycle"
      end_date      = "2025-12-31T23:59:59Z"
    },
    {
      upn           = "soc.analyst@contoso.com"
      justification = "SOC analyst - 6-month rotation"
      end_date      = "2025-06-30T23:59:59Z"
    },
  ]

  # User Administrators with permanent access
  user_admins = [
    {
      upn           = "helpdesk.lead@contoso.com"
      justification = "Help desk team lead - permanent access required"
    },
  ]
}

# Look up directory roles
data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

data "msgraph-entra_directory_role" "user_administrator" {
  display_name = "User Administrator"
}

# Look up users by UPN
data "msgraph-entra_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

data "msgraph-entra_user" "user_admins" {
  for_each            = { for admin in local.user_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Security Administrator assignments (afterDateTime)
resource "msgraph-entra_directory_role_eligible_assignment" "security_admins" {
  for_each = { for admin in local.security_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.security_administrator.template_id
  principal_id       = data.msgraph-entra_user.security_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = each.value.end_date
    }
  }
}

# User Administrator assignments (noExpiration)
resource "msgraph-entra_directory_role_eligible_assignment" "user_admins" {
  for_each = { for admin in local.user_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.user_administrator.template_id
  principal_id       = data.msgraph-entra_user.user_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type = "noExpiration"
    }
  }
}
```

## Common Directory Roles

Here are some commonly used Entra ID directory roles suitable for Terraform management:

- **Security Administrator** - Security operations and monitoring
- **User Administrator** - User and group management
- **Application Administrator** - Manage enterprise applications
- **Cloud Application Administrator** - Manage cloud applications
- **Authentication Administrator** - Manage authentication methods
- **Conditional Access Administrator** - Manage conditional access policies

## Best Practices

1. **Use Eligible Assignments**: Always prefer eligible over active assignments for privileged roles
2. **Understand Drift Behavior**:
   - `afterDateTime`: Terraform detects and reverts manual portal changes (strict enforcement)
   - `afterDuration`: Terraform preserves manual portal changes (flexible management)
3. **Justifications**: Always provide clear justifications for audit purposes
4. **Read-Only Fields**: Remember that `schedule_info.start_date_time` is set by Microsoft Graph for assignments being created at apply time.  Future start dates are not supported for this provider given that the schedule object doesn't get created until the schedule request is completed.
5. **Regular Reviews**: Periodically review and rotate role assignments
6. **Least Privilege**: Only assign the minimum required roles
7. **Separate Roles**: Use one file per role for better organization and git history
8. **Version Control**: Track all changes in git for audit trails
9.  **User Lookups**: Use the `msgraph-entra_user` data source to look up users by their UPN (email address) - no need to manually obtain object IDs

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
- [examples/user-administrator/](./examples/user-administrator/) - User Administrator role example
- [examples/multiple-roles-schedules/](./examples/multiple-roles-schedules/) - Complete multi-role setup with different expiration strategies

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MPL-2.0
