# Using the Provider Locally (Without Publishing)

You can use this provider without publishing it to the Terraform Registry. Here are three methods:

## Method 1: Development Overrides (Easiest for Testing)

**Recommended for:** Local development and testing

### Quick Setup (Windows)

```powershell
# Run this script from the provider directory
.\setup-dev-override.ps1
```

This script will:
1. Build the provider
2. Create/update `%APPDATA%\terraform.rc` with development overrides
3. Configure Terraform to use your local provider binary

### Manual Setup

1. **Build the provider:**
   ```powershell
   go build -o terraform-provider-msgraph-entra.exe
   ```

2. **Create `.terraformrc` file** at `%APPDATA%\terraform.rc` (Windows) or `~/.terraformrc` (Linux/Mac):
   ```hcl
   provider_installation {
     dev_overrides {
       "yourusername/msgraph_entra" = "C:/Dev/terraform-provider-scaffolding-framework"
     }

     # For all other providers, install them directly as normal.
     direct {}
   }
   ```

3. **Use in your Terraform config:**
   ```hcl
   terraform {
     required_providers {
       msgraph_entra = {
         source = "yourusername/msgraph_entra"
         # Version is ignored with dev_overrides
       }
     }
   }

   provider "msgraph_entra" {
     # Your configuration
   }
   ```

4. **Run Terraform:**
   ```bash
   terraform init
   terraform plan
   ```

**Note:** You'll see this warning (it's expected and can be ignored):
```
Warning: Provider development overrides are in effect
```

### Advantages
- ✅ Easiest to set up
- ✅ No need to rebuild/reinstall after code changes
- ✅ Works across all Terraform projects
- ✅ Great for rapid development

### Disadvantages
- ⚠️ Shows warning on every Terraform run
- ⚠️ Affects ALL Terraform projects on your machine

---

## Method 2: Local Plugin Directory (Production-Like)

**Recommended for:** Testing before publishing, CI/CD pipelines

### Quick Setup (Windows)

```powershell
# Run this script from the provider directory
.\install-local.ps1
```

### Manual Setup

1. **Build the provider:**
   ```powershell
   go build -o terraform-provider-msgraph-entra.exe
   ```

2. **Create the plugin directory structure:**

   **Windows:**
   ```powershell
   $PluginDir = "$env:APPDATA\terraform.d\plugins\registry.terraform.io\yourusername\msgraph_entra\1.0.0\windows_amd64"
   New-Item -ItemType Directory -Force -Path $PluginDir
   ```

   **Linux:**
   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/yourusername/msgraph_entra/1.0.0/linux_amd64
   ```

   **Mac (Intel):**
   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/yourusername/msgraph_entra/1.0.0/darwin_amd64
   ```

   **Mac (Apple Silicon):**
   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/yourusername/msgraph_entra/1.0.0/darwin_arm64
   ```

3. **Copy the provider binary:**

   **Windows:**
   ```powershell
   Copy-Item terraform-provider-msgraph-entra.exe "$PluginDir\terraform-provider-msgraph_entra_v1.0.0.exe"
   ```

   **Linux/Mac:**
   ```bash
   cp terraform-provider-msgraph-entra ~/.terraform.d/plugins/registry.terraform.io/yourusername/msgraph_entra/1.0.0/linux_amd64/terraform-provider-msgraph_entra_v1.0.0
   chmod +x ~/.terraform.d/plugins/registry.terraform.io/yourusername/msgraph_entra/1.0.0/linux_amd64/terraform-provider-msgraph_entra_v1.0.0
   ```

4. **Use in your Terraform config:**
   ```hcl
   terraform {
     required_providers {
       msgraph_entra = {
         source  = "yourusername/msgraph_entra"
         version = "~> 1.0"
       }
     }
   }

   provider "msgraph_entra" {
     # Your configuration
   }
   ```

5. **Run Terraform:**
   ```bash
   terraform init
   terraform plan
   ```

### Advantages
- ✅ No warnings
- ✅ Version-specific (can have multiple versions)
- ✅ More production-like
- ✅ Per-project control

### Disadvantages
- ⚠️ Need to reinstall after every code change
- ⚠️ More complex directory structure

---

## Method 3: Filesystem Mirror (For Teams)

**Recommended for:** Sharing the provider with your team without a registry

### Setup

1. **Create a shared directory** (network share, git repository, etc.):
   ```
   /shared/terraform-providers/
   └── registry.terraform.io/
       └── yourusername/
           └── msgraph_entra/
               └── 1.0.0/
                   ├── windows_amd64/
                   │   └── terraform-provider-msgraph_entra_v1.0.0.exe
                   ├── linux_amd64/
                   │   └── terraform-provider-msgraph_entra_v1.0.0
                   └── darwin_amd64/
                       └── terraform-provider-msgraph_entra_v1.0.0
   ```

2. **Configure each team member's `.terraformrc`:**
   ```hcl
   provider_installation {
     filesystem_mirror {
       path    = "/shared/terraform-providers"
       include = ["yourusername/msgraph_entra"]
     }

     direct {
       exclude = ["yourusername/msgraph_entra"]
     }
   }
   ```

3. **Use normally in Terraform:**
   ```hcl
   terraform {
     required_providers {
       msgraph_entra = {
         source  = "yourusername/msgraph_entra"
         version = "~> 1.0"
       }
     }
   }
   ```

### Advantages
- ✅ Team can share the same provider version
- ✅ No individual setup needed after initial config
- ✅ Works well for air-gapped environments

### Disadvantages
- ⚠️ Requires shared infrastructure
- ⚠️ Need to manage multiple OS/architecture builds

---

## Method 4: Implied Local Mirror (Simplest for Quick Tests)

**Recommended for:** One-off tests, quick experiments

### Setup

1. **Build the provider:**
   ```bash
   go build -o terraform-provider-msgraph-entra.exe
   ```

2. **Place the binary in your Terraform project directory:**
   ```
   my-terraform-project/
   ├── main.tf
   └── terraform-provider-msgraph-entra.exe
   ```

3. **Configure Terraform to use local provider:**
   ```hcl
   terraform {
     required_providers {
       msgraph_entra = {
         source  = "terraform.local/local/msgraph_entra"
         version = "1.0.0"
       }
     }
   }

   provider "msgraph_entra" {
     # Your configuration
   }
   ```

4. **Create the local plugin structure:**
   ```powershell
   # Windows
   mkdir -p .terraform\providers\terraform.local\local\msgraph_entra\1.0.0\windows_amd64
   copy terraform-provider-msgraph-entra.exe .terraform\providers\terraform.local\local\msgraph_entra\1.0.0\windows_amd64\
   ```

   ```bash
   # Linux/Mac
   mkdir -p .terraform/providers/terraform.local/local/msgraph_entra/1.0.0/linux_amd64
   cp terraform-provider-msgraph-entra .terraform/providers/terraform.local/local/msgraph_entra/1.0.0/linux_amd64/
   ```

---

## Recommendation

**For local development:** Use **Method 1** (Development Overrides)
- Fastest iteration cycle
- No reinstall needed after code changes
- Just rebuild and run `terraform plan`

**For team testing before publishing:** Use **Method 2** (Local Plugin Directory)
- More production-like
- No warnings
- Version control

**For CI/CD or automation:** Use **Method 2** with the install script
- Predictable and repeatable
- Can be automated in your pipeline

---

## Testing Your Setup

Create a test Terraform file `test.tf`:

```hcl
terraform {
  required_providers {
    msgraph_entra = {
      source = "yourusername/msgraph_entra"
    }
  }
}

provider "msgraph_entra" {
  use_cli = true  # Use Azure CLI for testing
}

data "msgraph_entra_directory_role" "test" {
  display_name = "Global Administrator"
}

output "role_template_id" {
  value = data.msgraph_entra_directory_role.test.template_id
}
```

Run:
```bash
terraform init
terraform plan
```

If you see the role template ID in the output, the provider is working! 🎉

---

## Troubleshooting

### "Provider not found"
- Check your `.terraformrc` file location and syntax
- Ensure the provider binary exists at the specified path
- Run `terraform init -upgrade`

### "Failed to install provider"
- Verify the directory structure matches exactly
- Check file permissions (Linux/Mac: `chmod +x`)
- Ensure the binary name includes the version: `terraform-provider-msgraph_entra_v1.0.0.exe`

### "Development overrides warning"
- This is normal with Method 1 and can be ignored
- To remove: delete your `.terraformrc` file

---

## Next Steps

Once you're happy with the provider:
1. Publish to Terraform Registry (optional)
2. Use in production with GitHub Actions + OIDC
3. Manage your Entra PIM roles as code! 🚀
