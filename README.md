# Terraform Provider for Discord

Manage Discord guild resources (roles, categories, channels, webhooks) through
Terraform. Uses the Discord REST API via [`discordgo`](https://github.com/bwmarrin/discordgo).

## Requirements

- Terraform >= 1.6
- Go >= 1.26 (to build from source)
- A Discord bot token with the required guild permissions

## Usage

```hcl
terraform {
  required_providers {
    discord = {
      source  = "nomadsre/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {
  # token = var.discord_bot_token  # or set DISCORD_TOKEN env var
}

resource "discord_role" "admin" {
  guild_id    = var.guild_id
  name        = "admin"
  color       = 0xFF0000
  hoist       = true
  mentionable = true
  permissions = "8"
}
```

## Development

```sh
make build        # compile provider binary
make install      # install to local plugin cache (~/.terraform.d/plugins)
make test         # unit tests
make testacc      # acceptance tests (requires DISCORD_TOKEN + DISCORD_TEST_GUILD_ID)
make docs         # regenerate registry docs with tfplugindocs
```

## License

[MPL-2.0](./LICENSE)
