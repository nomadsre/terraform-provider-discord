// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = (*webhookResource)(nil)
	_ resource.ResourceWithConfigure   = (*webhookResource)(nil)
	_ resource.ResourceWithImportState = (*webhookResource)(nil)
)

// NewWebhookResource returns a resource.Resource factory for discord_webhook.
func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	client *client.Client
}

type webhookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	GuildID   types.String `tfsdk:"guild_id"`
	Name      types.String `tfsdk:"name"`
	Avatar    types.String `tfsdk:"avatar"`
	URL       types.String `tfsdk:"url"`
	Token     types.String `tfsdk:"token"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord channel webhook. The `url` and `token` attributes are sensitive — anyone holding them can post as the webhook without further authentication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Webhook ID (snowflake).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_id": schema.StringAttribute{
				Description: "ID of the channel the webhook posts to. Changing this moves the webhook; no replacement required.",
				Required:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "Guild that contains the webhook's channel.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Webhook display name.",
				Required:    true,
			},
			"avatar": schema.StringAttribute{
				Description: "Default avatar for the webhook as a base64-encoded image data URI (e.g. `data:image/png;base64,...`). Discord returns a hash of the uploaded image rather than the original, so this attribute is not refreshed from the API — drift in the image itself will not be detected.",
				Optional:    true,
				Sensitive:   true,
			},
			"url": schema.StringAttribute{
				Description: "Full webhook URL including token. Treat as a secret.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Description: "Webhook authentication token. Treat as a secret.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	avatar := ""
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		avatar = plan.Avatar.ValueString()
	}

	wh, err := r.client.Session.WebhookCreate(plan.ChannelID.ValueString(), plan.Name.ValueString(), avatar, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create webhook failed", err.Error())
		return
	}

	applyWebhook(wh, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, err := r.client.Session.Webhook(state.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read webhook failed", err.Error())
		return
	}

	applyWebhook(wh, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	avatar := ""
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		avatar = plan.Avatar.ValueString()
	}

	if _, err := r.client.Session.WebhookEdit(state.ID.ValueString(), plan.Name.ValueString(), avatar, plan.ChannelID.ValueString(), discordgo.WithContext(ctx)); err != nil {
		resp.Diagnostics.AddError("Update webhook failed", err.Error())
		return
	}

	// WebhookEdit returns a partial payload, so fetch the canonical view to
	// refresh computed attributes (including any token rotation).
	fresh, err := r.client.Session.Webhook(state.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Read webhook after update failed", err.Error())
		return
	}

	applyWebhook(fresh, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Session.WebhookDelete(state.ID.ValueString(), discordgo.WithContext(ctx)); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete webhook failed", err.Error())
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Webhook IDs are globally addressable; no guild/channel qualifier needed.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// applyWebhook writes API-derived attributes onto the model while leaving the
// user-supplied Avatar untouched — Discord returns a hash rather than the
// original data URI, so trusting the API value would produce perpetual drift.
func applyWebhook(w *discordgo.Webhook, m *webhookResourceModel) {
	m.ID = types.StringValue(w.ID)
	m.ChannelID = types.StringValue(w.ChannelID)
	m.GuildID = types.StringValue(w.GuildID)
	m.Name = types.StringValue(w.Name)
	m.URL = types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", w.ID, w.Token))
	m.Token = types.StringValue(w.Token)
}
