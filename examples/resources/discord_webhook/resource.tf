resource "discord_channel" "alerts" {
  guild_id = var.guild_id
  type     = "text"
  name     = "alerts"
}

resource "discord_webhook" "alertmanager" {
  channel_id = discord_channel.alerts.id
  name       = "Alertmanager"
}

output "webhook_url" {
  value     = discord_webhook.alertmanager.url
  sensitive = true
}
