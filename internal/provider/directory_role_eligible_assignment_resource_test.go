// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDirectoryRoleEligibleAssignmentResource_Basic(t *testing.T) {
	// Get test user principal ID from environment (can be UPN or object ID)
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	// Resolve to object ID (handles both UPN and object ID)
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	endTime := time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, endTime),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "principal_id", principalID),
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "directory_scope_id", "/"),
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "justification", "Test assignment for acceptance testing"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "id"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "role_definition_id"),
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.type", "afterDateTime"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "msgraph-entra_directory_role_eligible_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Justification is write-only (not persisted on the schedule, only sent in requests)
				ImportStateVerifyIgnore: []string{"justification"},
			},
			// Update and Read testing - change justification (in-place update)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_updated(principalID, endTime),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "justification", "Updated justification for testing"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "id"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccDirectoryRoleEligibleAssignmentResource_UpdateEndDate(t *testing.T) {
	// Get test user principal ID from environment (can be UPN or object ID)
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	// Resolve to object ID (handles both UPN and object ID)
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	endTime1 := time.Now().UTC().Add(180 * 24 * time.Hour).Format(time.RFC3339)
	endTime2 := time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	var scheduleID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial end date
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, endTime1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.end_date_time", endTime1),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["msgraph-entra_directory_role_eligible_assignment.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						scheduleID = rs.Primary.Attributes["schedule_id"]
						if scheduleID == "" {
							return fmt.Errorf("schedule_id is empty in state")
						}
						return nil
					},
				),
			},
			// Update end date (in-place update - this is the key test for adminUpdate functionality)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, endTime2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.end_date_time", endTime2),
					// The schedule_id should remain the same - proving it's an in-place update
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["msgraph-entra_directory_role_eligible_assignment.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						got := rs.Primary.Attributes["schedule_id"]
						if got != scheduleID {
							return fmt.Errorf("schedule_id changed across update: %s -> %s", scheduleID, got)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, endTime string) string {
	return fmt.Sprintf(`
provider "msgraph-entra" {
  # Authentication is configured via environment variables:
  # - ENTRA_TENANT_ID or ARM_TENANT_ID
  # - ENTRA_CLIENT_ID or ARM_CLIENT_ID
  # - ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET (for client credentials)
  # - ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN (for OIDC/GitHub Actions)
  # Or via Azure CLI (az login)
}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Test assignment for acceptance testing"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = %[2]q
    }
  }
}
`, principalID, endTime)
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_updated(principalID, endTime string) string {
	return fmt.Sprintf(`
provider "msgraph-entra" {
  # Authentication is configured via environment variables:
  # - ENTRA_TENANT_ID or ARM_TENANT_ID
  # - ENTRA_CLIENT_ID or ARM_CLIENT_ID
  # - ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET (for client credentials)
  # - ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN (for OIDC/GitHub Actions)
  # Or via Azure CLI (az login)
}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Updated justification for testing"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = %[2]q
    }
  }
}
`, principalID, endTime)
}

// Helper function to verify assignment exists in state.
//
//nolint:unused // Available for future test enhancements
func testAccCheckDirectoryRoleEligibleAssignmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["schedule_id"] == "" {
			return fmt.Errorf("No schedule_id is set")
		}

		return nil
	}
}

// TestAccDirectoryRoleEligibleAssignmentResource_MissingScheduleInfo tests that Graph rejects
// creating an assignment without schedule_info, validating that schedule_info is required.
func TestAccDirectoryRoleEligibleAssignmentResource_MissingScheduleInfo(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %q
  directory_scope_id = "/"
  justification      = "Test without schedule_info"
}
`, principalID),
				// Graph requires schedule_info on role eligibility assignments
				ExpectError: regexp.MustCompile(`role assignment request schedule is invalid`),
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_NoExpiration tests creating an assignment
// with noExpiration type.
func TestAccDirectoryRoleEligibleAssignmentResource_NoExpiration(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %q
  directory_scope_id = "/"

  schedule_info {
    expiration {
      type = "noExpiration"
    }
  }
}
`, principalID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.type", "noExpiration"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_InvalidExpirationType tests that the
// validator rejects invalid expiration type values.
func TestAccDirectoryRoleEligibleAssignmentResource_InvalidExpirationType(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %q
  directory_scope_id = "/"

  schedule_info {
    expiration {
      type = "totallyInvalidType"
    }
  }
}
`, principalID),
				// Validator should reject invalid expiration type values
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_AfterDuration tests creating an assignment
// with afterDuration expiration type using ISO 8601 duration format.
func TestAccDirectoryRoleEligibleAssignmentResource_AfterDuration(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_afterDuration(principalID, "PT8H"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.type", "afterDuration"),
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.duration", "PT8H"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "id"),
					// Verify start_date_time is populated (read-only from Graph)
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.start_date_time"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "msgraph-entra_directory_role_eligible_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Justification is write-only (not persisted on the schedule, only sent in requests)
				// For afterDuration, Graph normalizes to afterDateTime and drops duration, so
				// import cannot reproduce the original type/duration pair exactly.
				ImportStateVerifyIgnore: []string{
					"justification",
					"schedule_info.expiration.type",
					"schedule_info.expiration.duration",
				},
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_UpdateDuration tests updating the duration
// in-place without creating a new schedule (verifies adminUpdate functionality).
func TestAccDirectoryRoleEligibleAssignmentResource_UpdateDuration(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	var scheduleID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial duration
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_afterDuration(principalID, "PT8H"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.duration", "PT8H"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["msgraph-entra_directory_role_eligible_assignment.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						scheduleID = rs.Primary.Attributes["schedule_id"]
						if scheduleID == "" {
							return fmt.Errorf("schedule_id is empty in state")
						}
						return nil
					},
				),
			},
			// Update duration (in-place update - this is the key test for adminUpdate functionality)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_afterDuration(principalID, "PT24H"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.duration", "PT24H"),
					// The schedule_id should remain the same - proving it's an in-place update
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["msgraph-entra_directory_role_eligible_assignment.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						got := rs.Primary.Attributes["schedule_id"]
						if got != scheduleID {
							return fmt.Errorf("schedule_id changed across update: %s -> %s", scheduleID, got)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_InvalidEndDateTime tests that the
// provider or Graph rejects malformed RFC3339 end_date_time values.
func TestAccDirectoryRoleEligibleAssignmentResource_InvalidEndDateTime(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %q
  directory_scope_id = "/"

  schedule_info {
    expiration {
      type          = "afterDateTime"
      end_date_time = "not-a-valid-rfc3339-date"
    }
  }
}
`, principalID),
				// Should fail validation or Graph API call
				ExpectError: regexp.MustCompile(`invalid|parse|format`),
			},
		},
	})
}

// TestAccDirectoryRoleEligibleAssignmentResource_InvalidDuration tests that the
// provider or Graph rejects malformed ISO 8601 duration values.
func TestAccDirectoryRoleEligibleAssignmentResource_InvalidDuration(t *testing.T) {
	principalIdentifier := os.Getenv("TEST_PRINCIPAL_ID")
	if principalIdentifier == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}
	principalID := testAccResolvePrincipalID(t, principalIdentifier)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %q
  directory_scope_id = "/"

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = "not-a-valid-iso8601-duration"
    }
  }
}
`, principalID),
				// Should fail validation or Graph API call
				ExpectError: regexp.MustCompile(`invalid|parse|format`),
			},
		},
	})
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_afterDuration(principalID, duration string) string {
	return fmt.Sprintf(`
provider "msgraph-entra" {}

data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Test afterDuration assignment"

  schedule_info {
    expiration {
      type     = "afterDuration"
      duration = %[2]q
    }
  }
}
`, principalID, duration)
}
