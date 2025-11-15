# Complete Example: Managing Multiple Roles with Scalable Pattern
# This example demonstrates a production-ready structure for managing
# Entra ID role eligible assignments across multiple roles

terraform {
  required_providers {
    msgraph-entra = {
      source = "registry.terraform.io/ethorneloe/msgraph-entra"
    }
  }
}

# Configure the Entra provider
provider "msgraph-entra" {
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
# Note: High-privilege roles like Global Administrator and Privileged Role Administrator
# are intentionally excluded and should be managed manually
locals {
  # Define all role assignments in a structured format
  role_assignments = {
    "Security Administrator" = [
      {
        upn           = "security.lead@contoso.com"
        justification = "Security operations lead"
        end_date      = "2025-12-31T23:59:59Z"
      },
      {
        upn           = "security.analyst1@contoso.com"
        justification = "SOC analyst - on-call rotation"
        end_date      = "2025-06-30T23:59:59Z"
      },
      {
        upn           = "security.analyst2@contoso.com"
        justification = "SOC analyst - on-call rotation"
        end_date      = "2025-06-30T23:59:59Z"
      },
    ]

    "User Administrator" = [
      {
        upn           = "helpdesk.lead@contoso.com"
        justification = "Help desk user management"
        end_date      = "2025-12-31T23:59:59Z"
      },
      {
        upn           = "hr.admin@contoso.com"
        justification = "HR onboarding/offboarding"
        end_date      = "2025-12-31T23:59:59Z"
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
        end_date      = assignment.end_date
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
data "msgraph-entra_user" "users" {
  for_each            = toset([for assignment in local.all_assignments : assignment.upn])
  user_principal_name = each.key
}

# Create all eligible role assignments
resource "msgraph-entra_directory_role_eligible_assignment" "assignments" {
  for_each = { for assignment in local.all_assignments : assignment.key => assignment }

  role_definition_id = data.msgraph-entra_directory_role.roles[each.value.role_name].template_id
  principal_id       = data.msgraph-entra_user.users[each.value.upn].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type     = "afterDateTime"
      end_date = each.value.end_date
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
