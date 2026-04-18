resource "discord_category" "ops" {
  guild_id = var.guild_id
  name     = "ops"
}

resource "discord_channel" "alerts" {
  guild_id  = var.guild_id
  type      = "text"
  name      = "alerts"
  topic     = "Automated alerts from monitoring systems."
  parent_id = discord_category.ops.id
}

resource "discord_channel" "oncall_voice" {
  guild_id   = var.guild_id
  type       = "voice"
  name       = "oncall"
  parent_id  = discord_category.ops.id
  bitrate    = 64000
  user_limit = 10
}
