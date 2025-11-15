// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
	// Get test user UPN from environment variable
	testUserUPN := os.Getenv("TEST_PRINCIPAL_ID")
	if testUserUPN == "" {
		t.Skip("TEST_PRINCIPAL_ID environment variable not set, skipping user data source test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccUserDataSourceConfig(testUserUPN),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.msgraph-entra_user.test", "user_principal_name", testUserUPN),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_user.test", "id"),
					resource.TestCheckResourceAttrSet("data.msgraph-entra_user.test", "display_name"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig(upn string) string {
	return `
provider "msgraph-entra" {
  # Authentication is configured via environment variables:
  # - ENTRA_TENANT_ID or ARM_TENANT_ID
  # - ENTRA_CLIENT_ID or ARM_CLIENT_ID
  # - ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET (for client credentials)
  # - ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN (for OIDC/GitHub Actions)
  # Or via Azure CLI (az login)
}

data "msgraph-entra_user" "test" {
  user_principal_name = "` + upn + `"
}
`
}
