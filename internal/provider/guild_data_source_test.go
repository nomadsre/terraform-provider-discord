// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceGuild_basic(t *testing.T) {
	dsAddr := "data.discord_guild.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGuildDataSourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddr,
						tfjsonpath.New("id"), knownvalue.StringExact(testAccGuildID())),
					statecheck.ExpectKnownValue(dsAddr,
						tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAddr,
						tfjsonpath.New("owner_id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccGuildDataSourceConfig() string {
	return fmt.Sprintf(`
provider "discord" {}

data "discord_guild" "test" {
  id = %[1]q
}
`, testAccGuildID())
}
