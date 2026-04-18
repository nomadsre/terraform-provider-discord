// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceRole_byName(t *testing.T) {
	rName := "tf-acc-ds-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	dsAddr := "data.discord_role.byname"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddr,
						tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(dsAddr,
						tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccRoleDataSourceConfig(name string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_role" "seed" {
  guild_id = %[1]q
  name     = %[2]q
}

data "discord_role" "byname" {
  guild_id = %[1]q
  name     = discord_role.seed.name
}
`, testAccGuildID(), name)
}
