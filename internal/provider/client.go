// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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

// GetDirectoryRole retrieves a directory role by display name or template ID.
func (c *GraphClient) GetDirectoryRole(ctx context.Context, roleIdentifier string) (models.DirectoryRoleable, error) {
	// Try to get by template ID first.
	roles, err := c.client.DirectoryRoles().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory roles: %w", err)
	}

	for _, role := range roles.GetValue() {
		roleTemplateID := role.GetRoleTemplateId()
		displayName := role.GetDisplayName()

		if roleTemplateID != nil && *roleTemplateID == roleIdentifier {
			return role, nil
		}
		if displayName != nil && *displayName == roleIdentifier {
			return role, nil
		}
	}

	return nil, fmt.Errorf("directory role not found: %s", roleIdentifier)
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

// FindRoleEligibilitySchedule finds an existing eligible role assignment schedule by principal, role, and scope.
// Returns nil if no matching schedule is found.
func (c *GraphClient) FindRoleEligibilitySchedule(ctx context.Context, principalID, roleDefinitionID, directoryScopeID string) (models.UnifiedRoleEligibilityScheduleable, error) {
	schedules, err := c.ListRoleEligibilitySchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list role eligibility schedules: %w", err)
	}

	for _, schedule := range schedules {
		if schedule.GetPrincipalId() != nil && *schedule.GetPrincipalId() == principalID &&
			schedule.GetRoleDefinitionId() != nil && *schedule.GetRoleDefinitionId() == roleDefinitionID &&
			schedule.GetDirectoryScopeId() != nil && *schedule.GetDirectoryScopeId() == directoryScopeID {
			return schedule, nil
		}
	}

	return nil, nil // Not found, but not an error
}
