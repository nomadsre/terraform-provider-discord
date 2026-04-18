// Copyright (c) nomadsre
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/bwmarrin/discordgo"
)

const (
	envTestGuildID = "DISCORD_TEST_GUILD_ID"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"discord": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies required environment variables are set before
// running acceptance tests. Acceptance tests mutate real Discord resources,
// so they are gated behind TF_ACC and an explicit throwaway guild.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv(envToken) == "" {
		t.Fatalf("%s must be set for acceptance tests", envToken)
	}
	if os.Getenv(envTestGuildID) == "" {
		t.Fatalf("%s must be set for acceptance tests", envTestGuildID)
	}
}

// testAccAPISession returns a raw discordgo session for test helpers that need
// to interrogate or mutate Discord directly (e.g. exists/disappears checks).
func testAccAPISession(t *testing.T) *discordgo.Session {
	t.Helper()
	s, err := discordgo.New("Bot " + os.Getenv(envToken))
	if err != nil {
		t.Fatalf("construct test session: %s", err)
	}
	return s
}

func testAccGuildID() string {
	return os.Getenv(envTestGuildID)
}

func stateResourceAtAddress(state *tfjson.State, address string) (*tfjson.StateResource, error) {
	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		return nil, fmt.Errorf("no state available")
	}
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == address {
			return r, nil
		}
	}
	return nil, fmt.Errorf("not found in state: %s", address)
}

// ctxWith returns a background context for tests that need one.
func ctxWith(_ *testing.T) context.Context {
	return context.Background()
}

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
