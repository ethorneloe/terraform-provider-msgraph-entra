# Example: Using OIDC Authentication in GitHub Actions
# This example demonstrates how to configure the provider to use
# OIDC/Workload Identity Federation for GitHub Actions pipelines

terraform {
  required_providers {
    msgraph_entra = {
      source  = "yourusername/msgraph_entra"
      version = "~> 1.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.0"
    }
  }
}

# ============================================================================
# OIDC Authentication Configuration
# ============================================================================
# When using GitHub Actions with OIDC, the provider automatically uses
# the OIDC token when ARM_USE_OIDC=true is set in the environment.
# No client_secret is needed!

provider "msgraph_entra" {
  # These values come from GitHub Secrets:
  # - ARM_TENANT_ID = ${{ secrets.AZURE_TENANT_ID }}
  # - ARM_CLIENT_ID = ${{ secrets.AZURE_CLIENT_ID }}
  # - ARM_USE_OIDC = true
  #
  # The OIDC token is automatically obtained from GitHub Actions
  # when permissions.id-token: write is set in the workflow
}

# For local development/testing, you can use Azure CLI instead:
# provider "msgraph_entra" {
#   use_cli = true
# }

# AzureAD provider for user/group lookups (also supports OIDC)
provider "azuread" {
  # Uses the same ARM_* environment variables
}

# ============================================================================
# Example: Security Administrator Role Assignments
# ============================================================================

# Look up the Security Administrator role
data "msgraph_entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Define users who need Security Administrator eligible access
locals {
  security_admins = [
    {
      upn           = "security.lead@contoso.com"
      justification = "Security operations lead - incident response and security monitoring"
      duration      = "P365D"  # 1 year
    },
    {
      upn           = "security.analyst@contoso.com"
      justification = "SOC analyst - security investigations"
      duration      = "P180D"  # 6 months
    },
  ]
}

# Look up each user
data "azuread_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Create eligible role assignments using OIDC authentication
resource "msgraph_entra_directory_role_eligible_assignment" "security_admins" {
  for_each = { for admin in local.security_admins : admin.upn => admin }

  role_definition_id = data.msgraph_entra_directory_role.security_admin.template_id
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

# ============================================================================
# Example: Global Administrator (Break-Glass Accounts)
# ============================================================================

data "msgraph_entra_directory_role" "global_admin" {
  display_name = "Global Administrator"
}

locals {
  global_admins = [
    {
      upn           = "breakglass1@contoso.com"
      justification = "Break-glass emergency access account #1"
      duration      = "P365D"
    },
    {
      upn           = "breakglass2@contoso.com"
      justification = "Break-glass emergency access account #2"
      duration      = "P365D"
    },
  ]
}

data "azuread_user" "global_admins" {
  for_each            = { for admin in local.global_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

resource "msgraph_entra_directory_role_eligible_assignment" "global_admins" {
  for_each = { for admin in local.global_admins : admin.upn => admin }

  role_definition_id = data.msgraph_entra_directory_role.global_admin.template_id
  principal_id       = data.azuread_user.global_admins[each.key].id
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

# ============================================================================
# Outputs
# ============================================================================

output "security_admin_assignments" {
  description = "Security Administrator role assignments created via OIDC"
  value = {
    for k, v in msgraph_entra_directory_role_eligible_assignment.security_admins :
    k => {
      id          = v.id
      schedule_id = v.schedule_id
    }
  }
}

output "global_admin_assignments" {
  description = "Global Administrator role assignments created via OIDC"
  value = {
    for k, v in msgraph_entra_directory_role_eligible_assignment.global_admins :
    k => {
      id          = v.id
      schedule_id = v.schedule_id
    }
  }
}
