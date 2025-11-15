# Security Administrator Role - Eligible Assignments
# This file manages all eligible assignments for the Security Administrator role

# Data source to look up the Security Administrator role
data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

# Define the list of users who should have eligible access to Security Administrator role
locals {
  security_admins = [
    {
      upn           = "john.doe@contoso.com"
      justification = "Security team lead - requires elevated access for incident response"
      end_date      = "2025-12-31T23:59:59Z"
    },
    {
      upn           = "jane.smith@contoso.com"
      justification = "Security analyst - on-call rotation"
      end_date      = "2025-06-30T23:59:59Z"
    },
    {
      upn           = "bob.johnson@contoso.com"
      justification = "SOC manager - operational oversight"
      end_date      = "2026-01-08T00:00:00Z"
    },
  ]
}

# Look up each user by UPN using the native msgraph-entra_user data source
data "msgraph-entra_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Create eligible role assignments for each user
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

# Output the created assignments for verification
output "security_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.security_admins :
    k => {
      id          = v.id
      schedule_id = v.schedule_id
    }
  }
  description = "Map of Security Administrator eligible role assignments"
}
