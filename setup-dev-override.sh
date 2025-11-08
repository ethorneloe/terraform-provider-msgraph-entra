#!/bin/bash
# Bash script to set up development overrides
# This is the easiest way for local development

set -e

NAMESPACE="${1:-yourusername}"
PROVIDER_PATH="${2:-$(pwd)}"

echo "Building provider..."
go build -o terraform-provider-msgraph-entra

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Create .terraformrc content
TERRAFORMRC_CONTENT="provider_installation {
  dev_overrides {
    \"$NAMESPACE/msgraph_entra\" = \"$PROVIDER_PATH\"
  }

  # For all other providers, install them directly as normal.
  direct {}
}
"

# Determine terraformrc location
TERRAFORMRC_PATH="$HOME/.terraformrc"

# Check if file exists
if [ -f "$TERRAFORMRC_PATH" ]; then
    echo ""
    echo "Existing .terraformrc found at: $TERRAFORMRC_PATH"
    echo "Contents:"
    cat "$TERRAFORMRC_PATH"
    echo ""

    read -p "Do you want to overwrite it? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo ""
        echo "Please manually add the following to your .terraformrc:"
        echo "$TERRAFORMRC_CONTENT"
        exit 0
    fi
fi

# Write the file
echo ""
echo "Writing .terraformrc to: $TERRAFORMRC_PATH"
echo "$TERRAFORMRC_CONTENT" > "$TERRAFORMRC_PATH"

echo ""
echo "✅ Development override configured!"
echo ""
echo "You can now use the provider in your Terraform configuration:"
cat <<EOF

terraform {
  required_providers {
    msgraph_entra = {
      source  = "$NAMESPACE/msgraph_entra"
      version = "~> 1.0"  # Version is ignored with dev_overrides
    }
  }
}

provider "msgraph_entra" {
  # Your configuration
}
EOF

echo ""
echo "⚠️  Note: You will see a warning about development overrides when running Terraform. This is normal."
echo "To remove the override, delete: $TERRAFORMRC_PATH"
echo ""
