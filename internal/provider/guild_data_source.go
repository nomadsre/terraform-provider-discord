// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

var (
	_ datasource.DataSource              = (*guildDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*guildDataSource)(nil)
)

// NewGuildDataSource returns a datasource.DataSource factory for discord_guild.
func NewGuildDataSource() datasource.DataSource {
	return &guildDataSource{}
}

type guildDataSource struct {
	client *client.Client
}

type guildDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	OwnerID  types.String `tfsdk:"owner_id"`
	Icon     types.String `tfsdk:"icon"`
	Features types.List   `tfsdk:"features"`
}

func (d *guildDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild"
}

func (d *guildDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a Discord guild by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Guild ID (snowflake).",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Guild name.",
				Computed:    true,
			},
			"owner_id": schema.StringAttribute{
				Description: "User ID of the guild owner.",
				Computed:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Hash of the guild's icon, or empty if no icon is set.",
				Computed:    true,
			},
			"features": schema.ListAttribute{
				Description: "List of guild feature flags (e.g. `COMMUNITY`, `VERIFIED`).",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *guildDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *guildDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data guildDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	guild, err := d.client.Session.Guild(data.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Read guild failed", err.Error())
		return
	}

	features := make([]string, len(guild.Features))
	for i, f := range guild.Features {
		features[i] = string(f)
	}
	featuresList, diags := types.ListValueFrom(ctx, types.StringType, features)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Name = types.StringValue(guild.Name)
	data.OwnerID = types.StringValue(guild.OwnerID)
	data.Icon = types.StringValue(guild.Icon)
	data.Features = featuresList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
