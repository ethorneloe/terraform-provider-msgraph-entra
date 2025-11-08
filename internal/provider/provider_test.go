// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
//
//nolint:unused // Used by acceptance tests in _test.go files
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"msgraph-entra": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the msgraph-entra provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
//
//nolint:unused // Used by acceptance tests in _test.go files when testing ephemeral resources
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"msgraph-entra": providerserver.NewProtocol6WithError(New("test")()),
	"echo":          echoprovider.NewProviderServer(),
}

//nolint:unused // Used by acceptance tests in _test.go files
func testAccPreCheck(t *testing.T) {
	// Check for required environment variables for acceptance testing
	// These tests require actual Azure AD/Entra ID credentials

	// Option 1: Use Azure CLI authentication (easiest for local testing)
	// Just ensure you're logged in with: az login

	// Option 2: Use service principal with client credentials
	// Set these environment variables:
	// - ENTRA_TENANT_ID or ARM_TENANT_ID
	// - ENTRA_CLIENT_ID or ARM_CLIENT_ID
	// - ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET

	// Option 3: Use OIDC token (for GitHub Actions)
	// - ENTRA_TENANT_ID or ARM_TENANT_ID
	// - ENTRA_CLIENT_ID or ARM_CLIENT_ID
	// - ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN

	// The actual authentication will be handled by the provider Configure method
	// We just need to ensure that SOME authentication method is available

	// Check if we have any credentials configured
	hasAzureCLI := false
	hasClientCreds := false
	hasOIDC := false

	// Check for tenant ID (required for non-CLI auth)
	tenantID := getEnvWithFallback("ENTRA_TENANT_ID", "ARM_TENANT_ID")
	clientID := getEnvWithFallback("ENTRA_CLIENT_ID", "ARM_CLIENT_ID")
	clientSecret := getEnvWithFallback("ENTRA_CLIENT_SECRET", "ARM_CLIENT_SECRET")
	oidcToken := getEnvWithFallback("ENTRA_OIDC_TOKEN", "ARM_OIDC_TOKEN")
	useOIDC := getEnvWithFallback("ENTRA_USE_OIDC", "ARM_USE_OIDC") == "true"

	if clientID != "" && clientSecret != "" && tenantID != "" {
		hasClientCreds = true
	}

	// OIDC is available if we have explicit token OR if ARM_USE_OIDC=true (GitHub Actions)
	if clientID != "" && tenantID != "" && (oidcToken != "" || useOIDC) {
		hasOIDC = true
	}

	// For Azure CLI, we assume if neither of the above are set, CLI will be used
	// The actual check for CLI availability happens in the provider
	if !hasClientCreds && !hasOIDC {
		hasAzureCLI = true // Optimistically assume CLI is available
	}

	if !hasAzureCLI && !hasClientCreds && !hasOIDC {
		t.Fatal("No authentication method available. " +
			"Either run 'az login' for Azure CLI authentication, " +
			"or set ENTRA_TENANT_ID, ENTRA_CLIENT_ID, and ENTRA_CLIENT_SECRET for service principal authentication, " +
			"or set ENTRA_TENANT_ID, ENTRA_CLIENT_ID, and ENTRA_OIDC_TOKEN for OIDC authentication.")
	}

	t.Logf("Using authentication method: CLI=%v, ClientCreds=%v, OIDC=%v", hasAzureCLI, hasClientCreds, hasOIDC)
}

//nolint:unused // Used by acceptance tests in _test.go files
func testAccResolvePrincipalID(t *testing.T, principalIdentifier string) string {
	// If it's already a GUID (object ID), return as-is
	// GUIDs are 36 characters: 8-4-4-4-12 with dashes
	if len(principalIdentifier) == 36 && principalIdentifier[8] == '-' && principalIdentifier[13] == '-' {
		return principalIdentifier
	}

	// Otherwise, assume it's a UPN and resolve it to object ID
	t.Logf("Resolving UPN %s to object ID", principalIdentifier)

	// Create a temporary Graph client to resolve the UPN
	ctx := context.Background()
	tenantID := getEnvWithFallback("ENTRA_TENANT_ID", "ARM_TENANT_ID")
	clientID := getEnvWithFallback("ENTRA_CLIENT_ID", "ARM_CLIENT_ID")
	clientSecret := getEnvWithFallback("ENTRA_CLIENT_SECRET", "ARM_CLIENT_SECRET")
	oidcToken := getEnvWithFallback("ENTRA_OIDC_TOKEN", "ARM_OIDC_TOKEN")
	useOIDC := getEnvWithFallback("ENTRA_USE_OIDC", "ARM_USE_OIDC") == "true"

	var graphClient *GraphClient
	var err error

	// Try OIDC first (for GitHub Actions)
	if useOIDC {
		// Fetch OIDC token from GitHub Actions
		ghToken, ghErr := getGitHubActionsOIDCToken(ctx)
		if ghErr != nil {
			t.Fatalf("Failed to fetch OIDC token: %v", ghErr)
		}
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, "", ghToken, false, false)
	} else if clientSecret != "" {
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, clientSecret, oidcToken, false, false)
	} else {
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, "", "", true, false)
	}

	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	objectID, err := graphClient.GetUserByUserPrincipalName(ctx, principalIdentifier)
	if err != nil {
		t.Fatalf("Failed to resolve UPN %s to object ID: %v", principalIdentifier, err)
	}

	t.Logf("Resolved UPN %s to object ID %s", principalIdentifier, objectID)
	return objectID
}
