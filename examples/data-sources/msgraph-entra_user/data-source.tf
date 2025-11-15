# Example: Look up a user by their User Principal Name (UPN)
data "msgraph-entra_user" "example" {
  user_principal_name = "john.doe@contoso.com"
}

# Output the user's object ID
output "user_object_id" {
  value       = data.msgraph-entra_user.example.id
  description = "The object ID of the user in Azure AD"
}

# Output the user's display name
output "user_display_name" {
  value       = data.msgraph-entra_user.example.display_name
  description = "The display name of the user"
}

# Use the user's object ID for role assignments
resource "msgraph-entra_directory_role_eligible_assignment" "example" {
  role_definition_id = "62e90394-69f5-4237-9190-012177145e10" # Security Administrator
  principal_id       = data.msgraph-entra_user.example.id
  directory_scope_id = "/"
  justification      = "Example assignment using UPN lookup"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "2025-12-31T23:59:59Z"
    }
  }
}
