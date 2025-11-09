// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	ID               types.String       `tfsdk:"id"`
	RoleDefinitionID types.String       `tfsdk:"role_definition_id"`
	PrincipalID      types.String       `tfsdk:"principal_id"`
	DirectoryScopeID types.String       `tfsdk:"directory_scope_id"`
	Justification    types.String       `tfsdk:"justification"`
	ScheduleInfo     *ScheduleInfoModel `tfsdk:"schedule_info"`
	ScheduleID       types.String       `tfsdk:"schedule_id"`
}

// ScheduleInfoModel represents the schedule configuration.
type ScheduleInfoModel struct {
	StartDateTime types.String     `tfsdk:"start_date_time"`
	Expiration    *ExpirationModel `tfsdk:"expiration"`
}

// ExpirationModel represents the expiration configuration.
type ExpirationModel struct {
	Type        types.String `tfsdk:"type"`
	EndDateTime types.String `tfsdk:"end_date_time"`
	Duration    types.String `tfsdk:"duration"`
}

func (r *DirectoryRoleEligibleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_role_eligible_assignment"
}

func (r *DirectoryRoleEligibleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an eligible assignment for an Entra ID directory role using Privileged Identity Management (PIM). " +
			"\n\n**Conceptual Model:** This resource represents a PIM role eligibility *schedule* (the durable object), not the schedule request (the transient operation). " +
			"The schedule is uniquely identified by the combination of principal_id, role_definition_id, and directory_scope_id. " +
			"The resource ID is the schedule ID returned by Microsoft Graph. " +
			"\n\n**Operations:** Under the hood, this provider uses adminAssign (create), adminUpdate (update), and adminRemove (delete) requests to manage the schedule. " +
			"These requests are asynchronous operations that converge the schedule to the desired state. " +
			"\n\n**Import:** If a schedule already exists for a given principal/role/scope combination, you must import it rather than creating a new one. " +
			"Use: `terraform import msgraph-entra_directory_role_eligible_assignment.example <schedule_id>`",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the role eligibility schedule (unifiedRoleEligibilitySchedule). This is the durable schedule object, not the transient request ID.",
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
				MarkdownDescription: "Justification for the role assignment request. This is a write-only field sent with create/update requests but not persisted on the schedule. Imported resources will have this field set to null.",
				Optional:            true,
			},
			"schedule_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the created role eligibility schedule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"schedule_info": schema.SingleNestedBlock{
				MarkdownDescription: "The schedule of the role eligibility.",
				Attributes: map[string]schema.Attribute{
					"start_date_time": schema.StringAttribute{
						MarkdownDescription: "When the eligibility starts. Must be in RFC3339 format (e.g., '2025-01-08T00:00:00Z'). Defaults to now. Note: Azure may adjust this value slightly.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"expiration": schema.SingleNestedBlock{
						MarkdownDescription: "When and how the eligibility expires.",
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								MarkdownDescription: "The type of expiration: 'noExpiration', 'afterDateTime', or 'afterDuration'. Note: Azure may convert 'afterDuration' to 'afterDateTime'.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.String{
									stringvalidator.OneOf("noExpiration", "afterDateTime", "afterDuration"),
								},
							},
							"end_date_time": schema.StringAttribute{
								MarkdownDescription: "The end date/time when type is 'afterDateTime'. Must be in RFC3339 format. Note: Azure may adjust this value slightly.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"duration": schema.StringAttribute{
								MarkdownDescription: "The duration when type is 'afterDuration'. Use ISO 8601 duration format (e.g., 'PT8H' for 8 hours, 'P365D' for 365 days). Note: Azure often converts this to an end_date_time.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
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
	// If an assignment already exists, the API will return an error directing users to import it
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
		if !data.ScheduleInfo.StartDateTime.IsNull() && data.ScheduleInfo.StartDateTime.ValueString() != "" {
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

			if !data.ScheduleInfo.Expiration.EndDateTime.IsNull() && data.ScheduleInfo.Expiration.EndDateTime.ValueString() != "" {
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

			if !data.ScheduleInfo.Expiration.Duration.IsNull() && data.ScheduleInfo.Expiration.Duration.ValueString() != "" {
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
		// Check if the error indicates a schedule already exists
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "RoleAssignmentExists") {
			// Try to find the existing schedule to provide helpful import guidance
			existingSchedule, findErr := r.client.FindRoleEligibilitySchedule(ctx, principalID, roleDefID, dirScopeID)
			if findErr == nil && existingSchedule != nil && existingSchedule.GetId() != nil {
				scheduleID := *existingSchedule.GetId()
				resp.Diagnostics.AddError(
					"Role Assignment Already Exists",
					fmt.Sprintf("An eligible role assignment already exists for this principal/role/scope combination.\n\n"+
						"Schedule ID: %s\n\n"+
						"To manage this existing assignment with Terraform, import it using:\n"+
						"  terraform import msgraph-entra_directory_role_eligible_assignment.<name> %s\n\n"+
						"Or remove the existing assignment in the Azure Portal before creating a new one.",
						scheduleID, scheduleID),
				)
			} else {
				// Could not find existing schedule, provide generic guidance
				resp.Diagnostics.AddError(
					"Role Assignment Already Exists",
					"An eligible role assignment already exists for this principal/role/scope combination.\n\n"+
						"To resolve this:\n"+
						"1. Check the Azure Portal for existing eligible assignments\n"+
						"2. Import the existing assignment using: terraform import msgraph-entra_directory_role_eligible_assignment.<name> <schedule_id>\n"+
						"3. Or remove the existing assignment before creating a new one\n\n"+
						"Original error: "+err.Error(),
				)
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Creating Role Eligible Assignment",
				"Could not create role eligible assignment: "+err.Error(),
			)
		}
		return
	}

	// The request ID is transient - we'll set the actual resource ID after we get the schedule
	tflog.Debug(ctx, "Created schedule request", map[string]any{
		"request_id": *result.GetId(),
	})

	// Wait for the schedule to be created (the request is processed asynchronously)
	// Retry for up to 30 seconds
	var schedule models.UnifiedRoleEligibilityScheduleable
	maxRetries := 15
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			resp.Diagnostics.AddError(
				"Context Cancelled",
				fmt.Sprintf("Operation was cancelled: %s", ctx.Err().Error()),
			)
			return
		}

		if i > 0 {
			tflog.Debug(ctx, "Waiting for schedule to be created", map[string]any{
				"attempt": i + 1,
				"max":     maxRetries,
			})
			time.Sleep(retryDelay)
		}

		schedule, err = r.client.FindRoleEligibilitySchedule(ctx, principalID, roleDefID, dirScopeID)
		if err == nil && schedule != nil {
			break
		}
	}

	if schedule == nil {
		resp.Diagnostics.AddError(
			"Schedule Creation Timeout",
			fmt.Sprintf("The role eligibility schedule request was created (ID: %s) but the schedule did not appear after %d seconds. "+
				"This may indicate Azure is experiencing delays. Please wait a moment and try again.",
				*result.GetId(), int(retryDelay.Seconds()*float64(maxRetries))),
		)
		return
	}

	// Successfully found the schedule - populate all fields from the actual schedule
	// Use the schedule ID as the resource ID (not the request ID)
	if schedule.GetId() != nil {
		scheduleID := types.StringPointerValue(schedule.GetId())
		data.ID = scheduleID
		data.ScheduleID = scheduleID
	}

	// Update schedule info from the actual schedule
	if schedule.GetScheduleInfo() != nil {
		scheduleInfo := schedule.GetScheduleInfo()
		if data.ScheduleInfo == nil {
			data.ScheduleInfo = &ScheduleInfoModel{}
		}

		if scheduleInfo.GetStartDateTime() != nil {
			data.ScheduleInfo.StartDateTime = types.StringValue(scheduleInfo.GetStartDateTime().Format(time.RFC3339))
		}

		if scheduleInfo.GetExpiration() != nil {
			expiration := scheduleInfo.GetExpiration()
			if data.ScheduleInfo.Expiration == nil {
				data.ScheduleInfo.Expiration = &ExpirationModel{}
			}

			if expiration.GetTypeEscaped() != nil {
				switch *expiration.GetTypeEscaped() {
				case models.NOEXPIRATION_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("noExpiration")
				case models.AFTERDATETIME_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("afterDateTime")
				case models.AFTERDURATION_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("afterDuration")
				}
			}

			if expiration.GetEndDateTime() != nil {
				data.ScheduleInfo.Expiration.EndDateTime = types.StringValue(expiration.GetEndDateTime().Format(time.RFC3339))
			} else {
				data.ScheduleInfo.Expiration.EndDateTime = types.StringNull()
			}

			if expiration.GetDuration() != nil {
				data.ScheduleInfo.Expiration.Duration = types.StringValue(expiration.GetDuration().String())
			} else {
				data.ScheduleInfo.Expiration.Duration = types.StringNull()
			}
		}
	}

	tflog.Info(ctx, "Schedule created and available", map[string]any{
		"schedule_id": data.ScheduleID.ValueString(),
	})

	tflog.Trace(ctx, "Created directory role eligible assignment", map[string]any{
		"id":          data.ID.ValueString(),
		"schedule_id": data.ScheduleID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the actual schedule (persistent object) instead of the schedule request (transient operation)
	// This provides better drift detection and state management
	var schedule models.UnifiedRoleEligibilityScheduleable
	var err error

	// The resource ID is the schedule ID, try to look it up directly first
	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		// Try to get by ID (schedule ID) first (most efficient)
		schedule, err = r.client.GetRoleEligibilitySchedule(ctx, data.ID.ValueString())
		if err != nil {
			// ID lookup failed - if we have principal/role/scope info, fall back to searching
			if !data.PrincipalID.IsNull() && !data.RoleDefinitionID.IsNull() && !data.DirectoryScopeID.IsNull() {
				tflog.Warn(ctx, "Failed to get schedule by ID, searching by principal+role+scope", map[string]any{
					"id":    data.ID.ValueString(),
					"error": err.Error(),
				})
				schedule, err = r.client.FindRoleEligibilitySchedule(
					ctx,
					data.PrincipalID.ValueString(),
					data.RoleDefinitionID.ValueString(),
					data.DirectoryScopeID.ValueString(),
				)
			}
		}
	} else if !data.PrincipalID.IsNull() && !data.RoleDefinitionID.IsNull() && !data.DirectoryScopeID.IsNull() {
		// No ID, search by principal+role+scope
		schedule, err = r.client.FindRoleEligibilitySchedule(
			ctx,
			data.PrincipalID.ValueString(),
			data.RoleDefinitionID.ValueString(),
			data.DirectoryScopeID.ValueString(),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role Eligible Assignment",
			"Could not read role eligible assignment: "+err.Error(),
		)
		return
	}

	if schedule == nil {
		// Schedule no longer exists - remove from state
		tflog.Info(ctx, "Role eligible assignment no longer exists, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the state with the current schedule information
	if schedule.GetId() != nil {
		scheduleID := types.StringPointerValue(schedule.GetId())
		data.ID = scheduleID
		data.ScheduleID = scheduleID
	}

	// Populate principal, role, and scope from the schedule (needed for import and drift detection)
	if schedule.GetPrincipalId() != nil {
		data.PrincipalID = types.StringPointerValue(schedule.GetPrincipalId())
	}
	if schedule.GetRoleDefinitionId() != nil {
		data.RoleDefinitionID = types.StringPointerValue(schedule.GetRoleDefinitionId())
	}
	if schedule.GetDirectoryScopeId() != nil {
		data.DirectoryScopeID = types.StringPointerValue(schedule.GetDirectoryScopeId())
	}

	// Update schedule info from the actual schedule
	if schedule.GetScheduleInfo() != nil {
		scheduleInfo := schedule.GetScheduleInfo()
		if data.ScheduleInfo == nil {
			data.ScheduleInfo = &ScheduleInfoModel{}
		}

		if scheduleInfo.GetStartDateTime() != nil {
			data.ScheduleInfo.StartDateTime = types.StringValue(scheduleInfo.GetStartDateTime().Format(time.RFC3339))
		}

		if scheduleInfo.GetExpiration() != nil {
			expiration := scheduleInfo.GetExpiration()
			if data.ScheduleInfo.Expiration == nil {
				data.ScheduleInfo.Expiration = &ExpirationModel{}
			}

			if expiration.GetTypeEscaped() != nil {
				switch *expiration.GetTypeEscaped() {
				case models.NOEXPIRATION_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("noExpiration")
				case models.AFTERDATETIME_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("afterDateTime")
				case models.AFTERDURATION_EXPIRATIONPATTERNTYPE:
					data.ScheduleInfo.Expiration.Type = types.StringValue("afterDuration")
				}
			}

			if expiration.GetEndDateTime() != nil {
				data.ScheduleInfo.Expiration.EndDateTime = types.StringValue(expiration.GetEndDateTime().Format(time.RFC3339))
			} else {
				data.ScheduleInfo.Expiration.EndDateTime = types.StringNull()
			}

			if expiration.GetDuration() != nil {
				data.ScheduleInfo.Expiration.Duration = types.StringValue(expiration.GetDuration().String())
			} else {
				data.ScheduleInfo.Expiration.Duration = types.StringNull()
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DirectoryRoleEligibleAssignmentResourceModel
	var state DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Use adminUpdate action to modify the existing assignment in-place
	// This avoids breaking user access during the update
	tflog.Info(ctx, "Updating directory role eligible assignment", map[string]any{
		"schedule_id":        state.ScheduleID.ValueString(),
		"role_definition_id": plan.RoleDefinitionID.ValueString(),
		"principal_id":       plan.PrincipalID.ValueString(),
	})

	// Create an update request using adminUpdate action
	request := models.NewUnifiedRoleEligibilityScheduleRequest()

	action := models.ADMINUPDATE_UNIFIEDROLESCHEDULEREQUESTACTIONS
	request.SetAction(&action)

	roleDefID := plan.RoleDefinitionID.ValueString()
	request.SetRoleDefinitionId(&roleDefID)

	principalID := plan.PrincipalID.ValueString()
	request.SetPrincipalId(&principalID)

	dirScopeID := plan.DirectoryScopeID.ValueString()
	request.SetDirectoryScopeId(&dirScopeID)

	// Set the target schedule ID to update the existing schedule
	scheduleID := state.ScheduleID.ValueString()
	request.SetTargetScheduleId(&scheduleID)

	if !plan.Justification.IsNull() {
		justification := plan.Justification.ValueString()
		request.SetJustification(&justification)
	}

	// Set updated schedule info
	if plan.ScheduleInfo != nil {
		scheduleInfo := models.NewRequestSchedule()

		// Start date time
		// If start_date_time is specified in plan, use it. Otherwise, preserve from state.
		// We don't default to time.Now() unless neither plan nor state has a value.
		var startDateTime time.Time
		if !plan.ScheduleInfo.StartDateTime.IsNull() && plan.ScheduleInfo.StartDateTime.ValueString() != "" {
			// User explicitly set start_date_time in plan - use it
			var err error
			startDateTime, err = time.Parse(time.RFC3339, plan.ScheduleInfo.StartDateTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Start Date Time",
					fmt.Sprintf("Could not parse start_date_time: %s", err.Error()),
				)
				return
			}
		} else if state.ScheduleInfo != nil && !state.ScheduleInfo.StartDateTime.IsNull() && state.ScheduleInfo.StartDateTime.ValueString() != "" {
			// Preserve start_date_time from state (don't reset to now on update)
			var err error
			startDateTime, err = time.Parse(time.RFC3339, state.ScheduleInfo.StartDateTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Start Date Time from State",
					fmt.Sprintf("Could not parse start_date_time from state: %s", err.Error()),
				)
				return
			}
		} else {
			// Neither plan nor state has a value - default to now
			startDateTime = time.Now().UTC()
		}
		scheduleInfo.SetStartDateTime(&startDateTime)

		// Expiration
		if plan.ScheduleInfo.Expiration != nil {
			expiration := models.NewExpirationPattern()

			if !plan.ScheduleInfo.Expiration.Type.IsNull() {
				expTypeStr := plan.ScheduleInfo.Expiration.Type.ValueString()
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

			if !plan.ScheduleInfo.Expiration.EndDateTime.IsNull() && plan.ScheduleInfo.Expiration.EndDateTime.ValueString() != "" {
				endDateTime, err := time.Parse(time.RFC3339, plan.ScheduleInfo.Expiration.EndDateTime.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Invalid End Date Time",
						fmt.Sprintf("Could not parse end_date_time: %s", err.Error()),
					)
					return
				}
				expiration.SetEndDateTime(&endDateTime)
			}

			if !plan.ScheduleInfo.Expiration.Duration.IsNull() && plan.ScheduleInfo.Expiration.Duration.ValueString() != "" {
				durationStr := plan.ScheduleInfo.Expiration.Duration.ValueString()
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

	// Submit the update request
	result, err := r.client.CreateRoleEligibilityScheduleRequest(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Role Eligible Assignment",
			"Could not update role eligible assignment: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Role eligibility schedule update request created", map[string]any{
		"request_id": result.GetId(),
	})

	// The result is the request, not the schedule.
	// We need to wait for the request to be processed and then read the schedule.
	// IMPORTANT: We don't touch plan.ID or plan.ScheduleID here - they remain as schedule IDs from state.
	// The request ID is logged above but not stored in state.

	tflog.Trace(ctx, "Updated directory role eligible assignment", map[string]any{
		"id":          plan.ID.ValueString(),
		"schedule_id": plan.ScheduleID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DirectoryRoleEligibleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DirectoryRoleEligibleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the role eligibility
	tflog.Info(ctx, "Deleting directory role eligible assignment", map[string]any{
		"id":          data.ID.ValueString(),
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
		// If the assignment doesn't exist, that's actually success for a delete operation
		errMsg := err.Error()
		if strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "NotFound") {
			tflog.Warn(ctx, "Role assignment already deleted or not found", map[string]any{
				"schedule_id": data.ScheduleID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Role Eligible Assignment",
			"Could not delete role eligible assignment: "+err.Error(),
		)
		return
	}

	// Wait for the schedule to be deleted (adminRemove is processed asynchronously)
	// Poll for up to 30 seconds to verify the schedule is gone
	tflog.Debug(ctx, "Submitted delete request, waiting for schedule to be removed")
	maxRetries := 15
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			resp.Diagnostics.AddError(
				"Context Cancelled",
				fmt.Sprintf("Operation was cancelled: %s", ctx.Err().Error()),
			)
			return
		}

		if i > 0 {
			tflog.Debug(ctx, "Checking if schedule is deleted", map[string]any{
				"attempt": i + 1,
				"max":     maxRetries,
			})
			time.Sleep(retryDelay)
		}

		// Try to find the schedule - if it's gone, we're done
		schedule, err := r.client.FindRoleEligibilitySchedule(
			ctx,
			data.PrincipalID.ValueString(),
			data.RoleDefinitionID.ValueString(),
			data.DirectoryScopeID.ValueString(),
		)

		if err != nil {
			// Error querying - might be transient, continue retrying
			tflog.Debug(ctx, "Error checking schedule status", map[string]any{
				"error": err.Error(),
			})
			continue
		}

		if schedule == nil {
			// Schedule is gone - success!
			tflog.Info(ctx, "Schedule successfully deleted and verified")
			return
		}

		tflog.Debug(ctx, "Schedule still exists, continuing to wait")
	}

	// Schedule still exists after timeout - warn but don't fail
	// The delete was submitted successfully, it just might take longer to process
	tflog.Warn(ctx, "Delete request submitted but schedule still exists after 30 seconds", map[string]any{
		"schedule_id": data.ScheduleID.ValueString(),
	})
	resp.Diagnostics.AddWarning(
		"Delete Processing",
		fmt.Sprintf("The delete request was submitted successfully, but the schedule (ID: %s) is still present after 30 seconds. "+
			"Azure may still be processing the deletion. The schedule should be removed shortly.",
			data.ScheduleID.ValueString()),
	)
}

func (r *DirectoryRoleEligibleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
