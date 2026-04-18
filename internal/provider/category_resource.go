// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*categoryResource)(nil)
	_ resource.ResourceWithConfigure   = (*categoryResource)(nil)
	_ resource.ResourceWithImportState = (*categoryResource)(nil)
)

// NewCategoryResource returns a resource.Resource factory for discord_category.
func NewCategoryResource() resource.Resource {
	return &categoryResource{}
}

type categoryResource struct {
	client *client.Client
}

type categoryResourceModel struct {
	ID       types.String `tfsdk:"id"`
	GuildID  types.String `tfsdk:"guild_id"`
	Name     types.String `tfsdk:"name"`
	Position types.Int64  `tfsdk:"position"`
}

func (r *categoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category"
}

func (r *categoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord channel category. Categories group channels together in the guild sidebar; a channel's `parent_id` must reference a category's ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Category channel ID (snowflake).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"guild_id": schema.StringAttribute{
				Description: "ID of the guild that owns this category.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Category name (1–100 chars).",
				Required:    true,
			},
			"position": schema.Int64Attribute{
				Description: "Sort position among categories. If omitted, Discord assigns the next available position.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *categoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *categoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan categoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := discordgo.GuildChannelCreateData{
		Name: plan.Name.ValueString(),
		Type: discordgo.ChannelTypeGuildCategory,
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		data.Position = int(plan.Position.ValueInt64())
	}

	channel, err := r.client.Session.GuildChannelCreateComplex(plan.GuildID.ValueString(), data, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create category failed", err.Error())
		return
	}

	categoryToState(channel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *categoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state categoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channel, err := r.client.Session.Channel(state.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read category failed", err.Error())
		return
	}
	if channel.Type != discordgo.ChannelTypeGuildCategory {
		resp.Diagnostics.AddError(
			"Channel is not a category",
			fmt.Sprintf("Channel %s has type %d, expected %d (category).", channel.ID, channel.Type, discordgo.ChannelTypeGuildCategory),
		)
		return
	}

	categoryToState(channel, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *categoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state categoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	edit := &discordgo.ChannelEdit{Name: plan.Name.ValueString()}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		pos := int(plan.Position.ValueInt64())
		edit.Position = &pos
	}

	channel, err := r.client.Session.ChannelEditComplex(state.ID.ValueString(), edit, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Update category failed", err.Error())
		return
	}

	categoryToState(channel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *categoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state categoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Session.ChannelDelete(state.ID.ValueString(), discordgo.WithContext(ctx)); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete category failed", err.Error())
	}
}

func (r *categoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	guildID, channelID, ok := strings.Cut(req.ID, ":")
	if !ok || guildID == "" || channelID == "" {
		resp.Diagnostics.AddError("Invalid import ID", `Expected "<guild_id>:<channel_id>", got: `+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("guild_id"), guildID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), channelID)...)
}

func categoryToState(c *discordgo.Channel, m *categoryResourceModel) {
	m.ID = types.StringValue(c.ID)
	m.GuildID = types.StringValue(c.GuildID)
	m.Name = types.StringValue(c.Name)
	m.Position = types.Int64Value(int64(c.Position))
}
