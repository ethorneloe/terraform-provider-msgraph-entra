// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	abstractions "github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DirectoryRoleEligibleAssignmentResource{}
var _ resource.ResourceWithImportState = &DirectoryRoleEligibleAssignmentResource{}

func NewDirectoryRoleEligibleAssignmentResource() resource.Resource {
	return &DirectoryRoleEligibleAssignmentResource{}
}

// DirectoryRoleEligibleAssignmentResource defines the resource implementation.
type DirectoryRoleEligibleAssignmentResource struct {
	client *GraphClient
}

// DirectoryRoleEligibleAssignmentResourceModel describes the resource data model.
type DirectoryRoleEligibleAssignmentResourceModel struct {
	ID                 types.String                      `tfsdk:"id"`
	RoleDefinitionID   types.String                      `tfsdk:"role_definition_id"`
	PrincipalID        types.String                      `tfsdk:"principal_id"`
	DirectoryScopeID   types.String                      `tfsdk:"directory_scope_id"`
	Justification      types.String                      `tfsdk:"justification"`
	ScheduleInfo       *ScheduleInfoModel                `tfsdk:"schedule_info"`
	ScheduleID         types.String                      `tfsdk:"schedule_id"`
}

// ScheduleInfoModel represents the schedule configuration
type ScheduleInfoModel struct {
	StartDateTime types.String      `tfsdk:"start_date_time"`
	Expiration    *ExpirationModel  `tfsdk:"expiration"`
}

// ExpirationModel represents the expiration configuration
type ExpirationModel struct {
	Type          types.String `tfsdk:"type"`
	EndDateTime   types.String `tfsdk:"end_date_time"`
	Duration      types.String `tfsdk:"duration"`
}

func (r *DirectoryRoleEligibleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_role_eligible_assignment"
}

func (r *DirectoryRoleEligibleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an eligible assignment for an Entra ID directory role using Privileged Identity Management (PIM). " +
			"This resource creates time-bound eligible role assignments that users must activate to use.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the role eligibility schedule request.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_definition_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the directory role definition (template ID). Example: '62e90394-69f5-4237-9190-012177145e10' for Global Administrator.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_id": schema.StringAttribute{
				MarkdownDescription: "The object ID of the principal (user, group, or service principal) to assign the eligible role to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"directory_scope_id": schema.StringAttribute{
				MarkdownDescription: "The scope of the role assignment. Use '/' for tenant-wide scope.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"justification": schema.StringAttribute{
				MarkdownDescription: "Justification for the role assignment.",
				Optional:            true,
			},
			"schedule_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the created role eligibility schedule.",
				Computed:            true,
			},
			"schedule_info": schema.SingleNestedAttribute{
				MarkdownDescription: "The schedule of the role eligibility.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"start_date_time": schema.StringAttribute{
						MarkdownDescription: "When the eligibility starts. Must be in RFC3339 format (e.g., '2025-01-08T00:00:00Z'). Defaults to now.",
						Optional:            true,
						Computed:            true,
					},
					"expiration": schema.SingleNestedAttribute{
						MarkdownDescription: "When and how the eligibility expires.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								MarkdownDescription: "The type of expiration: 'noExpiration', 'afterDateTime', or 'afterDuration'.",
								Optional:            true,
								Computed:            true,
							},
							"end_date_time": schema.StringAttribute{
								MarkdownDescription: "The end date/time when type is 'afterDateTime'. Must be in RFC3339 format.",
								Optional:            true,
							},
							"duration": schema.StringAttribute{
								MarkdownDescription: "The duration when type is 'afterDuration'. Use ISO 8601 duration format (e.g., 'PT8H' for 8 hours, 'P365D' for 365 days).",
								Optional:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (r *DirectoryRoleEligibleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*GraphClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *GraphClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *DirectoryRoleEligibleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults
	if data.DirectoryScopeID.IsNull() {
		data.DirectoryScopeID = types.StringValue("/")
	}

	// Create the role eligibility schedule request
	request := models.NewUnifiedRoleEligibilityScheduleRequest()

	action := models.ADMINASSIGN_UNIFIEDROLESCHEDULEREQUESTACTIONS
	request.SetAction(&action)

	roleDefID := data.RoleDefinitionID.ValueString()
	request.SetRoleDefinitionId(&roleDefID)

	principalID := data.PrincipalID.ValueString()
	request.SetPrincipalId(&principalID)

	dirScopeID := data.DirectoryScopeID.ValueString()
	request.SetDirectoryScopeId(&dirScopeID)

	if !data.Justification.IsNull() {
		justification := data.Justification.ValueString()
		request.SetJustification(&justification)
	}

	// Set schedule info
	if data.ScheduleInfo != nil {
		scheduleInfo := models.NewRequestSchedule()

		// Start date time
		var startDateTime time.Time
		if !data.ScheduleInfo.StartDateTime.IsNull() {
			var err error
			startDateTime, err = time.Parse(time.RFC3339, data.ScheduleInfo.StartDateTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Start Date Time",
					fmt.Sprintf("Could not parse start_date_time: %s", err.Error()),
				)
				return
			}
		} else {
			startDateTime = time.Now().UTC()
		}
		scheduleInfo.SetStartDateTime(&startDateTime)

		// Expiration
		if data.ScheduleInfo.Expiration != nil {
			expiration := models.NewExpirationPattern()

			if !data.ScheduleInfo.Expiration.Type.IsNull() {
				expTypeStr := data.ScheduleInfo.Expiration.Type.ValueString()
				var expType models.ExpirationPatternType
				switch expTypeStr {
				case "noExpiration":
					expType = models.NOEXPIRATION_EXPIRATIONPATTERNTYPE
				case "afterDateTime":
					expType = models.AFTERDATETIME_EXPIRATIONPATTERNTYPE
				case "afterDuration":
					expType = models.AFTERDURATION_EXPIRATIONPATTERNTYPE
				}
				expiration.SetTypeEscaped(&expType)
			}

			if !data.ScheduleInfo.Expiration.EndDateTime.IsNull() {
				endDateTime, err := time.Parse(time.RFC3339, data.ScheduleInfo.Expiration.EndDateTime.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Invalid End Date Time",
						fmt.Sprintf("Could not parse end_date_time: %s", err.Error()),
					)
					return
				}
				expiration.SetEndDateTime(&endDateTime)
			}

			if !data.ScheduleInfo.Expiration.Duration.IsNull() {
				// Duration as ISO 8601 format (e.g., "PT8H", "P365D")
				durationStr := data.ScheduleInfo.Expiration.Duration.ValueString()
				duration, err := abstractions.ParseISODuration(durationStr)
				if err != nil {
					resp.Diagnostics.AddError(
						"Invalid Duration",
						fmt.Sprintf("Could not parse duration '%s': %s", durationStr, err.Error()),
					)
					return
				}
				expiration.SetDuration(duration)
			}

			scheduleInfo.SetExpiration(expiration)
		}

		request.SetScheduleInfo(scheduleInfo)
	}

	// Create the request
	tflog.Info(ctx, "Creating directory role eligible assignment", map[string]any{
		"role_definition_id": roleDefID,
		"principal_id":       principalID,
		"directory_scope_id": dirScopeID,
	})

	result, err := r.client.CreateRoleEligibilityScheduleRequest(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Role Eligible Assignment",
			"Could not create role eligible assignment: "+err.Error(),
		)
		return
	}

	// Set the ID and schedule ID
	if result.GetId() != nil {
		data.ID = types.StringPointerValue(result.GetId())
	}

	if result.GetTargetScheduleId() != nil {
		data.ScheduleID = types.StringPointerValue(result.GetTargetScheduleId())
	}

	tflog.Trace(ctx, "Created directory role eligible assignment", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the request to verify it still exists
	scheduleRequest, err := r.client.GetRoleEligibilityScheduleRequest(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role Eligible Assignment",
			"Could not read role eligible assignment ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update computed fields
	if scheduleRequest.GetTargetScheduleId() != nil {
		data.ScheduleID = types.StringPointerValue(scheduleRequest.GetTargetScheduleId())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Most changes require replacement, but justification can be updated
	// For now, we'll treat updates as requiring recreation
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"This resource does not support in-place updates. Changes will require replacement.",
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the role eligibility
	tflog.Info(ctx, "Deleting directory role eligible assignment", map[string]any{
		"id":         data.ID.ValueString(),
		"schedule_id": data.ScheduleID.ValueString(),
	})

	err := r.client.DeleteRoleEligibilityScheduleRequest(
		ctx,
		data.ScheduleID.ValueString(),
		data.PrincipalID.ValueString(),
		data.RoleDefinitionID.ValueString(),
		data.DirectoryScopeID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Role Eligible Assignment",
			"Could not delete role eligible assignment: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Deleted directory role eligible assignment")
}

func (r *DirectoryRoleEligibleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
