# Config format

```yaml
---

bot_token: '...'
guild_id: '...'

module_configs:
  - type: module-type
    attributes:
      key: value

...
```

# Modules

## Type: `schedule`

Posts stream schedule derived from Twitch schedule as embed in Discord channel

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `discord_channel_id` | ✅ | string |  | ID of the Discord channel to post the message to |
| `embed_title` | ✅ | string |  | Title of the embed (used to find the managed post, must be unique for that channel) |
| `twitch_channel_id` | ✅ | string |  | ID (not name) of the channel to fetch the schedule from |
| `twitch_client_id` | ✅ | string |  | Twitch client ID the token was issued for |
| `twitch_token` | ✅ | string |  | Token for the user the `twitch_channel_id` belongs to |
| `cron` |  | string | `*/10 * * * *` | When to execute the schedule transfer |
| `embed_color` |  | int64 | `3066993` | Integer representation of the hex color for the embed (default is #2ECC71) |
| `embed_description` |  | string |  | Description for the embed block |
| `embed_thumbnail_height` |  | int64 |  | Height of the thumbnail |
| `embed_thumbnail_url` |  | string |  | Publically hosted image URL to use as thumbnail |
| `embed_thumbnail_width` |  | int64 |  | Width of the thumbnail |
| `schedule_entries` |  | int64 | `5` | How many schedule entries to add to the embed as fields |
| `schedule_past_time` |  | duration | `15m` | How long in the past should the schedule contain an entry |


<!-- vim: set ft=markdown : -->
