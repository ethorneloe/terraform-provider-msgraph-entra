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
var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *GraphClient
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	UserPrincipalName types.String `tfsdk:"user_principal_name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Mail              types.String `tfsdk:"mail"`
	MailNickname      types.String `tfsdk:"mail_nickname"`
	GivenName         types.String `tfsdk:"given_name"`
	Surname           types.String `tfsdk:"surname"`
	JobTitle          types.String `tfsdk:"job_title"`
	Department        types.String `tfsdk:"department"`
	AccountEnabled    types.Bool   `tfsdk:"account_enabled"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an Entra ID user by their User Principal Name (UPN) or object ID. This data source is useful for looking up user object IDs needed for role assignments.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The object ID (GUID) of the user in Azure AD.",
				Computed:            true,
			},
			"user_principal_name": schema.StringAttribute{
				MarkdownDescription: "The user principal name (UPN) of the user, typically their email address (e.g., 'john.doe@contoso.com').",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the user.",
				Computed:            true,
			},
			"mail": schema.StringAttribute{
				MarkdownDescription: "The primary email address of the user.",
				Computed:            true,
			},
			"mail_nickname": schema.StringAttribute{
				MarkdownDescription: "The mail alias for the user.",
				Computed:            true,
			},
			"given_name": schema.StringAttribute{
				MarkdownDescription: "The given name (first name) of the user.",
				Computed:            true,
			},
			"surname": schema.StringAttribute{
				MarkdownDescription: "The surname (last name) of the user.",
				Computed:            true,
			},
			"job_title": schema.StringAttribute{
				MarkdownDescription: "The user's job title.",
				Computed:            true,
			},
			"department": schema.StringAttribute{
				MarkdownDescription: "The department in which the user works.",
				Computed:            true,
			},
			"account_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the account is enabled.",
				Computed:            true,
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	upn := data.UserPrincipalName.ValueString()

	tflog.Info(ctx, "Looking up user", map[string]any{
		"user_principal_name": upn,
	})

	// Get the user
	user, err := d.client.GetUser(ctx, upn)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User",
			fmt.Sprintf("Could not find user '%s': %s", upn, err.Error()),
		)
		return
	}

	// Set the data
	if user.GetId() != nil {
		data.ID = types.StringPointerValue(user.GetId())
	}

	if user.GetDisplayName() != nil {
		data.DisplayName = types.StringPointerValue(user.GetDisplayName())
	} else {
		data.DisplayName = types.StringNull()
	}

	if user.GetMail() != nil {
		data.Mail = types.StringPointerValue(user.GetMail())
	} else {
		data.Mail = types.StringNull()
	}

	if user.GetMailNickname() != nil {
		data.MailNickname = types.StringPointerValue(user.GetMailNickname())
	} else {
		data.MailNickname = types.StringNull()
	}

	if user.GetGivenName() != nil {
		data.GivenName = types.StringPointerValue(user.GetGivenName())
	} else {
		data.GivenName = types.StringNull()
	}

	if user.GetSurname() != nil {
		data.Surname = types.StringPointerValue(user.GetSurname())
	} else {
		data.Surname = types.StringNull()
	}

	if user.GetJobTitle() != nil {
		data.JobTitle = types.StringPointerValue(user.GetJobTitle())
	} else {
		data.JobTitle = types.StringNull()
	}

	if user.GetDepartment() != nil {
		data.Department = types.StringPointerValue(user.GetDepartment())
	} else {
		data.Department = types.StringNull()
	}

	if user.GetAccountEnabled() != nil {
		data.AccountEnabled = types.BoolPointerValue(user.GetAccountEnabled())
	} else {
		data.AccountEnabled = types.BoolNull()
	}

	tflog.Trace(ctx, "Read user", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
