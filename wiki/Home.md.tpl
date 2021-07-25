# Discord-Community

## Modules

{% for module in modules -%}
### {{ module.type }}

{{ module.description }}

| Attribute | Required | Type | Default Value | Description |
| --------- | -------- | ---- | ------------- | ----------- |
{%- for attr in module.attributes %}
| `{{ attr.name }}` | {% if attr.required == 'required' %}✅{% endif %} | {{ attr.type }} | {% if attr.default != "" %}`{{ attr.default }}`{% endif %} | {{ attr.description }} |
{%- endfor %}
{% endfor %}

<!-- vim: set ft=markdown : -->
