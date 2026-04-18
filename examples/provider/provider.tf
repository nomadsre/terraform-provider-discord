terraform {
  required_providers {
    discord = {
      source  = "nomadsre/discord"
      version = "~> 0.1"
    }
  }
}

# The bot token is read from the DISCORD_TOKEN env var by default.
# Set it explicitly here only if you need per-workspace credentials.
provider "discord" {
  # token = var.discord_bot_token
}
