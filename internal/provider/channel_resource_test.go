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

func TestAccChannel_text(t *testing.T) {
	rName := "tf-acc-chan-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	resourceName := "discord_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelConfig_text(rName, "hello"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("type"), knownvalue.StringExact("text")),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("topic"), knownvalue.StringExact("hello")),
				},
			},
			{
				Config: testAccChannelConfig_text(rName, "updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("topic"), knownvalue.StringExact("updated")),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccChannelImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccChannel_underCategory(t *testing.T) {
	rName := "tf-acc-nested-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelConfig_nested(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists("discord_category.parent"),
					stateCheckChannelExists("discord_channel.child"),
					statecheck.ExpectKnownValue("discord_channel.child",
						tfjsonpath.New("parent_id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccChannelConfig_text(name, topic string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_channel" "test" {
  guild_id = %[1]q
  type     = "text"
  name     = %[2]q
  topic    = %[3]q
}
`, testAccGuildID(), name, topic)
}

func testAccChannelConfig_nested(name string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_category" "parent" {
  guild_id = %[1]q
  name     = %[2]q
}

resource "discord_channel" "child" {
  guild_id  = %[1]q
  type      = "text"
  name      = "%[2]s-child"
  parent_id = discord_category.parent.id
}
`, testAccGuildID(), name)
}
