// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDirectoryRoleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - Security Administrator role
			{
				Config: testAccDirectoryRoleDataSourceConfig("Security Administrator"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.msgraph-entra_directory_role.test", "display_name", "Security Administrator"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "id"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "template_id"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "description"),
				),
			},
			// Read testing - Global Administrator role
			{
				Config: testAccDirectoryRoleDataSourceConfig("Global Administrator"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.msgraph-entra_directory_role.test", "display_name", "Global Administrator"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "id"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "template_id"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_directory_role.test", "description"),
				),
			},
		},
	})
}

func testAccDirectoryRoleDataSourceConfig(displayName string) string {
	return `
terraform {
  required_providers {
    msgraph-entra = {
      source = "ethorneloe/msgraph-entra"
    }
  }
}

provider "msgraph-entra" {
  # Authentication is configured via environment variables:
  # - ENTRA_TENANT_ID or ARM_TENANT_ID
  # - ENTRA_CLIENT_ID or ARM_CLIENT_ID
  # - ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET (for client credentials)
  # - ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN (for OIDC/GitHub Actions)
  # Or via Azure CLI (az login)
}

data "msgraph-entra_directory_role" "test" {
  display_name = "` + displayName + `"
}
`
}
