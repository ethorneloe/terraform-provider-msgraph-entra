# Provider Naming Suggestions

## Problem
The name "entra" is too generic and could be taken by Microsoft in the future.

## Suggested Alternative Names

### Option 1: **terraform-provider-entrapim** ⭐ RECOMMENDED
- **Pros:**
  - Clearly indicates PIM focus
  - Unlikely to conflict with Microsoft
  - Short and memorable
  - Domain-specific
- **Cons:**
  - Might limit perceived scope
- **Usage:** `provider "entrapim"`

### Option 2: **terraform-provider-msgraph-pim**
- **Pros:**
  - Indicates it uses Microsoft Graph
  - Clear PIM focus
  - Won't conflict with Microsoft's msgraph provider
- **Cons:**
  - Longer name
  - Hyphenated
- **Usage:** `provider "msgraph-pim"` or `provider "msgraph_pim"`

### Option 3: **terraform-provider-entraid-pim**
- **Pros:**
  - Uses "Entra ID" which is the full product name
  - Clear PIM focus
  - Professional naming
- **Cons:**
  - Longer name
  - Hyphenated
- **Usage:** `provider "entraid-pim"` or `provider "entraid_pim"`

### Option 4: **terraform-provider-azuread-pim**
- **Pros:**
  - Uses familiar "Azure AD" naming
  - Clear differentiation from hashicorp/azuread
  - Many people still search for "Azure AD" not "Entra"
- **Cons:**
  - Azure AD is the old branding
  - Could confuse with hashicorp/azuread provider
- **Usage:** `provider "azuread-pim"` or `provider "azuread_pim"`

### Option 5: **terraform-provider-entragov**
- **Pros:**
  - "Governance" is broader than just PIM
  - Allows future expansion (access packages, entitlement management)
  - Professional naming
- **Cons:**
  - Less obvious what it does
- **Usage:** `provider "entragov"`

### Option 6: **terraform-provider-graphpim**
- **Pros:**
  - Short and clear
  - Indicates Graph API usage
  - PIM focus
- **Cons:**
  - Might be confused with general Graph access
- **Usage:** `provider "graphpim"`

## Recommendation

**Go with `terraform-provider-entrapim`** for these reasons:

1. ✅ **Clear Purpose**: Name immediately tells users it's for PIM
2. ✅ **Unlikely Conflict**: Microsoft unlikely to create "entrapim" provider
3. ✅ **Memorable**: Short, single word, easy to type
4. ✅ **Room to Grow**: Can add non-PIM features later without name being misleading
5. ✅ **SEO Friendly**: People searching "entra pim terraform" will find it

## Registry Names

For Terraform Registry, you'd publish as:
- `yourusername/entrapim`
- Or with your organization: `yourorg/entrapim`

Example usage:
```hcl
terraform {
  required_providers {
    entrapim = {
      source  = "yourusername/entrapim"
      version = "~> 1.0"
    }
  }
}

provider "entrapim" {
  tenant_id = var.tenant_id
  client_id = var.client_id
  # OIDC authentication for GitHub Actions
}
```

## Implementation Steps to Rename

1. Update `go.mod`:
   ```
   module github.com/yourusername/terraform-provider-entrapim
   ```

2. Update `main.go`:
   ```go
   Address: "registry.terraform.io/yourusername/entrapim",
   ```

3. Update `provider.go`:
   ```go
   resp.TypeName = "entrapim"
   ```

4. Update all documentation and examples

5. Rebuild:
   ```bash
   go build -o terraform-provider-entrapim.exe
   ```

Would you like me to make these changes now?
