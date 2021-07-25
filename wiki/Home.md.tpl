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
