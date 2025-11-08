# Global Administrator Role - Eligible Assignments
# This file manages all eligible assignments for the Global Administrator role
# Use sparingly - this is the highest privilege level in Entra ID

# Data source to look up the Global Administrator role
data "msgraph_entra_directory_role" "global_administrator" {
  display_name = "Global Administrator"
}

# Define the list of users who should have eligible access to Global Administrator role
# Keep this list VERY small - only for break-glass and emergency scenarios
locals {
  global_admins = [
    {
      upn           = "admin.breakglass1@contoso.com"
      justification = "Break-glass account #1 - emergency access only"
      start_date    = "2025-01-08T00:00:00Z"
      duration      = "P365D"  # 365 days
    },
    {
      upn           = "admin.breakglass2@contoso.com"
      justification = "Break-glass account #2 - emergency access only"
      start_date    = "2025-01-08T00:00:00Z"
      duration      = "P365D"  # 365 days
    },
  ]
}

# Look up each user by UPN
data "azuread_user" "global_admins" {
  for_each            = { for admin in local.global_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Create eligible role assignments for each user
resource "msgraph_entra_directory_role_eligible_assignment" "global_admins" {
  for_each = { for admin in local.global_admins : admin.upn => admin }

  role_definition_id = data.msgraph_entra_directory_role.global_administrator.template_id
  principal_id       = data.azuread_user.global_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    start_date_time = each.value.start_date

    expiration {
      type     = "afterDuration"
      duration = each.value.duration
    }
  }
}

# Output the created assignments for verification
output "global_admin_assignments" {
  value = {
    for k, v in entra_directory_role_eligible_assignment.global_admins :
    k => {
      id          = v.id
      schedule_id = v.schedule_id
    }
  }
  description = "Map of Global Administrator eligible role assignments"
}
