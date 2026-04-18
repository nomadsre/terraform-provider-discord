// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure DiscordProvider satisfies the provider.Provider interface.
var _ provider.Provider = (*DiscordProvider)(nil)

// DiscordProvider manages Discord resources via the REST API.
type DiscordProvider struct {
	version string
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
		Description: "The Discord provider manages Discord guild resources through the Discord REST API.",
		Attributes:  map[string]schema.Attribute{},
	}
}

func (p *DiscordProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *DiscordProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *DiscordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
