# Testing Guide - msgraph_entra Provider

This document describes how to run the acceptance tests for the msgraph_entra Terraform provider.

## Overview

The provider includes comprehensive acceptance tests that validate the full lifecycle of PIM role assignments against the actual Microsoft Graph API. These tests:

- ✅ Test data source functionality (reading directory roles)
- ✅ Test resource Create, Read, Update, Delete (CRUD) operations
- ✅ Test in-place updates using `adminUpdate` action
- ✅ Test import functionality
- ✅ Test idempotency (creating the same assignment twice)
- ✅ Verify that schedule updates don't break user access

## Prerequisites

### 1. Azure AD Tenant

You need an Azure AD (Entra ID) tenant with:
- Azure AD Premium P2 license (for PIM features)
- A test user or service principal to assign roles to

### 2. Permissions

The credentials used for testing need:
- **Application Permissions**:
  - `RoleManagement.ReadWrite.Directory`
  - `Directory.Read.All`
- **Admin Consent** granted for these permissions

### 3. Test Principal

You need a user or service principal ID to use in tests. Set this in the `TEST_PRINCIPAL_ID` environment variable.

**To get a user's object ID:**
```powershell
# Using Azure CLI
az ad user show --id user@yourcompany.com --query id -o tsv

# Using PowerShell
Get-AzADUser -UserPrincipalName user@yourcompany.com | Select-Object Id
```

**To get a service principal's object ID:**
```powershell
# Using Azure CLI
az ad sp show --id <app-id> --query id -o tsv

# Using PowerShell
Get-AzADServicePrincipal -ApplicationId <app-id> | Select-Object Id
```

## Authentication Methods

The tests support three authentication methods:

### Option 1: Azure CLI (Easiest for Local Testing)

```bash
# Login with Azure CLI
az login

# Set the test principal ID
export TEST_PRINCIPAL_ID="00000000-0000-0000-0000-000000000000"

# Run tests
TF_ACC=1 go test -v ./internal/provider/ -timeout 30m
```

### Option 2: Service Principal with Client Secret

```bash
# Set credentials
export ENTRA_TENANT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_CLIENT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_CLIENT_SECRET="your-client-secret"
export TEST_PRINCIPAL_ID="00000000-0000-0000-0000-000000000000"

# Run tests
TF_ACC=1 go test -v ./internal/provider/ -timeout 30m
```

### Option 3: OIDC Token (GitHub Actions)

```bash
# Set credentials
export ENTRA_TENANT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_CLIENT_ID="00000000-0000-0000-0000-000000000000"
export ENTRA_OIDC_TOKEN="<oidc-token>"
export TEST_PRINCIPAL_ID="00000000-0000-0000-0000-000000000000"

# Run tests
TF_ACC=1 go test -v ./internal/provider/ -timeout 30m
```

## Running Tests

### Run All Acceptance Tests

```bash
TF_ACC=1 go test -v ./internal/provider/ -timeout 30m
```

The `TF_ACC=1` environment variable is required to run acceptance tests. Without it, only unit tests run.

### Run Specific Test

```bash
# Test directory role data source
TF_ACC=1 go test -v ./internal/provider/ -run TestAccDirectoryRoleDataSource -timeout 10m

# Test basic resource lifecycle
TF_ACC=1 go test -v ./internal/provider/ -run TestAccDirectoryRoleEligibleAssignmentResource_Basic -timeout 10m

# Test in-place updates (adminUpdate)
TF_ACC=1 go test -v ./internal/provider/ -run TestAccDirectoryRoleEligibleAssignmentResource_UpdateEndDate -timeout 10m

# Test duration-based expiration
TF_ACC=1 go test -v ./internal/provider/ -run TestAccDirectoryRoleEligibleAssignmentResource_WithDuration -timeout 10m

# Test idempotent create
TF_ACC=1 go test -v ./internal/provider/ -run TestAccDirectoryRoleEligibleAssignmentResource_IdempotentCreate -timeout 10m
```

### Run with Verbose Logging

```bash
TF_ACC=1 TF_LOG=DEBUG go test -v ./internal/provider/ -timeout 30m
```

This will show detailed Terraform logs including API calls to Microsoft Graph.

## Test Cases

### 1. Directory Role Data Source Tests

**Test**: `TestAccDirectoryRoleDataSource`

Validates:
- Reading Security Administrator role
- Reading Global Administrator role
- Correct template_id is returned
- Description is populated

### 2. Basic Resource Lifecycle Tests

**Test**: `TestAccDirectoryRoleEligibleAssignmentResource_Basic`

Validates:
- **Create**: Creating a new PIM eligible assignment
- **Read**: Reading the assignment back from API
- **Update**: Updating justification (in-place update via adminUpdate)
- **Delete**: Removing the assignment
- **Import**: Importing existing assignment into state

### 3. Duration-Based Expiration Tests

**Test**: `TestAccDirectoryRoleEligibleAssignmentResource_WithDuration`

Validates:
- Creating assignment with duration (`P180D`)
- Updating duration to different value (`P365D`)
- In-place update works correctly

### 4. Update End Date Tests

**Test**: `TestAccDirectoryRoleEligibleAssignmentResource_UpdateEndDate`

This is the **critical test** for validating the adminUpdate functionality:
- Creates assignment with end date 180 days from now
- Updates end date to 365 days from now
- **Verifies the schedule_id remains the same** (proving it's an in-place update, not destroy+recreate)
- Confirms user access is not interrupted

### 5. Idempotent Create Tests

**Test**: `TestAccDirectoryRoleEligibleAssignmentResource_IdempotentCreate`

Validates:
- Creating an assignment
- Applying the same configuration again
- Provider detects existing assignment and imports it
- No error occurs

## Test Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `TF_ACC` | Yes | Must be `1` to run acceptance tests | `1` |
| `TEST_PRINCIPAL_ID` | Yes | Object ID of user/SP to test with | `00000000-0000-0000-0000-000000000000` |
| `ENTRA_TENANT_ID` or `ARM_TENANT_ID` | Conditional | Tenant ID (not needed for CLI auth) | `00000000-0000-0000-0000-000000000000` |
| `ENTRA_CLIENT_ID` or `ARM_CLIENT_ID` | Conditional | App registration client ID | `00000000-0000-0000-0000-000000000000` |
| `ENTRA_CLIENT_SECRET` or `ARM_CLIENT_SECRET` | Conditional | Client secret for SP auth | `your-secret` |
| `ENTRA_OIDC_TOKEN` or `ARM_OIDC_TOKEN` | Conditional | OIDC token for federated auth | `eyJ0eXAi...` |
| `TF_LOG` | Optional | Set to `DEBUG` for verbose logging | `DEBUG` |

## GitHub Actions Integration

For running tests in GitHub Actions, create a workflow:

```yaml
name: Acceptance Tests

on:
  pull_request:
    branches: [main]
  workflow_dispatch:

permissions:
  id-token: write  # Required for OIDC
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run Acceptance Tests
        env:
          TF_ACC: '1'
          ARM_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          ARM_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          ARM_USE_OIDC: 'true'
          TEST_PRINCIPAL_ID: ${{ secrets.TEST_PRINCIPAL_ID }}
        run: go test -v ./internal/provider/ -timeout 30m
```

## Cleanup

The tests automatically clean up resources they create. If a test fails partway through, you may need to manually remove the test assignment:

```powershell
# Using PowerShell with Microsoft.Graph module
Remove-MgRoleManagementDirectoryRoleEligibilitySchedule -UnifiedRoleEligibilityScheduleId "<schedule-id>"
```

Or via Azure Portal:
1. Go to Azure AD → Roles and administrators
2. Find the Security Administrator role
3. Click "Assignments"
4. Find the test assignment and remove it

## Troubleshooting

### "TEST_PRINCIPAL_ID environment variable must be set"

Set the `TEST_PRINCIPAL_ID` environment variable to a valid user or service principal object ID in your tenant.

### "No authentication method available"

Ensure you either:
- Are logged in with `az login`, OR
- Have set `ENTRA_TENANT_ID`, `ENTRA_CLIENT_ID`, and `ENTRA_CLIENT_SECRET`, OR
- Have set `ENTRA_TENANT_ID`, `ENTRA_CLIENT_ID`, and `ENTRA_OIDC_TOKEN`

### "RoleManagement.ReadWrite.Directory permission required"

Your service principal needs application permission `RoleManagement.ReadWrite.Directory` with admin consent granted.

### Tests timeout

Some tests may take longer if the Microsoft Graph API is slow. Increase the timeout:

```bash
TF_ACC=1 go test -v ./internal/provider/ -timeout 60m
```

### Rate Limiting

If you hit rate limits during testing, you'll see 429 errors. Wait a few minutes and try again, or reduce the number of tests running concurrently.

## Best Practices

1. **Use a test/dev tenant**: Don't run acceptance tests against production
2. **Use a dedicated test user/SP**: Create a specific test principal for automation
3. **Clean up manually if needed**: If tests fail, verify cleanup completed
4. **Run tests before PRs**: Ensure all tests pass before submitting changes
5. **Review logs**: Check `TF_LOG=DEBUG` output for API call details

## What the Tests Verify

The comprehensive test suite validates:

### Functional Correctness
- ✅ Assignments are created successfully
- ✅ Assignments can be read back correctly
- ✅ Updates modify existing assignments (not destroy+recreate)
- ✅ Deletes remove assignments completely
- ✅ Import brings existing assignments into state

### API Correctness
- ✅ Queries schedules (persistent objects) not requests (transient)
- ✅ Uses `adminUpdate` action for in-place updates
- ✅ Uses `adminAssign` action for creates
- ✅ Uses `adminRemove` action for deletes

### User Experience
- ✅ No access disruption during updates (schedule_id doesn't change)
- ✅ Idempotent creates (same config twice doesn't error)
- ✅ Drift detection works (manual changes detected)
- ✅ Different expiration types work (afterDateTime, afterDuration)

## Contributing

When adding new features:

1. Add corresponding acceptance tests
2. Ensure all existing tests still pass
3. Document any new test requirements
4. Update this TESTING.md if authentication or setup changes

## Support

For issues with tests:
- Check the [troubleshooting section](#troubleshooting)
- Review test logs with `TF_LOG=DEBUG`
- Open an issue at https://github.com/ethorneloe/terraform-provider-msgraph-entra/issues
