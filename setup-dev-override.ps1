# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# PowerShell script to set up development overrides
# This is the easiest way for local development

param(
    [string]$Namespace = "yourusername",
    [string]$ProviderPath = $PSScriptRoot
)

$ErrorActionPreference = "Stop"

# Build the provider first
Write-Host "Building provider..." -ForegroundColor Green
go build -o terraform-provider-msgraph-entra.exe

if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed!"
    exit 1
}

# Create .terraformrc content
$TerraformRC = @"
provider_installation {
  dev_overrides {
    "$Namespace/msgraph-entra" = "$($ProviderPath -replace '\\', '/')"
  }

  # For all other providers, install them directly as normal.
  direct {}
}
"@

# Determine terraformrc location
$TerraformRCPath = "$env:APPDATA\terraform.rc"

# Check if file exists
if (Test-Path $TerraformRCPath) {
    Write-Host "`nExisting .terraformrc found at: $TerraformRCPath" -ForegroundColor Yellow
    Write-Host "Contents:" -ForegroundColor Yellow
    Get-Content $TerraformRCPath
    Write-Host ""

    $Response = Read-Host "Do you want to overwrite it? (y/N)"
    if ($Response -ne 'y' -and $Response -ne 'Y') {
        Write-Host "`nPlease manually add the following to your .terraformrc:" -ForegroundColor Cyan
        Write-Host $TerraformRC
        exit 0
    }
}

# Write the file
Write-Host "`nWriting .terraformrc to: $TerraformRCPath" -ForegroundColor Green
$TerraformRC | Out-File -FilePath $TerraformRCPath -Encoding UTF8

Write-Host "`n✅ Development override configured!" -ForegroundColor Green
Write-Host "`nYou can now use the provider in your Terraform configuration:" -ForegroundColor Cyan
Write-Host @"

terraform {
  required_providers {
    msgraph-entra = {
      source  = "$Namespace/msgraph-entra"
      version = "~> 1.0"  # Version is ignored with dev_overrides
    }
  }
}

provider "msgraph-entra" {
  # Your configuration
}
"@

Write-Host "`n⚠️  Note: You will see a warning about development overrides when running Terraform. This is normal." -ForegroundColor Yellow
Write-Host "To remove the override, delete: $TerraformRCPath`n" -ForegroundColor Yellow
