// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

var (
	_ datasource.DataSource                   = (*roleDataSource)(nil)
	_ datasource.DataSourceWithConfigure      = (*roleDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*roleDataSource)(nil)
)

// NewRoleDataSource returns a datasource.DataSource factory for discord_role.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *client.Client
}

type roleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	GuildID     types.String `tfsdk:"guild_id"`
	Name        types.String `tfsdk:"name"`
	Color       types.Int64  `tfsdk:"color"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Permissions types.String `tfsdk:"permissions"`
	Position    types.Int64  `tfsdk:"position"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Discord role by ID or name. Exactly one of `id` or `name` must be set.",
		Attributes: map[string]schema.Attribute{
			"guild_id": schema.StringAttribute{
				Description: "Guild to search within.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Role ID (snowflake). If unset, the role is looked up by `name`.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Role name. If unset, the role is looked up by `id`.",
				Optional:    true,
				Computed:    true,
			},
			"color": schema.Int64Attribute{Computed: true, Description: "Role color as a decimal RGB integer."},
			"hoist": schema.BoolAttribute{Computed: true, Description: "Whether the role is displayed separately in the member list."},
			"mentionable": schema.BoolAttribute{Computed: true, Description: "Whether the role is mentionable."},
			"permissions": schema.StringAttribute{Computed: true, Description: "Permission bitfield as a decimal string."},
			"position":    schema.Int64Attribute{Computed: true, Description: "Position in the role hierarchy."},
		},
	}
}

func (d *roleDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.Session.GuildRoles(data.GuildID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("List roles failed", err.Error())
		return
	}

	var match *discordgo.Role
	switch {
	case !data.ID.IsNull() && !data.ID.IsUnknown():
		for _, role := range roles {
			if role.ID == data.ID.ValueString() {
				match = role
				break
			}
		}
	case !data.Name.IsNull() && !data.Name.IsUnknown():
		for _, role := range roles {
			if role.Name == data.Name.ValueString() {
				if match != nil {
					resp.Diagnostics.AddError(
						"Ambiguous role name",
						fmt.Sprintf("Multiple roles named %q exist in guild %s. Look up by id instead.", data.Name.ValueString(), data.GuildID.ValueString()),
					)
					return
				}
				match = role
			}
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("Role not found", "No role matched the given id/name in the guild.")
		return
	}

	data.ID = types.StringValue(match.ID)
	data.Name = types.StringValue(match.Name)
	data.Color = types.Int64Value(int64(match.Color))
	data.Hoist = types.BoolValue(match.Hoist)
	data.Mentionable = types.BoolValue(match.Mentionable)
	data.Permissions = types.StringValue(strconv.FormatInt(match.Permissions, 10))
	data.Position = types.Int64Value(int64(match.Position))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
