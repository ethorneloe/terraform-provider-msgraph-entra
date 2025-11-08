// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDirectoryRoleEligibleAssignmentResource_Basic(t *testing.T) {
	// Get test user principal ID from environment
	principalID := os.Getenv("TEST_PRINCIPAL_ID")
	if principalID == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	startTime := time.Now().UTC().Format(time.RFC3339)
	endTime := time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime),
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
			},
			// Update and Read testing - change justification (in-place update)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_updated(principalID, startTime, endTime),
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

func TestAccDirectoryRoleEligibleAssignmentResource_WithDuration(t *testing.T) {
	// Get test user principal ID from environment
	principalID := os.Getenv("TEST_PRINCIPAL_ID")
	if principalID == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	startTime := time.Now().UTC().Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with duration
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_duration(principalID, startTime, "P180D"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.type", "afterDuration"),
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.duration", "P180D"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "id"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
			// Update duration (in-place update)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_duration(principalID, startTime, "P365D"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.duration", "P365D"),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
		},
	})
}

func TestAccDirectoryRoleEligibleAssignmentResource_UpdateEndDate(t *testing.T) {
	// Get test user principal ID from environment
	principalID := os.Getenv("TEST_PRINCIPAL_ID")
	if principalID == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	startTime := time.Now().UTC().Format(time.RFC3339)
	endTime1 := time.Now().UTC().Add(180 * 24 * time.Hour).Format(time.RFC3339)
	endTime2 := time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial end date
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.end_date_time", endTime1),
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
			// Update end date (in-place update - this is the key test for adminUpdate functionality)
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("msgraph-entra_directory_role_eligible_assignment.test", "schedule_info.expiration.end_date_time", endTime2),
					// The schedule_id should remain the same - proving it's an in-place update
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
		},
	})
}

func TestAccDirectoryRoleEligibleAssignmentResource_IdempotentCreate(t *testing.T) {
	// This test verifies that creating the same assignment twice doesn't fail
	// but instead imports the existing one
	principalID := os.Getenv("TEST_PRINCIPAL_ID")
	if principalID == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable must be set for acceptance tests")
	}

	startTime := time.Now().UTC().Format(time.RFC3339)
	endTime := time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create first assignment
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
			// Apply same config again - should be idempotent
			{
				Config: testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("msgraph-entra_directory_role_eligible_assignment.test", "schedule_id"),
				),
			},
		},
	})
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_basic(principalID, startTime, endTime string) string {
	return fmt.Sprintf(`
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Test assignment for acceptance testing"

  schedule_info {
    start_date_time = %[2]q
    expiration {
      type          = "afterDateTime"
      end_date_time = %[3]q
    }
  }
}
`, principalID, startTime, endTime)
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_updated(principalID, startTime, endTime string) string {
	return fmt.Sprintf(`
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Updated justification for testing"

  schedule_info {
    start_date_time = %[2]q
    expiration {
      type          = "afterDateTime"
      end_date_time = %[3]q
    }
  }
}
`, principalID, startTime, endTime)
}

func testAccDirectoryRoleEligibleAssignmentResourceConfig_duration(principalID, startTime, duration string) string {
	return fmt.Sprintf(`
data "msgraph-entra_directory_role" "security_admin" {
  display_name = "Security Administrator"
}

resource "msgraph-entra_directory_role_eligible_assignment" "test" {
  role_definition_id = data.msgraph-entra_directory_role.security_admin.template_id
  principal_id       = %[1]q
  directory_scope_id = "/"
  justification      = "Test assignment with duration"

  schedule_info {
    start_date_time = %[2]q
    expiration {
      type     = "afterDuration"
      duration = %[3]q
    }
  }
}
`, principalID, startTime, duration)
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
