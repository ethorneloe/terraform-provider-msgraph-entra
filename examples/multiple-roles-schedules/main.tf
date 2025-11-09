# Advanced Example: Managing Multiple Roles with Different Expiration Strategies
#
# This example demonstrates how to manage eligible role assignments for multiple
# directory roles with different expiration strategies:
# - Security Administrator: Fixed date expiration
# - User Administrator: Permanent access (no expiration)

terraform {
  required_providers {
    msgraph-entra = {
      source = "registry.terraform.io/ethorneloe/msgraph-entra"
    }
  }
}

# Look up directory roles using data sources
data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

data "msgraph-entra_directory_role" "user_administrator" {
  display_name = "User Administrator"
}

# Define users and their role assignments with different schedules
# Users are looked up by their UPN (email address) - no need to manually find object IDs!
locals {
  # Security Administrators: Fixed end dates for annual reviews
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

  # User Administrators: Permanent access for help desk team
  user_admins = [
    {
      upn           = "helpdesk.lead@contoso.com"
      justification = "Help desk team lead - permanent access required"
    },
    {
      upn           = "helpdesk.senior@contoso.com"
      justification = "Senior help desk technician - permanent access"
    },
  ]
}

# Look up users by UPN for each role
data "msgraph-entra_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

data "msgraph-entra_user" "user_admins" {
  for_each            = { for admin in local.user_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Security Administrator Assignments (afterDateTime)
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

# User Administrator Assignments (noExpiration)
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

# Outputs for monitoring and verification
output "security_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.security_admins :
    k => {
      schedule_id  = v.schedule_id
      actual_start = v.schedule_info[0].start_date_time
      expiration   = v.schedule_info[0].expiration[0].end_date_time
      type         = v.schedule_info[0].expiration[0].type
    }
  }
  description = "Security Administrator eligible assignments with fixed end dates"
}

output "user_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.user_admins :
    k => {
      schedule_id  = v.schedule_id
      actual_start = v.schedule_info[0].start_date_time
      type         = v.schedule_info[0].expiration[0].type
    }
  }
  description = "User Administrator eligible assignments with no expiration"
}

# Summary output showing all assignments by role
output "summary" {
  value = {
    security_administrator = length(local.security_admins)
    user_administrator     = length(local.user_admins)
    total_assignments      = length(local.security_admins) + length(local.user_admins)
  }
  description = "Summary of eligible role assignments by role type"
}
