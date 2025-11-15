# User Administrator Role - Eligible Assignments
# This file manages all eligible assignments for the User Administrator role

# Data source to look up the User Administrator role
data "msgraph-entra_directory_role" "user_administrator" {
  display_name = "User Administrator"
}

# Define the list of users who should have eligible access to User Administrator role
locals {
  user_admins = [
    {
      upn           = "helpdesk.lead@contoso.com"
      justification = "Help desk lead - user account management"
      duration      = "P180D" # 180 days (6 months)
    },
    {
      upn           = "hr.admin@contoso.com"
      justification = "HR admin - onboarding/offboarding coordination"
      duration      = "P365D" # 1 year
    },
  ]
}

# Look up each user by UPN using the native msgraph-entra_user data source
data "msgraph-entra_user" "user_admins" {
  for_each            = { for admin in local.user_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Create eligible role assignments for each user
resource "msgraph-entra_directory_role_eligible_assignment" "user_admins" {
  for_each = { for admin in local.user_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.user_administrator.template_id
  principal_id       = data.msgraph-entra_user.user_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = each.value.duration
    }
  }
}

# Output the created assignments for verification
output "user_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.user_admins :
    k => {
      id          = v.id
      schedule_id = v.schedule_id
    }
  }
  description = "Map of User Administrator eligible role assignments"
}
