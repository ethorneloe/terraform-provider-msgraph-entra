# Quick Start Guide - msgraph_entra Provider

## 🚀 Fastest Way to Get Started (Local Development)

### 1. Set Up Development Override (One-Time Setup)

**Windows:**
```powershell
cd c:\Dev\terraform-provider-scaffolding-framework
.\setup-dev-override.ps1
```

**Linux/Mac:**
```bash
cd /path/to/terraform-provider-scaffolding-framework
chmod +x setup-dev-override.sh
./setup-dev-override.sh
```

This creates `~/.terraformrc` (or `%APPDATA%\terraform.rc` on Windows) pointing to your local build.

### 2. Create a Test Project

```bash
mkdir ~/entra-pim-test
cd ~/entra-pim-test
```

Create `main.tf`:
```hcl
terraform {
  required_providers {
    msgraph_entra = {
      source = "yourusername/msgraph_entra"
    }
  }
}

# Use Azure CLI for authentication (easiest for testing)
provider "msgraph_entra" {
  use_cli = true
}

# Look up a directory role
data "msgraph_entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Output the role template ID
output "security_admin_template_id" {
  value = data.msgraph_entra_directory_role.security_admin.template_id
}
```

### 3. Authenticate with Azure CLI

```bash
az login
az account set --subscription "your-subscription-name"
```

### 4. Run Terraform

```bash
terraform init
terraform plan
```

You should see the Security Administrator role template ID! 🎉

---

## 📋 Creating Your First Eligible Role Assignment

### Example: Security Administrator for a User

Create `security-admins.tf`:

```hcl
# Data source for the role
data "msgraph_entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

# Data source for the user (using azuread provider)
data "azuread_user" "john" {
  user_principal_name = "john.doe@yourcompany.com"
}

# Create eligible assignment
resource "msgraph_entra_directory_role_eligible_assignment" "john_security_admin" {
  role_definition_id = data.msgraph_entra_directory_role.security_admin.template_id
  principal_id       = data.azuread_user.john.id
  directory_scope_id = "/"
  justification      = "Security team member - incident response"

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"

    expiration {
      type     = "afterDuration"
      duration = "P365D"  # 1 year
    }
  }
}
```

Run:
```bash
terraform plan
terraform apply
```

John now has an **eligible** Security Administrator role that he can activate in the Azure portal!

---

## 🔐 For GitHub Actions (Production)

### 1. Set Up OIDC in Azure

```bash
# Create federated credential for your GitHub repo
az ad app federated-credential create \
  --id YOUR_APP_ID \
  --parameters '{
    "name": "github-actions-main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yourorg/yourrepo:ref:refs/heads/main",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

### 2. Add GitHub Secrets

In your GitHub repo settings, add:
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`

### 3. Create GitHub Workflow

`.github/workflows/terraform.yml`:
```yaml
name: Terraform

on:
  push:
    branches: [main]

permissions:
  id-token: write
  contents: read

jobs:
  terraform:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: hashicorp/setup-terraform@v3

      - name: Terraform Init
        run: terraform init
        env:
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: true

      - name: Terraform Apply
        run: terraform apply -auto-approve
        env:
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: true
```

**No secrets stored in GitHub!** 🔒

---

## 📝 Common Patterns

### Pattern 1: Multiple Users, One Role

```hcl
locals {
  security_admins = [
    "john.doe@company.com",
    "jane.smith@company.com",
    "bob.jones@company.com",
  ]
}

data "msgraph_entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

data "azuread_user" "security_admins" {
  for_each            = toset(local.security_admins)
  user_principal_name = each.value
}

resource "msgraph_entra_directory_role_eligible_assignment" "security_admins" {
  for_each = toset(local.security_admins)

  role_definition_id = data.msgraph_entra_directory_role.security_admin.template_id
  principal_id       = data.azuread_user.security_admins[each.key].id
  directory_scope_id = "/"
  justification      = "Security operations team"

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"
    expiration {
      type     = "afterDuration"
      duration = "P180D"  # 6 months
    }
  }
}
```

### Pattern 2: Role-Assignable Group

```hcl
# Using hashicorp/azuread provider
resource "azuread_group" "security_admins_eligible" {
  display_name          = "Security-Admins-Eligible"
  security_enabled      = true
  assignable_to_role    = true
}

# Assign the role to the group (eligible)
resource "msgraph_entra_directory_role_eligible_assignment" "group_assignment" {
  role_definition_id = data.msgraph_entra_directory_role.security_admin.template_id
  principal_id       = azuread_group.security_admins_eligible.id
  directory_scope_id = "/"
  justification      = "Security operations team group"

  schedule_info {
    start_date_time = "2025-01-08T00:00:00Z"
    expiration {
      type = "noExpiration"
    }
  }
}

# Users in this group can activate the Security Admin role
```

---

## 🎯 Available Directory Roles

Common roles you can assign:
- `Global Administrator`
- `Security Administrator`
- `User Administrator`
- `Privileged Role Administrator`
- `Application Administrator`
- `Cloud Application Administrator`
- `Groups Administrator`
- `Helpdesk Administrator`
- `Authentication Administrator`
- `Conditional Access Administrator`

---

## 🔑 Required Permissions

Your app registration needs:
- ✅ `RoleManagement.ReadWrite.Directory` (Application)
- ✅ `Directory.Read.All` (Application)

Grant admin consent after adding these permissions.

---

## 📚 More Examples

Check the `examples/` directory:
- `examples/security-administrator/` - Per-role pattern
- `examples/global-administrator/` - Break-glass accounts
- `examples/complete-setup/` - Centralized multi-role config

---

## 🆘 Troubleshooting

**"Provider not found"**
→ Run `.\setup-dev-override.ps1` or check `.terraformrc`

**"Permission denied"**
→ Grant `RoleManagement.ReadWrite.Directory` and admin consent

**"Role not found"**
→ Check exact display name (case-sensitive)

**"Invalid duration"**
→ Use ISO 8601: `PT8H` (8 hours), `P30D` (30 days), `P365D` (1 year)

---

## 🚀 You're Ready!

You now have:
- ✅ Local provider set up
- ✅ Example configuration
- ✅ Authentication working
- ✅ PIM as code! 🎉

Start managing your Entra ID role assignments with Terraform!
