// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccWebhook_basic(t *testing.T) {
	rName := "tf-acc-wh-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	rNameUpdated := rName + "-upd"
	webhookAddr := "discord_webhook.test"
	channelAddr := "discord_channel.host"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckWebhookDestroy(t),
			testAccCheckChannelDestroy(t),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookConfig(rName, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckChannelExists(channelAddr),
					stateCheckWebhookExists(webhookAddr),
					statecheck.ExpectKnownValue(webhookAddr,
						tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(webhookAddr,
						tfjsonpath.New("url"),
						knownvalue.StringRegexp(regexp.MustCompile(`^https://discord\.com/api/webhooks/\d+/`))),
					statecheck.ExpectSensitiveValue(webhookAddr, tfjsonpath.New("url")),
					statecheck.ExpectSensitiveValue(webhookAddr, tfjsonpath.New("token")),
				},
			},
			{
				Config: testAccWebhookConfig(rName, rNameUpdated),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckWebhookExists(webhookAddr),
					statecheck.ExpectKnownValue(webhookAddr,
						tfjsonpath.New("name"), knownvalue.StringExact(rNameUpdated)),
				},
			},
			{
				ResourceName:            webhookAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"avatar"},
			},
		},
	})
}

func testAccWebhookConfig(channelName, webhookName string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_channel" "host" {
  guild_id = %[1]q
  type     = "text"
  name     = %[2]q
}

resource "discord_webhook" "test" {
  channel_id = discord_channel.host.id
  name       = %[3]q
}
`, testAccGuildID(), channelName, webhookName)
}

func stateCheckWebhookExists(address string) statecheck.StateCheck {
	return webhookExistsCheck{address: address}
}

type webhookExistsCheck struct {
	address string
}

func (c webhookExistsCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
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
	if _, err := s.Webhook(id); err != nil {
		resp.Error = fmt.Errorf("webhook %s not found: %w", id, err)
	}
}

func testAccCheckWebhookDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		sess := testAccAPISession(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "discord_webhook" {
				continue
			}
			if _, err := sess.Webhook(rs.Primary.ID); err == nil {
				return fmt.Errorf("webhook %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}
