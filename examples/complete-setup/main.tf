# Complete Example: Managing Multiple Roles with Scalable Pattern
# This example demonstrates a production-ready structure for managing
# Entra ID role eligible assignments across multiple roles

terraform {
  required_providers {
    msgraph-entra = {
      source = "yourusername/msgraph-entra"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.0"
    }
  }
}

# Configure the Entra provider
provider "msgraph-entra" {
  tenant_id     = var.tenant_id
  client_id     = var.client_id
  client_secret = var.client_secret
}

# Configure AzureAD provider (for user lookups)
provider "azuread" {
  tenant_id     = var.tenant_id
  client_id     = var.client_id
  client_secret = var.client_secret
}

# Variables
variable "tenant_id" {
  description = "Azure AD Tenant ID"
  type        = string
}

variable "client_id" {
  description = "Service Principal Client ID"
  type        = string
}

variable "client_secret" {
  description = "Service Principal Client Secret"
  type        = string
  sensitive   = true
}

# Central configuration for all role assignments
# This approach makes it easy to see all assignments at a glance
locals {
  # Define all role assignments in a structured format
  role_assignments = {
    "Global Administrator" = [
      {
        upn           = "breakglass1@contoso.com"
        justification = "Break-glass emergency access"
        duration      = "P365D"
      },
      {
        upn           = "breakglass2@contoso.com"
        justification = "Break-glass emergency access"
        duration      = "P365D"
      },
    ]

    "Security Administrator" = [
      {
        upn           = "security.lead@contoso.com"
        justification = "Security operations lead"
        duration      = "P365D"
      },
      {
        upn           = "security.analyst1@contoso.com"
        justification = "SOC analyst - on-call rotation"
        duration      = "P180D"
      },
      {
        upn           = "security.analyst2@contoso.com"
        justification = "SOC analyst - on-call rotation"
        duration      = "P180D"
      },
    ]

    "User Administrator" = [
      {
        upn           = "helpdesk.lead@contoso.com"
        justification = "Help desk user management"
        duration      = "P180D"
      },
      {
        upn           = "hr.admin@contoso.com"
        justification = "HR onboarding/offboarding"
        duration      = "P365D"
      },
    ]

    "Privileged Role Administrator" = [
      {
        upn           = "identity.admin@contoso.com"
        justification = "Identity and access management"
        duration      = "P365D"
      },
    ]
  }

  # Flatten the structure for easier processing
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

# Look up all directory roles
data "msgraph-entra_directory_role" "roles" {
  for_each     = toset(keys(local.role_assignments))
  display_name = each.key
}

# Look up all users
data "azuread_user" "users" {
  for_each            = toset([for assignment in local.all_assignments : assignment.upn])
  user_principal_name = each.key
}

# Create all eligible role assignments
resource "msgraph-entra_directory_role_eligible_assignment" "assignments" {
  for_each = { for assignment in local.all_assignments : assignment.key => assignment }

  role_definition_id = data.msgraph-entra_directory_role.roles[each.value.role_name].template_id
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

# Outputs
output "role_assignment_summary" {
  value = {
    for role_name in keys(local.role_assignments) :
    role_name => [
      for assignment in local.role_assignments[role_name] :
      assignment.upn
    ]
  }
  description = "Summary of all role assignments by role"
}

output "user_assignment_summary" {
  value = {
    for user_upn in toset([for assignment in local.all_assignments : assignment.upn]) :
    user_upn => [
      for assignment in local.all_assignments :
      assignment.role_name if assignment.upn == user_upn
    ]
  }
  description = "Summary of all role assignments by user"
}
