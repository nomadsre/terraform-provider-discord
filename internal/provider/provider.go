// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nomadsre/terraform-provider-discord/internal/client"
)

const envToken = "DISCORD_TOKEN"

// Ensure DiscordProvider satisfies the provider.Provider interface.
var _ provider.Provider = (*DiscordProvider)(nil)

// DiscordProvider manages Discord resources via the REST API.
type DiscordProvider struct {
	version string
}

// providerModel mirrors the provider block schema.
type providerModel struct {
	Token types.String `tfsdk:"token"`
}

// New returns a provider.Provider factory bound to the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DiscordProvider{version: version}
	}
}

func (p *DiscordProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "discord"
	resp.Version = p.version
}

func (p *DiscordProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Discord guild resources through the Discord REST API.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "Discord bot token. May be provided via the `DISCORD_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *DiscordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Defer Configure until the token is known. When a token value is derived
	// from an unknown input at plan time, Terraform will re-invoke Configure
	// during apply once the value is resolved.
	if data.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Discord bot token",
			"The provider cannot create a Discord client because the token attribute is unknown. "+
				"Either resolve the reference at plan time, or set the token via the "+envToken+" environment variable.",
		)
		return
	}

	token := data.Token.ValueString()
	if token == "" {
		token = os.Getenv(envToken)
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Discord bot token",
			"Set the `token` provider attribute or the "+envToken+" environment variable.",
		)
		return
	}

	ua := fmt.Sprintf("DiscordBot (https://github.com/nomadsre/terraform-provider-discord, %s)", p.version)
	c, err := client.New(token, ua)
	if err != nil {
		resp.Diagnostics.AddError("Failed to initialize Discord client", err.Error())
		return
	}

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *DiscordProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *DiscordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
