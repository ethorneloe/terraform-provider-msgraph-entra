// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
)

// GraphClient wraps the Microsoft Graph SDK client.
type GraphClient struct {
	client *msgraphsdk.GraphServiceClient
}

// NewGraphClient creates a new Microsoft Graph API client.
func NewGraphClient(ctx context.Context, tenantID, clientID, clientSecret, oidcToken string, useCLI, _ bool) (*GraphClient, error) {
	var credential azcore.TokenCredential
	var err error

	// Support multiple authentication methods
	if oidcToken != "" {
		// OIDC/Workload Identity Federation with explicit token
		// If oidcToken is explicitly provided, use ClientAssertionCredential
		credential, err = azidentity.NewClientAssertionCredential(
			tenantID,
			clientID,
			func(ctx context.Context) (string, error) {
				return oidcToken, nil
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC credential: %w", err)
		}
	} else if clientID != "" && clientSecret != "" {
		// Client credentials flow (service principal with secret)
		credential, err = azidentity.NewClientSecretCredential(
			tenantID,
			clientID,
			clientSecret,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
	} else if useCLI {
		// Azure CLI authentication
		credential, err = azidentity.NewAzureCLICredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure CLI credential: %w", err)
		}
	} else {
		// Try default Azure credential chain (includes managed identity, environment vars, etc.)
		credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default Azure credential: %w", err)
		}
	}

	// Create Graph client
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(
		credential,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %w", err)
	}

	return &GraphClient{
		client: client,
	}, nil
}

// GetDirectoryRole retrieves a directory role template by display name or template ID.
// This returns the role template information needed for creating PIM eligible assignments.
// The function searches directory role templates, not activated roles, since we only need
// the template ID for PIM operations. Template IDs are matched exactly (case-sensitive),
// while display names are matched case-insensitively.
func (c *GraphClient) GetDirectoryRole(ctx context.Context, roleIdentifier string) (models.DirectoryRoleable, error) {
	// Search directory role templates
	// These are the built-in role definitions available in Entra ID
	templates, err := c.client.DirectoryRoleTemplates().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory role templates: %w", err)
	}

	// Convert identifier to lowercase for case-insensitive display name matching
	lowerIdentifier := strings.ToLower(roleIdentifier)

	for _, template := range templates.GetValue() {
		templateID := template.GetId()
		displayName := template.GetDisplayName()

		// Match by template ID (exact match, case-sensitive)
		if templateID != nil && *templateID == roleIdentifier {
			return convertTemplateToDirectoryRole(template), nil
		}

		// Match by display name (case-insensitive)
		if displayName != nil && strings.ToLower(*displayName) == lowerIdentifier {
			return convertTemplateToDirectoryRole(template), nil
		}
	}

	return nil, fmt.Errorf("directory role template not found: %s", roleIdentifier)
}

// convertTemplateToDirectoryRole creates a DirectoryRole object from a template.
// For PIM eligible assignments, we only need the template ID which is stored in both
// the Id and RoleTemplateId fields for compatibility.
func convertTemplateToDirectoryRole(template models.DirectoryRoleTemplateable) models.DirectoryRoleable {
	role := models.NewDirectoryRole()
	templateID := template.GetId()
	role.SetId(templateID)
	role.SetDisplayName(template.GetDisplayName())
	role.SetDescription(template.GetDescription())
	role.SetRoleTemplateId(templateID)
	return role
}

// GetDirectoryRoleDefinition retrieves a role definition by ID.
func (c *GraphClient) GetDirectoryRoleDefinition(ctx context.Context, roleDefinitionID string) (models.UnifiedRoleDefinitionable, error) {
	role, err := c.client.RoleManagement().Directory().RoleDefinitions().ByUnifiedRoleDefinitionId(roleDefinitionID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role definition: %w", err)
	}
	return role, nil
}

// CreateRoleEligibilityScheduleRequest creates an eligible role assignment.
func (c *GraphClient) CreateRoleEligibilityScheduleRequest(ctx context.Context, request models.UnifiedRoleEligibilityScheduleRequestable) (models.UnifiedRoleEligibilityScheduleRequestable, error) {
	result, err := c.client.RoleManagement().Directory().RoleEligibilityScheduleRequests().Post(ctx, request, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create role eligibility schedule request: %w", err)
	}
	return result, nil
}

// GetRoleEligibilityScheduleRequest retrieves a role eligibility schedule request.
func (c *GraphClient) GetRoleEligibilityScheduleRequest(ctx context.Context, requestID string) (models.UnifiedRoleEligibilityScheduleRequestable, error) {
	result, err := c.client.RoleManagement().Directory().RoleEligibilityScheduleRequests().ByUnifiedRoleEligibilityScheduleRequestId(requestID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role eligibility schedule request: %w", err)
	}
	return result, nil
}

// DeleteRoleEligibilityScheduleRequest deletes (revokes) an eligible role assignment.
func (c *GraphClient) DeleteRoleEligibilityScheduleRequest(ctx context.Context, roleEligibilityScheduleID string, principalID, roleDefinitionID, directoryScopeID string) error {
	// To delete, we create a new request with action = "adminRemove".
	request := models.NewUnifiedRoleEligibilityScheduleRequest()
	action := models.ADMINREMOVE_UNIFIEDROLESCHEDULEREQUESTACTIONS
	request.SetAction(&action)
	request.SetPrincipalId(&principalID)
	request.SetRoleDefinitionId(&roleDefinitionID)
	request.SetDirectoryScopeId(&directoryScopeID)
	request.SetTargetScheduleId(&roleEligibilityScheduleID)

	_, err := c.client.RoleManagement().Directory().RoleEligibilityScheduleRequests().Post(ctx, request, nil)
	if err != nil {
		return fmt.Errorf("failed to delete role eligibility: %w", err)
	}
	return nil
}

// ListRoleEligibilitySchedules lists all eligible role assignments.
func (c *GraphClient) ListRoleEligibilitySchedules(ctx context.Context) ([]models.UnifiedRoleEligibilityScheduleable, error) {
	result, err := c.client.RoleManagement().Directory().RoleEligibilitySchedules().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role eligibility schedules: %w", err)
	}
	return result.GetValue(), nil
}

// GetRoleEligibilitySchedule retrieves a specific eligible role assignment by ID.
func (c *GraphClient) GetRoleEligibilitySchedule(ctx context.Context, scheduleID string) (models.UnifiedRoleEligibilityScheduleable, error) {
	result, err := c.client.RoleManagement().Directory().RoleEligibilitySchedules().ByUnifiedRoleEligibilityScheduleId(scheduleID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role eligibility schedule: %w", err)
	}
	return result, nil
}

// GetRoleManagementDirectory is a helper function to get RoleManagement Directory RequestBuilder.
func (c *GraphClient) GetRoleManagementDirectory() *rolemanagement.DirectoryRequestBuilder {
	return c.client.RoleManagement().Directory()
}

// GetUserByUserPrincipalName retrieves a user's object ID by their UPN (email address).
// This is useful for converting user-friendly UPNs to object IDs required by the Graph API.
func (c *GraphClient) GetUserByUserPrincipalName(ctx context.Context, upn string) (string, error) {
	user, err := c.client.Users().ByUserId(upn).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get user by UPN %s: %w", upn, err)
	}

	if user.GetId() == nil {
		return "", fmt.Errorf("user %s has no object ID", upn)
	}

	return *user.GetId(), nil
}

// FindRoleEligibilitySchedule finds an existing eligible role assignment schedule by principal, role, and scope.
// Returns nil if no matching schedule is found.
func (c *GraphClient) FindRoleEligibilitySchedule(ctx context.Context, principalID, roleDefinitionID, directoryScopeID string) (models.UnifiedRoleEligibilityScheduleable, error) {
	// Use OData filter to query only the specific schedule we're looking for
	// This is more efficient than listing all schedules and filtering in memory
	filterQuery := fmt.Sprintf(
		"principalId eq '%s' and roleDefinitionId eq '%s' and directoryScopeId eq '%s'",
		principalID,
		roleDefinitionID,
		directoryScopeID,
	)

	requestConfig := &rolemanagement.DirectoryRoleEligibilitySchedulesRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.DirectoryRoleEligibilitySchedulesRequestBuilderGetQueryParameters{
			Filter: &filterQuery,
		},
	}

	result, err := c.client.RoleManagement().Directory().RoleEligibilitySchedules().Get(ctx, requestConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find role eligibility schedule: %w", err)
	}

	schedules := result.GetValue()
	if len(schedules) == 0 {
		return nil, nil // Not found, but not an error
	}

	// Return the first matching schedule (should only be one)
	return schedules[0], nil
}
