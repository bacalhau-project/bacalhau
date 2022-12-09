---
layout: default
---

# JSON Schema
<ul>
 {% for k, v in jsonSchemas %} <li><a href="jsonschema/{{ v }}">{{ k }}</a></li> {% endfor %} 
</ul>

# Open API
<ul>
 {% for k, v in openAPIs %} <li><a href="openapi/{{ v }}">{{ k }}</a></li> {% endfor %} 
</ul>