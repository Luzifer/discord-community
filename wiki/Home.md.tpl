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

# See documentation above
bot_token: '...'
# ID of your Discord "server" (internally named "guild")
guild_id: '...'
# File location to store a persistent state for the modules
store_location: /path/to/storage.json

module_configs:
  - id: 'unique id for the module instance (e.g. UUID)'
    type: module-type
    attributes:
      key: value

...
```

# Modules

{% for module in modules -%}
## Type: `{{ module.type }}`

{{ module.description }}

| Attribute | Req. | Type | Default Value | Description |
| --------- | :--: | ---- | ------------- | ----------- |
{%- for attr in module.attributes %}
| `{{ attr.name }}` | {% if attr.required == 'required' %}✅{% endif %} | {{ attr.type }} | {% if attr.default != "" %}`{{ attr.default }}`{% endif %} | {{ attr.description }} |
{%- endfor %}

{% endfor %}

<!-- vim: set ft=markdown : -->
