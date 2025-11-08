// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure EntraProvider satisfies various provider interfaces.
var _ provider.Provider = &EntraProvider{}
var _ provider.ProviderWithFunctions = &EntraProvider{}
var _ provider.ProviderWithEphemeralResources = &EntraProvider{}

// EntraProvider defines the provider implementation.
type EntraProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// EntraProviderModel describes the provider data model.
type EntraProviderModel struct {
	TenantID     types.String `tfsdk:"tenant_id"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	OIDCToken    types.String `tfsdk:"oidc_token"`
	UseCLI       types.Bool   `tfsdk:"use_cli"`
}

func (p *EntraProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "msgraph-entra"
	resp.Version = p.version
}

func (p *EntraProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The msgraph_entra provider enables Terraform to manage Microsoft Entra ID (Azure AD) resources via Microsoft Graph API, with a focus on Privileged Identity Management (PIM) features.",
		Attributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The Azure AD tenant ID. Can also be set via the ENTRA_TENANT_ID or ARM_TENANT_ID environment variable.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The application (client) ID of the service principal. Can also be set via the ENTRA_CLIENT_ID or ARM_CLIENT_ID environment variable.",
				Optional:            true,
				Sensitive:           false,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The client secret of the service principal. Can also be set via the ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"oidc_token": schema.StringAttribute{
				MarkdownDescription: "The OIDC token for workload identity federation (GitHub Actions, etc.). Can also be set via the ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN environment variable. When using GitHub Actions, set this to the ACTIONS_ID_TOKEN_REQUEST_TOKEN environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"use_cli": schema.BoolAttribute{
				MarkdownDescription: "Use Azure CLI authentication instead of service principal. Defaults to false.",
				Optional:            true,
			},
		},
	}
}

func (p *EntraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data EntraProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from environment variables if not set in config
	// Support both ENTRA_* and ARM_* environment variables for compatibility
	tenantID := getEnvWithFallback("ENTRA_TENANT_ID", "ARM_TENANT_ID")
	if !data.TenantID.IsNull() {
		tenantID = data.TenantID.ValueString()
	}

	clientID := getEnvWithFallback("ENTRA_CLIENT_ID", "ARM_CLIENT_ID")
	if !data.ClientID.IsNull() {
		clientID = data.ClientID.ValueString()
	}

	clientSecret := getEnvWithFallback("ENTRA_CLIENT_SECRET", "ARM_CLIENT_SECRET")
	if !data.ClientSecret.IsNull() {
		clientSecret = data.ClientSecret.ValueString()
	}

	oidcToken := getEnvWithFallback("ENTRA_OIDC_TOKEN", "ARM_OIDC_TOKEN")
	if !data.OIDCToken.IsNull() {
		oidcToken = data.OIDCToken.ValueString()
	}

	useCLI := false
	if !data.UseCLI.IsNull() {
		useCLI = data.UseCLI.ValueBool()
	}

	// Check for ARM_USE_OIDC environment variable or GitHub Actions OIDC environment
	useOIDC := getEnvWithFallback("ENTRA_USE_OIDC", "ARM_USE_OIDC") == "true"
	githubActionsToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	inGitHubActions := githubActionsToken != ""

	// Validate configuration
	if !useCLI {
		if tenantID == "" {
			resp.Diagnostics.AddError(
				"Missing Tenant ID",
				"The provider requires a tenant_id to be configured. Set the tenant_id attribute or the ENTRA_TENANT_ID/ARM_TENANT_ID environment variable.",
			)
		}

		if clientID == "" {
			resp.Diagnostics.AddError(
				"Missing Client ID",
				"The provider requires a client_id. Set the client_id attribute or the ENTRA_CLIENT_ID/ARM_CLIENT_ID environment variable.",
			)
		}

		// Need either client_secret OR oidc_token OR GitHub Actions OIDC
		if clientSecret == "" && oidcToken == "" && !useOIDC && !inGitHubActions {
			resp.Diagnostics.AddError(
				"Missing Authentication Credentials",
				"The provider requires either client_secret or oidc_token when not using Azure CLI authentication. "+
					"For service principal with secret: set ENTRA_CLIENT_SECRET or ARM_CLIENT_SECRET. "+
					"For OIDC/Workload Identity (GitHub Actions): set ARM_USE_OIDC=true or ENTRA_OIDC_TOKEN or ARM_OIDC_TOKEN. "+
					"Alternatively, set use_cli = true to use Azure CLI authentication.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create Microsoft Graph client
	tflog.Info(ctx, "Creating Microsoft Graph client")

	var graphClient *GraphClient
	var err error

	if useCLI {
		tflog.Info(ctx, "Using Azure CLI authentication")
		graphClient, err = NewGraphClient(ctx, "", "", "", "", true, false)
	} else if oidcToken != "" {
		tflog.Info(ctx, "Using OIDC/Workload Identity authentication with explicit token", map[string]any{
			"tenant_id": tenantID,
			"client_id": clientID,
		})
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, "", oidcToken, false, false)
	} else if useOIDC || inGitHubActions {
		// Fetch OIDC token from GitHub Actions
		tflog.Info(ctx, "Attempting to fetch OIDC token from GitHub Actions", map[string]any{
			"tenant_id":         tenantID,
			"client_id":         clientID,
			"use_oidc":          useOIDC,
			"in_github_actions": inGitHubActions,
		})

		ghToken, ghErr := getGitHubActionsOIDCToken(ctx)
		if ghErr != nil {
			tflog.Warn(ctx, "Failed to fetch GitHub Actions OIDC token", map[string]any{
				"error": ghErr.Error(),
			})
			resp.Diagnostics.AddError(
				"Failed to Fetch GitHub Actions OIDC Token",
				fmt.Sprintf("Could not fetch OIDC token from GitHub Actions: %s", ghErr.Error()),
			)
			return
		}

		tflog.Info(ctx, "Successfully fetched OIDC token from GitHub Actions")
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, "", ghToken, false, false)
	} else {
		tflog.Info(ctx, "Using client secret authentication", map[string]any{
			"tenant_id": tenantID,
			"client_id": clientID,
		})
		graphClient, err = NewGraphClient(ctx, tenantID, clientID, clientSecret, "", false, false)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Microsoft Graph API Client",
			"An unexpected error occurred when creating the Microsoft Graph API client. "+
				"Error: "+err.Error(),
		)
		return
	}

	// Make the client available to resources and data sources
	resp.DataSourceData = graphClient
	resp.ResourceData = graphClient
}

func (p *EntraProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDirectoryRoleEligibleAssignmentResource,
	}
}

func (p *EntraProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *EntraProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDirectoryRoleDataSource,
	}
}

func (p *EntraProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &EntraProvider{
			version: version,
		}
	}
}

// getEnvWithFallback returns the value of the primary environment variable, or falls back to the secondary if not set.
func getEnvWithFallback(primary, secondary string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(secondary)
}

// getGitHubActionsOIDCToken fetches an OIDC token from GitHub Actions.
// Returns the token string and nil error on success, or empty string and error on failure.
func getGitHubActionsOIDCToken(ctx context.Context) (string, error) {
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")

	if requestURL == "" || requestToken == "" {
		return "", fmt.Errorf("GitHub Actions OIDC environment variables not found")
	}

	// Request OIDC token from GitHub Actions
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL+"&audience=api://AzureADTokenExchange", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token, status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Value == "" {
		return "", fmt.Errorf("token value is empty")
	}

	return result.Value, nil
}
