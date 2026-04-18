// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

var (
	_ resource.Resource                = (*roleResource)(nil)
	_ resource.ResourceWithConfigure   = (*roleResource)(nil)
	_ resource.ResourceWithImportState = (*roleResource)(nil)
)

// NewRoleResource returns a resource.Resource factory for discord_role.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *client.Client
}

type roleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	GuildID     types.String `tfsdk:"guild_id"`
	Name        types.String `tfsdk:"name"`
	Color       types.Int64  `tfsdk:"color"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Permissions types.String `tfsdk:"permissions"`
	Position    types.Int64  `tfsdk:"position"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Discord guild role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Role ID (snowflake).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"guild_id": schema.StringAttribute{
				Description: "ID of the guild that owns this role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Role name.",
				Required:    true,
			},
			"color": schema.Int64Attribute{
				Description: "Role color as a decimal RGB integer (e.g. 0xFF0000 = 16711680). 0 means no color.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"hoist": schema.BoolAttribute{
				Description: "Display role members separately in the member list.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"mentionable": schema.BoolAttribute{
				Description: "Whether the role can be @mentioned by anyone.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"permissions": schema.StringAttribute{
				Description: "Role permissions as a Discord permission bitfield encoded as a decimal string.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"position": schema.Int64Attribute{
				Description: "Position of the role in the guild hierarchy. Higher values sort above lower ones; @everyone is always position 0. If omitted, Discord assigns the next available position.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params, diags := buildRoleParams(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.Session.GuildRoleCreate(plan.GuildID.ValueString(), params, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create role failed", err.Error())
		return
	}

	if !plan.Position.IsNull() && !plan.Position.IsUnknown() && int(plan.Position.ValueInt64()) != role.Position {
		role, err = r.reorderRole(ctx, plan.GuildID.ValueString(), role.ID, int(plan.Position.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError("Reorder role failed", err.Error())
			return
		}
	}

	roleToState(role, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.findRole(ctx, state.GuildID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read role failed", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	roleToState(role, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params, diags := buildRoleParams(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.Session.GuildRoleEdit(plan.GuildID.ValueString(), state.ID.ValueString(), params, discordgo.WithContext(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Update role failed", err.Error())
		return
	}

	if !plan.Position.IsNull() && !plan.Position.IsUnknown() && int(plan.Position.ValueInt64()) != role.Position {
		role, err = r.reorderRole(ctx, plan.GuildID.ValueString(), role.ID, int(plan.Position.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError("Reorder role failed", err.Error())
			return
		}
	}

	roleToState(role, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Session.GuildRoleDelete(state.GuildID.ValueString(), state.ID.ValueString(), discordgo.WithContext(ctx))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete role failed", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	guildID, roleID, ok := strings.Cut(req.ID, ":")
	if !ok || guildID == "" || roleID == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			`Expected "<guild_id>:<role_id>", got: `+req.ID,
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("guild_id"), guildID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), roleID)...)
}

// findRole fetches a single role by listing guild roles (Discord has no
// get-role-by-id endpoint). Returns (nil, nil) when the role is not found.
func (r *roleResource) findRole(ctx context.Context, guildID, roleID string) (*discordgo.Role, error) {
	roles, err := r.client.Session.GuildRoles(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == roleID {
			return role, nil
		}
	}
	return nil, nil
}

// reorderRole moves the role to the requested position. Discord requires
// submitting the full ordered role list; we fetch the current list and patch
// the single role's position in-place before sending.
func (r *roleResource) reorderRole(ctx context.Context, guildID, roleID string, position int) (*discordgo.Role, error) {
	roles, err := r.client.Session.GuildRoles(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list roles for reorder: %w", err)
	}
	var updated *discordgo.Role
	for _, role := range roles {
		if role.ID == roleID {
			role.Position = position
			updated = role
			break
		}
	}
	if updated == nil {
		return nil, fmt.Errorf("role %s not found in guild %s", roleID, guildID)
	}
	newRoles, err := r.client.Session.GuildRoleReorder(guildID, roles, discordgo.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	for _, role := range newRoles {
		if role.ID == roleID {
			return role, nil
		}
	}
	return updated, nil
}

func buildRoleParams(m roleResourceModel) (*discordgo.RoleParams, diag.Diagnostics) {
	var diags diag.Diagnostics
	params := &discordgo.RoleParams{Name: m.Name.ValueString()}

	if !m.Color.IsNull() && !m.Color.IsUnknown() {
		c := int(m.Color.ValueInt64())
		params.Color = &c
	}
	if !m.Hoist.IsNull() && !m.Hoist.IsUnknown() {
		h := m.Hoist.ValueBool()
		params.Hoist = &h
	}
	if !m.Mentionable.IsNull() && !m.Mentionable.IsUnknown() {
		v := m.Mentionable.ValueBool()
		params.Mentionable = &v
	}
	if !m.Permissions.IsNull() && !m.Permissions.IsUnknown() {
		perms, err := strconv.ParseInt(m.Permissions.ValueString(), 10, 64)
		if err != nil {
			diags.AddAttributeError(path.Root("permissions"), "Invalid permissions bitfield",
				fmt.Sprintf("permissions must be a base-10 integer string: %s", err))
			return nil, diags
		}
		params.Permissions = &perms
	}
	return params, diags
}

func roleToState(role *discordgo.Role, m *roleResourceModel) {
	m.ID = types.StringValue(role.ID)
	m.Name = types.StringValue(role.Name)
	m.Color = types.Int64Value(int64(role.Color))
	m.Hoist = types.BoolValue(role.Hoist)
	m.Mentionable = types.BoolValue(role.Mentionable)
	m.Permissions = types.StringValue(strconv.FormatInt(role.Permissions, 10))
	m.Position = types.Int64Value(int64(role.Position))
}
