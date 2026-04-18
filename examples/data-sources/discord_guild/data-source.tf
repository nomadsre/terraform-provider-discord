data "discord_guild" "current" {
  id = var.guild_id
}

output "guild_name" {
  value = data.discord_guild.current.name
}
