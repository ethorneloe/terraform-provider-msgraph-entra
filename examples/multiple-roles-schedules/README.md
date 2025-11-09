# Multiple Roles with Different Expiration Strategies

This example demonstrates managing eligible role assignments across multiple directory roles with different expiration strategies based on security requirements.

## Overview

This configuration manages three types of directory roles with different access patterns:

1. **Security Administrator** - Annual/periodic reviews with fixed end dates
2. **Global Administrator** - Tight control with duration-based expiration
3. **User Administrator** - Permanent access for operational teams

## Expiration Strategies

### Fixed Date Expiration (`afterDateTime`)

Best for:
- Annual access reviews
- Contractor/temporary staff with known end dates
- Time-bound projects

```hcl
schedule_info {
  expiration {
    type          = "afterDateTime"
    end_date_time = "2025-12-31T23:59:59Z"
  }
}
```

**Characteristics:**
- User specifies exact end date in RFC3339 format
- Clear visibility of when access expires
- Ideal for compliance and audit requirements

### Duration-Based Expiration (`afterDuration`)

Best for:
- Break-glass/emergency accounts
- Temporary elevated access
- Quarterly or periodic access windows

```hcl
schedule_info {
  expiration {
    type     = "afterDuration"
    duration = "PT24H"  # 24 hours
  }
}
```

**Characteristics:**
- Automatically calculated from creation time
- Uses ISO 8601 duration format (e.g., `PT8H` = 8 hours, `P90D` = 90 days)
- Graph converts to `afterDateTime` internally but provider preserves your config
- **Resistant to manual changes:** Won't cause drift if modified in Azure Portal

**Important Notes:**

1. **Internal Conversion:** Microsoft Graph internally converts `afterDuration` to `afterDateTime` with a computed end date. However, the provider preserves your original `type = "afterDuration"` and `duration` values in Terraform state for readability. The computed `end_date_time` is exposed as a read-only attribute for reporting.

2. **Protection Against Drift:** When using `afterDuration`, if someone manually modifies the assignment in the Azure Portal or Entra admin center (changing the end date, for example), Terraform will NOT detect this as drift and will NOT try to revert the manual change on the next apply. This is because the provider preserves your configured `type` and `duration` values rather than comparing against Graph's `end_date_time`.

   This behavior is different from `afterDateTime`, where Terraform WILL detect manual portal changes as drift and revert them on the next apply to match your configured `end_date_time`.

### Permanent Access (`noExpiration`)

Best for:
- Core operational teams
- Help desk staff
- Roles requiring continuous availability

```hcl
schedule_info {
  expiration {
    type = "noExpiration"
  }
}
```

**Characteristics:**
- Access never expires automatically
- Still requires activation through PIM
- Should be reviewed periodically via other processes

## Choosing Between afterDateTime and afterDuration

Understanding the drift detection behavior is crucial for choosing the right expiration type:

| Aspect | `afterDateTime` | `afterDuration` |
|--------|----------------|-----------------|
| **Configuration** | Explicit end date in RFC3339 | Duration in ISO 8601 (e.g., PT8H) |
| **Graph Storage** | Stored as `afterDateTime` + end date | Converted to `afterDateTime` + computed end |
| **Terraform State** | Stores exact `end_date_time` | Stores `type` + `duration` (not end date) |
| **Portal Changes** | **Detected as drift** - reverted on apply | **NOT detected** - manual changes preserved |
| **Use When** | You need strict enforcement of dates | You want flexibility for manual adjustments |
| **Best For** | Compliance, audits, contractors | Break-glass, temporary access, emergencies |

### Example Scenario

**With `afterDateTime`:**
```hcl
expiration {
  type          = "afterDateTime"
  end_date_time = "2025-12-31T23:59:59Z"
}
```
- Admin changes end date to 2026-06-30 in Azure Portal
- Next `terraform plan` shows drift: wants to change back to 2025-12-31
- `terraform apply` reverts the manual change ✅ Strict enforcement

**With `afterDuration`:**
```hcl
expiration {
  type     = "afterDuration"
  duration = "P365D"  # 365 days
}
```
- Admin changes end date in Azure Portal (Graph shows new afterDateTime)
- Next `terraform plan` shows NO drift (compares `duration` not `end_date_time`)
- `terraform apply` does nothing ✅ Manual change preserved

## Usage

1. Update the `locals` blocks with your actual users:
   ```hcl
   security_admins = [
     {
       upn           = "your-user@yourdomain.com"
       justification = "Your justification"
       end_date      = "2025-12-31T23:59:59Z"
     },
   ]
   ```

2. Apply the configuration:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. Review the outputs to verify assignments:
   ```bash
   terraform output summary
   terraform output security_admin_assignments
   ```

## Key Features Demonstrated

- **Multiple expiration types** in a single configuration
- **Data source lookups** for roles and users
- **for_each loops** for managing multiple assignments efficiently
- **Structured outputs** for monitoring and reporting
- **Different justifications** per assignment for audit trails

## Read-Only Fields

Note that `schedule_info.start_date_time` is read-only and set by Microsoft Graph when the schedule is created. It represents the actual time the eligibility started and cannot be configured by users.

## Import Support

Existing assignments can be imported using their schedule ID:

```bash
terraform import msgraph-entra_directory_role_eligible_assignment.security_admins[\"user@domain.com\"] <schedule-id>
```

**Note:** When importing assignments created with `afterDuration`, they will show as `afterDateTime` in the imported state (as this is how Graph stores them). To align with your config, simply run `terraform apply` to update the assignment in-place.

## Outputs

The configuration provides three categories of outputs:

1. **Per-role assignment details** - Schedule IDs, start times, expiration info
2. **Summary statistics** - Count of assignments per role
3. **Computed fields** - Actual start times and computed end dates (for duration-based)

These outputs are useful for:
- Auditing and compliance reporting
- Monitoring access grants
- Integration with SIEM/monitoring tools
