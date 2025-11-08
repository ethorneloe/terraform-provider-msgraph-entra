# Cleanup script to remove existing test role assignments
# Run this before running acceptance tests if you get "assignment already exists" errors

$ErrorActionPreference = "Stop"

Write-Host "Cleaning up existing test role assignments..." -ForegroundColor Cyan
Write-Host ""

# Check for required environment variable
$principalId = $env:TEST_PRINCIPAL_ID
if (-not $principalId) {
    Write-Host "ERROR: TEST_PRINCIPAL_ID environment variable must be set" -ForegroundColor Red
    exit 1
}

Write-Host "Test principal ID: $principalId" -ForegroundColor Yellow
Write-Host ""

# Security Administrator role definition ID (this is constant across all Entra ID tenants)
$roleDefinitionId = "194ae4cb-b126-40b2-bd5b-6091b380977d"

# Find all eligible assignments for this principal + role
Write-Host "Searching for existing eligible assignments..." -ForegroundColor Cyan
$existingSchedules = Get-MgRoleManagementDirectoryRoleEligibilitySchedule -All | Where-Object {
    $_.PrincipalId -eq $principalId -and
    $_.RoleDefinitionId -eq $roleDefinitionId -and
    $_.DirectoryScopeId -eq "/"
}

if ($existingSchedules.Count -eq 0) {
    Write-Host "No existing eligible assignments found. You're good to run tests!" -ForegroundColor Green
    exit 0
}

Write-Host "Found $($existingSchedules.Count) existing eligible assignment(s):" -ForegroundColor Yellow
foreach ($schedule in $existingSchedules) {
    Write-Host "  - Schedule ID: $($schedule.Id)" -ForegroundColor White
}
Write-Host ""

# Delete each schedule
Write-Host "Deleting existing eligible assignments..." -ForegroundColor Cyan
foreach ($schedule in $existingSchedules) {
    Write-Host "  Deleting schedule: $($schedule.Id)" -ForegroundColor White

    $params = @{
        Action = "adminRemove"
        PrincipalId = $principalId
        RoleDefinitionId = $roleDefinitionId
        DirectoryScopeId = "/"
        TargetScheduleId = $schedule.Id
    }

    try {
        New-MgRoleManagementDirectoryRoleEligibilityScheduleRequest -BodyParameter $params | Out-Null
        Write-Host "    Submitted removal request" -ForegroundColor Green
    }
    catch {
        Write-Host "    WARNING: Failed to delete: $_" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Waiting 30 seconds for Azure to process deletions..." -ForegroundColor Cyan
Start-Sleep -Seconds 30

# Verify cleanup
Write-Host "Verifying cleanup..." -ForegroundColor Cyan
$remainingSchedules = Get-MgRoleManagementDirectoryRoleEligibilitySchedule -All | Where-Object {
    $_.PrincipalId -eq $principalId -and
    $_.RoleDefinitionId -eq $roleDefinitionId -and
    $_.DirectoryScopeId -eq "/"
}

if ($remainingSchedules.Count -eq 0) {
    Write-Host ""
    Write-Host "SUCCESS: All eligible assignments have been removed!" -ForegroundColor Green
    Write-Host "You can now run acceptance tests." -ForegroundColor Green
    exit 0
}
else {
    Write-Host ""
    Write-Host "WARNING: $($remainingSchedules.Count) schedule(s) still exist:" -ForegroundColor Yellow
    foreach ($schedule in $remainingSchedules) {
        Write-Host "  - Schedule ID: $($schedule.Id)" -ForegroundColor White
    }
    Write-Host ""
    Write-Host "Azure may still be processing the deletions. Wait another 30 seconds and try running tests." -ForegroundColor Yellow
    exit 1
}
