# Advanced Example: Managing Multiple Roles with Different Expiration Strategies
#
# This example demonstrates how to manage eligible role assignments for multiple
# directory roles with different expiration strategies:
# - Security Administrator: Fixed date expiration
# - Global Administrator: Duration-based expiration (shorter access windows)
# - User Administrator: Permanent access (no expiration)

terraform {
  required_providers {
    msgraph-entra = {
      source = "registry.terraform.io/ethorneloe/msgraph-entra"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.0"
    }
  }
}

# Look up directory roles using data sources
data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

data "msgraph-entra_directory_role" "global_administrator" {
  display_name = "Global Administrator"
}

data "msgraph-entra_directory_role" "user_administrator" {
  display_name = "User Administrator"
}

# Define users and their role assignments with different schedules
locals {
  # Security Administrators: Fixed end dates for annual reviews
  security_admins = [
    {
      upn           = "security-lead@contoso.com"
      justification = "Security team lead - annual review cycle"
      end_date      = "2025-12-31T23:59:59Z"
    },
    {
      upn           = "soc-analyst@contoso.com"
      justification = "SOC analyst - 6-month rotation"
      end_date      = "2025-06-30T23:59:59Z"
    },
  ]

  # Global Administrators: Duration-based for tight control (24 hours)
  global_admins = [
    {
      upn           = "emergency-admin@contoso.com"
      justification = "Break-glass account - 24h activation window"
      duration      = "PT24H" # 24 hours
    },
    {
      upn           = "compliance-officer@contoso.com"
      justification = "Quarterly compliance reviews"
      duration      = "P90D" # 90 days
    },
  ]

  # User Administrators: Permanent access for help desk team
  user_admins = [
    {
      upn           = "helpdesk-lead@contoso.com"
      justification = "Help desk team lead - permanent access required"
    },
    {
      upn           = "helpdesk-senior@contoso.com"
      justification = "Senior help desk technician - permanent access"
    },
  ]
}

# Look up all users
data "azuread_user" "security_admins" {
  for_each            = { for admin in local.security_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

data "azuread_user" "global_admins" {
  for_each            = { for admin in local.global_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

data "azuread_user" "user_admins" {
  for_each            = { for admin in local.user_admins : admin.upn => admin }
  user_principal_name = each.value.upn
}

# Security Administrator Assignments (afterDateTime)
resource "msgraph-entra_directory_role_eligible_assignment" "security_admins" {
  for_each = { for admin in local.security_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.security_administrator.template_id
  principal_id       = data.azuread_user.security_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = each.value.end_date
    }
  }
}

# Global Administrator Assignments (afterDuration)
resource "msgraph-entra_directory_role_eligible_assignment" "global_admins" {
  for_each = { for admin in local.global_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.global_administrator.template_id
  principal_id       = data.azuread_user.global_admins[each.key].id
  directory_scope_id = "/"
  justification      = each.value.justification

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = each.value.duration
    }
  }
}

# User Administrator Assignments (noExpiration)
resource "msgraph-entra_directory_role_eligible_assignment" "user_admins" {
  for_each = { for admin in local.user_admins : admin.upn => admin }

  role_definition_id = data.msgraph-entra_directory_role.user_administrator.template_id
  principal_id       = data.azuread_user.user_admins[each.key].id
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
      schedule_id    = v.schedule_id
      actual_start   = v.schedule_info[0].start_date_time
      expiration     = v.schedule_info[0].expiration[0].end_date_time
      type          = v.schedule_info[0].expiration[0].type
    }
  }
  description = "Security Administrator eligible assignments with fixed end dates"
}

output "global_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.global_admins :
    k => {
      schedule_id    = v.schedule_id
      actual_start   = v.schedule_info[0].start_date_time
      duration       = v.schedule_info[0].expiration[0].duration
      computed_end   = v.schedule_info[0].expiration[0].end_date_time
      type          = v.schedule_info[0].expiration[0].type
    }
  }
  description = "Global Administrator eligible assignments with duration-based expiration"
}

output "user_admin_assignments" {
  value = {
    for k, v in msgraph-entra_directory_role_eligible_assignment.user_admins :
    k => {
      schedule_id  = v.schedule_id
      actual_start = v.schedule_info[0].start_date_time
      type        = v.schedule_info[0].expiration[0].type
    }
  }
  description = "User Administrator eligible assignments with no expiration"
}

# Summary output showing all assignments by role
output "summary" {
  value = {
    security_administrator = length(local.security_admins)
    global_administrator   = length(local.global_admins)
    user_administrator     = length(local.user_admins)
    total_assignments      = length(local.security_admins) + length(local.global_admins) + length(local.user_admins)
  }
  description = "Summary of eligible role assignments by role type"
}
