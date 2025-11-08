#!/bin/bash
# Bash script to install the provider locally
# Run this after building the provider

set -e

VERSION="${1:-1.0.0}"
NAMESPACE="${2:-yourusername}"

# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "Building provider..."
go build -o terraform-provider-msgraph-entra

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Create plugin directory
PLUGIN_DIR="$HOME/.terraform.d/plugins/registry.terraform.io/$NAMESPACE/msgraph_entra/$VERSION/${OS}_${ARCH}"

echo "Creating plugin directory: $PLUGIN_DIR"
mkdir -p "$PLUGIN_DIR"

# Copy the provider binary
echo "Copying provider binary..."
cp terraform-provider-msgraph-entra "$PLUGIN_DIR/terraform-provider-msgraph_entra_v${VERSION}"
chmod +x "$PLUGIN_DIR/terraform-provider-msgraph_entra_v${VERSION}"

echo ""
echo "✅ Provider installed successfully!"
echo ""
echo "You can now use it in your Terraform configuration:"
cat <<EOF

terraform {
  required_providers {
    msgraph_entra = {
      source  = "$NAMESPACE/msgraph_entra"
      version = "~> $VERSION"
    }
  }
}

provider "msgraph_entra" {
  # Your configuration
}
EOF

echo ""
echo "Run 'terraform init' in your project directory to use the provider."
