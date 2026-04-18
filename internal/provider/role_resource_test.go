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

func TestAccRole_basic(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	rNameUpdated := rName + "-upd"
	resourceName := "discord_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRoleExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("name"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("color"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("hoist"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("mentionable"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("permissions"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("guild_id"), knownvalue.StringExact(testAccGuildID())),
				},
			},
			{
				Config: testAccRoleConfig_full(rNameUpdated),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRoleExists(resourceName),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("name"), knownvalue.StringExact(rNameUpdated)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("color"), knownvalue.Int64Exact(16711680)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("hoist"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("mentionable"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("permissions"), knownvalue.StringExact("2048")),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRoleImportStateIdFunc(resourceName),
			},
		},
	})
}

func TestAccRole_disappears(t *testing.T) {
	rName := "tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resourceName := "discord_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRoleExists(resourceName),
					stateCheckRoleDisappears(t, resourceName),
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_role" "test" {
  guild_id = %[1]q
  name     = %[2]q
}
`, testAccGuildID(), name)
}

func testAccRoleConfig_full(name string) string {
	return fmt.Sprintf(`
provider "discord" {}

resource "discord_role" "test" {
  guild_id    = %[1]q
  name        = %[2]q
  color       = 16711680
  hoist       = true
  mentionable = true
  permissions = "2048"
}
`, testAccGuildID(), name)
}

func testAccRoleImportStateIdFunc(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return rs.Primary.Attributes["guild_id"] + ":" + rs.Primary.ID, nil
	}
}

// stateCheckRoleExists verifies that the role recorded in state still exists
// in Discord. It returns a statecheck.StateCheck for use in ConfigStateChecks.
func stateCheckRoleExists(address string) statecheck.StateCheck {
	return roleExistsCheck{address: address}
}

type roleExistsCheck struct {
	address string
}

func (c roleExistsCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	r, err := stateResourceAtAddress(req.State, c.address)
	if err != nil {
		resp.Error = err
		return
	}
	guildID, _ := r.AttributeValues["guild_id"].(string)
	roleID, _ := r.AttributeValues["id"].(string)
	if guildID == "" || roleID == "" {
		resp.Error = fmt.Errorf("missing guild_id/id on %s", c.address)
		return
	}

	s, err := discordgo.New("Bot " + mustEnv(envToken))
	if err != nil {
		resp.Error = err
		return
	}
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		resp.Error = fmt.Errorf("fetch roles for guild %s: %w", guildID, err)
		return
	}
	for _, role := range roles {
		if role.ID == roleID {
			return
		}
	}
	resp.Error = fmt.Errorf("role %s not found in guild %s", roleID, guildID)
}

func stateCheckRoleDisappears(t *testing.T, address string) statecheck.StateCheck {
	return roleDisappearsCheck{t: t, address: address}
}

type roleDisappearsCheck struct {
	t       *testing.T
	address string
}

func (c roleDisappearsCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	r, err := stateResourceAtAddress(req.State, c.address)
	if err != nil {
		resp.Error = err
		return
	}
	guildID, _ := r.AttributeValues["guild_id"].(string)
	roleID, _ := r.AttributeValues["id"].(string)
	s := testAccAPISession(c.t)
	if err := s.GuildRoleDelete(guildID, roleID); err != nil {
		resp.Error = fmt.Errorf("delete role %s externally: %w", roleID, err)
	}
}

func testAccCheckRoleDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		sess := testAccAPISession(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "discord_role" {
				continue
			}
			guildID := rs.Primary.Attributes["guild_id"]
			roles, err := sess.GuildRoles(guildID)
			if err != nil {
				return fmt.Errorf("list roles in %s: %w", guildID, err)
			}
			for _, role := range roles {
				if role.ID == rs.Primary.ID {
					return fmt.Errorf("role %s still exists in guild %s", rs.Primary.ID, guildID)
				}
			}
		}
		return nil
	}
}

func mustEnv(key string) string {
	v, ok := lookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("%s not set", key))
	}
	return v
}
