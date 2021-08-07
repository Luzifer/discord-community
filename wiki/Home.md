# Setup

## Discord Bot

- Go to https://discord.com/developers/applications and create your application
- Go to "Bot" in your newly created application and click "Add a bot"
- Give it a name which later will be your bots name, optionally upload an image which will be its profile image
- Disable "Public Bot", enable the "Privileged Gateway Intents"
- Copy and note your bots token (you will need to enter it into the `bot_token` field of the config)
- Add your bot to your server (replace `<client-id>` with the client ID of your bot, find that by clicking "OAuth2" in the left sidebar):  
`https://discord.com/oauth2/authorize?client_id=<client-id>&scope=bot%20applications.commands&permissions=1945627743`

## Create a config

- Create a new text file named `config.yaml` (you can name it otherwise, just adapt the rest of the examples)
- Put the text shown below ("Config format") into it
- Adjust the `module_configs`

## Start the bot

### Using Docker

```console
# docker pull luzifer/discord-community
# docker run --rm -ti -v /path/to/your/configfile:/config -e CONFIG=/config/config.yaml luzifer/discord-community
```

### Using Binary

- Download the latest release from the [release page](https://github.com/Luzifer/discord-community/releases)
- Unpack the archive you've downloaded
- Start the bot in the same directory as your config (or provide a path to the config):
```console
# ./discord-community_linux_amd64 --config=config.yaml
# discord-community_windows_amd64.exe --config=config.yaml
```

# Config format

```yaml
---

bot_token: '...'
guild_id: '...'
store_location: /path/to/storage.json

module_configs:
  - id: unique id for the module instance
    type: module-type
    attributes:
      key: value

...
```

# Modules

## Type: `clearchannel`

Cleans up old messages from a channel (for example announcement channel) which are older than the retention time

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `discord_channel_id` | ✅ | string |  | ID of the Discord channel to clean up |
| `retention` | ✅ | duration |  | How long to keep messages in this channel |
| `cron` |  | string | `0 * * * *` | When to execute the cleaner |
| `only_users` |  | []string | `[]` | When this list contains user IDs, only posts authored by those IDs will be deleted |
| `protect_users` |  | []string | `[]` | When this list contains user IDs, posts authored by those IDs will not be deleted |

## Type: `liveposting`

Announces stream live status based on Discord streaming status

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `discord_channel_id` | ✅ | string |  | ID of the Discord channel to post the message to |
| `post_text` | ✅ | string |  | Message to post to channel use `${displayname}` and `${username}` as placeholders |
| `twitch_client_id` | ✅ | string |  | Twitch client ID the token was issued for |
| `twitch_client_secret` | ✅ | string |  | Secret for the Twitch app identified with twitch_client_id |
| `auto_publish` |  | bool | `false` | Automatically publish (crosspost) the message to followers of the channel |
| `cron` |  | string | `*/5 * * * *` | Fetch live status of `poll_usernames` (set to empty string to disable): keep this below `stream_freshness` or you might miss streams |
| `disable_presence` |  | bool | `false` | Disable posting live-postings for discord presence changes |
| `poll_usernames` |  | []string | `[]` | Check these usernames for active streams when executing the `cron` (at most 100 users can be checked) |
| `stream_freshness` |  | duration | `5m` | How long after stream start to post shoutout |
| `whitelisted_role` |  | string |  | Only post for members of this role ID |

## Type: `liverole`

Adds live-role to certain group of users if they are streaming on Twitch

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `role_streamers_live` | ✅ | string |  | Role ID to assign to live streamers |
| `twitch_client_id` | ✅ | string |  | Twitch client ID the token was issued for |
| `twitch_client_secret` | ✅ | string |  | Secret for the Twitch app identified with twitch_client_id |
| `role_streamers` |  | string |  | Only take members with this role ID into account |

## Type: `presence`

Updates the presence status of the bot to display the next stream

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `fallback_text` | ✅ | string |  | What to set the text to when no stream is found (`playing <text>`) |
| `twitch_channel_id` | ✅ | string |  | ID (not name) of the channel to fetch the schedule from |
| `twitch_client_id` | ✅ | string |  | Twitch client ID the token was issued for |
| `twitch_token` | ✅ | string |  | Token for the user the `twitch_channel_id` belongs to |
| `cron` |  | string | `* * * * *` | When to execute the module |
| `schedule_past_time` |  | duration | `15m` | How long in the past should the schedule contain an entry |

## Type: `reactionrole`

Creates a post with pre-set reactions and assigns roles on reaction

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `discord_channel_id` | ✅ | string |  | ID of the Discord channel to post the message to |
| `embed_title` | ✅ | string |  | Title of the embed |
| `reaction_roles` | ✅ | []string |  | List of strings in format `emote=role-id[:set]`. `emote` equals an unicode emote (✅) or a custom emote in form `:<emote-name>:<emote-id>`. `role-id` is the integer ID of the guilds role to add with this emote. If `:set` is added at the end, the role will only be added but not removed when the reaction is removed. |
| `embed_color` |  | int64 | `0x2ECC71` | Integer / HEX representation of the color for the embed |
| `embed_description` |  | string |  | Description for the embed block |
| `embed_thumbnail_height` |  | int64 |  | Height of the thumbnail |
| `embed_thumbnail_url` |  | string |  | Publically hosted image URL to use as thumbnail |
| `embed_thumbnail_width` |  | int64 |  | Width of the thumbnail |

## Type: `schedule`

Posts stream schedule derived from Twitch schedule as embed in Discord channel

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
| `discord_channel_id` | ✅ | string |  | ID of the Discord channel to post the message to |
| `embed_title` | ✅ | string |  | Title of the embed |
| `twitch_channel_id` | ✅ | string |  | ID (not name) of the channel to fetch the schedule from |
| `twitch_client_id` | ✅ | string |  | Twitch client ID the token was issued for |
| `twitch_token` | ✅ | string |  | Token for the user the `twitch_channel_id` belongs to |
| `cron` |  | string | `*/10 * * * *` | When to execute the schedule transfer |
| `embed_color` |  | int64 | `0x2ECC71` | Integer / HEX representation of the color for the embed |
| `embed_description` |  | string |  | Description for the embed block |
| `embed_thumbnail_height` |  | int64 |  | Height of the thumbnail |
| `embed_thumbnail_url` |  | string |  | Publically hosted image URL to use as thumbnail |
| `embed_thumbnail_width` |  | int64 |  | Width of the thumbnail |
| `schedule_entries` |  | int64 | `5` | How many schedule entries to add to the embed as fields |
| `schedule_past_time` |  | duration | `15m` | How long in the past should the schedule contain an entry |



<!-- vim: set ft=markdown : -->
