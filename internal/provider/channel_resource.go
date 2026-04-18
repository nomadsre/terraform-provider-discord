// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*channelResource)(nil)
	_ resource.ResourceWithConfigure   = (*channelResource)(nil)
	_ resource.ResourceWithImportState = (*channelResource)(nil)
)

// channelTypes maps operator-friendly type names to Discord's numeric
// ChannelType. Category is not included here — it has its own resource.
var channelTypes = map[string]discordgo.ChannelType{
	"text":  discordgo.ChannelTypeGuildText,
	"voice": discordgo.ChannelTypeGuildVoice,
	"news":  discordgo.ChannelTypeGuildNews,
	"stage": discordgo.ChannelTypeGuildStageVoice,
	"forum": discordgo.ChannelTypeGuildForum,
}

func channelTypeNames() []string {
	out := make([]string, 0, len(channelTypes))
	for name := range channelTypes {
		out = append(out, name)
	}
	return out
}

func channelTypeName(t discordgo.ChannelType) string {
	for name, v := range channelTypes {
		if v == t {
			return name
		}
	}
	return ""
}

// NewChannelResource returns a resource.Resource factory for discord_channel.
func NewChannelResource() resource.Resource {
	return &channelResource{}
}

type channelResource struct {
	client *client.Client
}

type channelResourceModel struct {
	ID               types.String `tfsdk:"id"`
	GuildID          types.String `tfsdk:"guild_id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Topic            types.String `tfsdk:"topic"`
	NSFW             types.Bool   `tfsdk:"nsfw"`
	ParentID         types.String `tfsdk:"parent_id"`
	Position         types.Int64  `tfsdk:"position"`
	RateLimitPerUser types.Int64  `tfsdk:"rate_limit_per_user"`
	Bitrate          types.Int64  `tfsdk:"bitrate"`
	UserLimit        types.Int64  `tfsdk:"user_limit"`
}

func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord guild channel. Use `discord_category` for category (grouping) channels.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Channel ID (snowflake).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"guild_id": schema.StringAttribute{
				Description: "ID of the guild that owns this channel.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Channel type: `text`, `voice`, `news`, `stage`, or `forum`. Not all attributes apply to every type; fields that do not apply are silently ignored by the Discord API.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(channelTypeNames()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Channel name (1–100 chars).",
				Required:    true,
			},
			"topic": schema.StringAttribute{
				Description: "Channel topic (text/news/forum only).",
				Optional:    true,
				Computed:    true,
			},
			"nsfw": schema.BoolAttribute{
				Description: "Whether the channel is marked NSFW (age-restricted).",
				Optional:    true,
				Computed:    true,
			},
			"parent_id": schema.StringAttribute{
				Description: "ID of the parent category. Null for top-level channels.",
				Optional:    true,
				Computed:    true,
			},
			"position": schema.Int64Attribute{
				Description: "Sort position within the parent. If omitted, Discord assigns the next available position.",
				Optional:    true,
				Computed:    true,
			},
			"rate_limit_per_user": schema.Int64Attribute{
				Description: "Per-user slow-mode delay in seconds (text/forum only; 0–21600).",
				Optional:    true,
				Computed:    true,
			},
			"bitrate": schema.Int64Attribute{
				Description: "Voice bitrate in bits/sec (voice/stage only).",
				Optional:    true,
				Computed:    true,
			},
			"user_limit": schema.Int64Attribute{
				Description: "Max concurrent users (voice/stage only; 0 = unlimited).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := discordgo.GuildChannelCreateData{
		Name: plan.Name.ValueString(),
		Type: channelTypes[plan.Type.ValueString()],
	}
	if !plan.Topic.IsNull() && !plan.Topic.IsUnknown() {
		data.Topic = plan.Topic.ValueString()
	}
	if !plan.NSFW.IsNull() && !plan.NSFW.IsUnknown() {
		data.NSFW = plan.NSFW.ValueBool()
	}
	if !plan.ParentID.IsNull() && !plan.ParentID.IsUnknown() {
		data.ParentID = plan.ParentID.ValueString()
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		data.Position = int(plan.Position.ValueInt64())
	}
	if !plan.RateLimitPerUser.IsNull() && !plan.RateLimitPerUser.IsUnknown() {
		data.RateLimitPerUser = int(plan.RateLimitPerUser.ValueInt64())
	}
	if !plan.Bitrate.IsNull() && !plan.Bitrate.IsUnknown() {
		data.Bitrate = int(plan.Bitrate.ValueInt64())
	}
	if !plan.UserLimit.IsNull() && !plan.UserLimit.IsUnknown() {
		data.UserLimit = int(plan.UserLimit.ValueInt64())
	}

	channel, err := r.client.Session.GuildChannelCreateComplex(plan.GuildID.ValueString(), data, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create channel failed", err.Error())
		return
	}

	channelToState(channel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelResourceModel
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
		resp.Diagnostics.AddError("Read channel failed", err.Error())
		return
	}

	channelToState(channel, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	edit := &discordgo.ChannelEdit{Name: plan.Name.ValueString()}
	if !plan.Topic.IsNull() && !plan.Topic.IsUnknown() {
		edit.Topic = plan.Topic.ValueString()
	}
	if !plan.NSFW.IsNull() && !plan.NSFW.IsUnknown() {
		v := plan.NSFW.ValueBool()
		edit.NSFW = &v
	}
	if !plan.ParentID.IsNull() && !plan.ParentID.IsUnknown() {
		edit.ParentID = plan.ParentID.ValueString()
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		pos := int(plan.Position.ValueInt64())
		edit.Position = &pos
	}
	if !plan.RateLimitPerUser.IsNull() && !plan.RateLimitPerUser.IsUnknown() {
		v := int(plan.RateLimitPerUser.ValueInt64())
		edit.RateLimitPerUser = &v
	}
	if !plan.Bitrate.IsNull() && !plan.Bitrate.IsUnknown() {
		edit.Bitrate = int(plan.Bitrate.ValueInt64())
	}
	if !plan.UserLimit.IsNull() && !plan.UserLimit.IsUnknown() {
		edit.UserLimit = int(plan.UserLimit.ValueInt64())
	}

	channel, err := r.client.Session.ChannelEditComplex(state.ID.ValueString(), edit, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Update channel failed", err.Error())
		return
	}

	channelToState(channel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Session.ChannelDelete(state.ID.ValueString(), discordgo.WithContext(ctx)); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete channel failed", err.Error())
	}
}

func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	guildID, channelID, ok := strings.Cut(req.ID, ":")
	if !ok || guildID == "" || channelID == "" {
		resp.Diagnostics.AddError("Invalid import ID", `Expected "<guild_id>:<channel_id>", got: `+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("guild_id"), guildID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), channelID)...)
}

func channelToState(c *discordgo.Channel, m *channelResourceModel) {
	m.ID = types.StringValue(c.ID)
	m.GuildID = types.StringValue(c.GuildID)
	if name := channelTypeName(c.Type); name != "" {
		m.Type = types.StringValue(name)
	}
	m.Name = types.StringValue(c.Name)
	m.Topic = types.StringValue(c.Topic)
	m.NSFW = types.BoolValue(c.NSFW)
	if c.ParentID == "" {
		m.ParentID = types.StringNull()
	} else {
		m.ParentID = types.StringValue(c.ParentID)
	}
	m.Position = types.Int64Value(int64(c.Position))
	m.RateLimitPerUser = types.Int64Value(int64(c.RateLimitPerUser))
	m.Bitrate = types.Int64Value(int64(c.Bitrate))
	m.UserLimit = types.Int64Value(int64(c.UserLimit))
}
