// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCategory_basic(t *testing.T) {
	rName := "tf-acc-cat-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	rNameUpdated := rName + "-upd"
	resourceName := "discord_category.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckChannelDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccCategoryConfig(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccCategoryConfig(rNameUpdated),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("name"), knownvalue.StringExact(rNameUpdated)),
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

func testAccCategoryConfig(name string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_category" "test" {
  guild_id = %[1]q
  name     = %[2]q
}
`, testAccGuildID(), name)
}

// stateCheckChannelExists verifies any channel (text/voice/category/etc)
// exists in Discord. Shared by category, channel, and webhook tests.
func stateCheckChannelExists(address string) statecheck.StateCheck {
	return channelExistsCheck{address: address}
}

type channelExistsCheck struct {
	address string
}

func (c channelExistsCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	r, err := stateResourceAtAddress(req.State, c.address)
	if err != nil {
		resp.Error = err
		return
	}
	id, _ := r.AttributeValues["id"].(string)
	if id == "" {
		resp.Error = fmt.Errorf("missing id on %s", c.address)
		return
	}
	s, err := discordgo.New("Bot " + mustEnv(envToken))
	if err != nil {
		resp.Error = err
		return
	}
	if _, err := s.Channel(id); err != nil {
		resp.Error = fmt.Errorf("channel %s not found: %w", id, err)
	}
}

func testAccChannelImportStateIdFunc(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return rs.Primary.Attributes["guild_id"] + ":" + rs.Primary.ID, nil
	}
}

// testAccCheckChannelDestroy verifies destroyed channels/categories are gone.
// Webhooks reuse this too because they are deleted when their parent channel
// goes away, but the webhook's own resource type is also covered here.
func testAccCheckChannelDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		sess := testAccAPISession(t)
		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "discord_category", "discord_channel":
			default:
				continue
			}
			ch, err := sess.Channel(rs.Primary.ID)
			if err == nil && ch != nil {
				return fmt.Errorf("%s %s still exists", rs.Type, rs.Primary.ID)
			}
		}
		return nil
	}
}
