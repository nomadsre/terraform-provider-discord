data "discord_role" "admin" {
  guild_id = var.guild_id
  name     = "admin"
}

output "admin_role_id" {
  value = data.discord_role.admin.id
}
