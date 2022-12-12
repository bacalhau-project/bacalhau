from pathlib import Path

from jinja2 import Environment, FileSystemLoader

import pickle

SCHEMA_DIR = Path(__file__).parent.parent.parent / "schema.bacalhau.org"

# Need to do this upfront because we'll be switching branches
# Load index.jinja file and render it into schema.bacalhau.org/index.md
env = Environment(loader=FileSystemLoader(Path(__file__).parent / "templates/"))
template = env.get_template("index.jinja")

# Load temp.pkl and depickle it into a dictionary
with open(Path(__file__).parent / "temp.pkl", "rb") as f:
    jsonSchemas = pickle.load(f)

jsonSchemaIndexFile = SCHEMA_DIR / "jsonschema" / "index.md"
