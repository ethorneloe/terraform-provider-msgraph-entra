#!/bin/bash
# Cleanup script to remove existing test role assignments
# Run this before running acceptance tests if you get "assignment already exists" errors

set -e

echo "Cleaning up existing test role assignments..."

# Check for required environment variables
if [ -z "$TEST_PRINCIPAL_ID" ]; then
    echo "ERROR: TEST_PRINCIPAL_ID environment variable must be set"
    exit 1
fi

echo "Test principal: $TEST_PRINCIPAL_ID"
echo ""
echo "Please manually remove any existing 'Security Administrator' role assignments"
echo "for this user in the Azure Portal:"
echo ""
echo "1. Go to https://portal.azure.com"
echo "2. Navigate to: Entra ID > Roles and administrators"
echo "3. Search for 'Security Administrator'"
echo "4. Click on the role"
echo "5. Go to 'Eligible assignments' tab"
echo "6. Remove any assignments for user: $TEST_PRINCIPAL_ID"
echo ""
echo "Then wait 30 seconds for Azure to fully process the deletion"
echo ""
read -p "Press Enter after cleanup is complete..."

echo "Cleanup preparation complete. You can now run tests."
