---
layout: default
---

# JSON Schema
<ul>

{% assign schemas = site.static_files | reverse | where: "schema", true %}
{% assign latest = schemas | first %}
<li><a href="{{ latest.path | prepend: site.baseurl }}">LATEST</a></li>
{% for schema in schemas %}
    <li><a href="{{ schema.path | prepend: site.baseurl }}">{{ schema.basename }}</a></li>
{% endfor %}

</ul>

# Open API
<ul>

</ul>
