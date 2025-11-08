# PowerShell script to install the provider locally
# Run this after building the provider

param(
    [string]$Version = "1.0.0",
    [string]$Namespace = "yourusername"
)

$ErrorActionPreference = "Stop"

# Determine OS and architecture
$OS = "windows"
$ARCH = "amd64"

if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $ARCH = "arm64"
}

# Build the provider first
Write-Host "Building provider..." -ForegroundColor Green
go build -o terraform-provider-msgraph-entra.exe

if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed!"
    exit 1
}

# Create plugin directory
$PluginDir = "$env:APPDATA\terraform.d\plugins\registry.terraform.io\$Namespace\msgraph_entra\$Version\${OS}_${ARCH}"

Write-Host "Creating plugin directory: $PluginDir" -ForegroundColor Green
New-Item -ItemType Directory -Force -Path $PluginDir | Out-Null

# Copy the provider binary
Write-Host "Copying provider binary..." -ForegroundColor Green
Copy-Item -Path ".\terraform-provider-msgraph-entra.exe" -Destination "$PluginDir\terraform-provider-msgraph_entra_v${Version}.exe" -Force

Write-Host "`n✅ Provider installed successfully!" -ForegroundColor Green
Write-Host "`nYou can now use it in your Terraform configuration:" -ForegroundColor Cyan
Write-Host @"

terraform {
  required_providers {
    msgraph_entra = {
      source  = "$Namespace/msgraph_entra"
      version = "~> $Version"
    }
  }
}

provider "msgraph_entra" {
  # Your configuration
}
"@

Write-Host "`nRun 'terraform init' in your project directory to use the provider." -ForegroundColor Yellow
