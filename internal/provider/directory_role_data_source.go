// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DirectoryRoleDataSource{}

func NewDirectoryRoleDataSource() datasource.DataSource {
	return &DirectoryRoleDataSource{}
}

// DirectoryRoleDataSource defines the data source implementation.
type DirectoryRoleDataSource struct {
	client *GraphClient
}

// DirectoryRoleDataSourceModel describes the data source data model.
type DirectoryRoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	TemplateID  types.String `tfsdk:"template_id"`
	Description types.String `tfsdk:"description"`
}

func (d *DirectoryRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_role"
}

func (d *DirectoryRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an Entra ID directory role. Use this to get the template ID needed for role assignments.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The object ID of the directory role instance.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the directory role (e.g., 'Global Administrator', 'Security Administrator').",
				Required:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "The template ID of the directory role. This is the ID used for role assignments.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the directory role.",
				Computed:            true,
			},
		},
	}
}

func (d *DirectoryRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*GraphClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *GraphClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *DirectoryRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DirectoryRoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	displayName := data.DisplayName.ValueString()

	tflog.Info(ctx, "Looking up directory role", map[string]any{
		"display_name": displayName,
	})

	// Get the directory role
	role, err := d.client.GetDirectoryRole(ctx, displayName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Directory Role",
			fmt.Sprintf("Could not find directory role '%s': %s", displayName, err.Error()),
		)
		return
	}

	// Set the data
	if role.GetId() != nil {
		data.ID = types.StringPointerValue(role.GetId())
	}

	if role.GetRoleTemplateId() != nil {
		data.TemplateID = types.StringPointerValue(role.GetRoleTemplateId())
	}

	if role.GetDescription() != nil {
		data.Description = types.StringPointerValue(role.GetDescription())
	}

	tflog.Trace(ctx, "Read directory role", map[string]any{
		"template_id": data.TemplateID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
