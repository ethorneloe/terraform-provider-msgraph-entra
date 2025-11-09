# Example 1: Basic eligible assignment with fixed expiration date
# Using afterDateTime provides strict enforcement - if someone manually changes
# the end date in the Azure Portal, Terraform will detect drift and revert it
# on the next apply to match your configured end_date_time.
resource "msgraph-entra_directory_role_eligible_assignment" "security_admin_basic" {
  role_definition_id = "62e90394-69f5-4237-9190-012177145e10" # Security Administrator
  principal_id       = "00000000-0000-0000-0000-000000000000" # Replace with actual user/group ID
  directory_scope_id = "/"
  justification      = "Security team member - incident response"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-12-31T23:59:59Z"
    }
  }
}

# Example 2: Eligible assignment with duration-based expiration (8 hours)
# IMPORTANT: Using afterDuration protects against configuration drift from manual changes.
# If someone modifies the assignment in the Azure Portal/Entra admin center,
# Terraform will preserve your configured type and duration values and won't
# try to revert the change. This is different from afterDateTime, which will
# cause Terraform to detect drift and revert manual portal changes on next apply.
resource "msgraph-entra_directory_role_eligible_assignment" "temp_admin" {
  role_definition_id = "62e90394-69f5-4237-9190-012177145e10" # Security Administrator
  principal_id       = "11111111-1111-1111-1111-111111111111" # Replace with actual user/group ID
  directory_scope_id = "/"
  justification      = "Temporary access for system maintenance"

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = "PT8H" # 8 hours in ISO 8601 duration format
    }
  }
}

# Example 3: Permanent eligible assignment (no expiration)
resource "msgraph-entra_directory_role_eligible_assignment" "global_admin_permanent" {
  role_definition_id = "62e90394-69f5-4237-9190-012177145e10" # Security Administrator
  principal_id       = "22222222-2222-2222-2222-222222222222" # Replace with actual user/group ID
  directory_scope_id = "/"
  justification      = "Permanent security team member"

  schedule_info {
    expiration {
      type = "noExpiration"
    }
  }
}

# Example 4: Using data sources to look up role and user
data "msgraph-entra_directory_role" "security_administrator" {
  display_name = "Security Administrator"
}

data "azuread_user" "security_analyst" {
  user_principal_name = "analyst@contoso.com"
}

resource "msgraph-entra_directory_role_eligible_assignment" "security_analyst" {
  role_definition_id = data.msgraph-entra_directory_role.security_administrator.template_id
  principal_id       = data.azuread_user.security_analyst.id
  directory_scope_id = "/"
  justification      = "Security analyst - requires elevated access for investigations"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-06-30T23:59:59Z"
    }
  }
}

# Outputs to view created assignments
output "basic_assignment_id" {
  value       = msgraph-entra_directory_role_eligible_assignment.security_admin_basic.schedule_id
  description = "The schedule ID of the basic assignment"
}

output "actual_start_time" {
  value       = msgraph-entra_directory_role_eligible_assignment.security_admin_basic.schedule_info[0].start_date_time
  description = "The actual start time set by Microsoft Graph (read-only)"
}
