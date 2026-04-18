resource "discord_role" "admin" {
  guild_id    = var.guild_id
  name        = "admin"
  color       = 16711680 # 0xFF0000 red
  hoist       = true
  mentionable = true
  permissions = "8" # ADMINISTRATOR
}
