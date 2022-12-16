import git
from pathlib import Path
from packaging import version
import semver
from sys import argv

import subprocess

from jinja2 import Environment, FileSystemLoader

STARTING_SEMVER = version.parse("0.3.11")
SCHEMA_DIR = Path(__file__).parent.parent.parent / "schema.bacalhau.org"
LATEST_SEMVER = None
rootPath = Path(__file__).parent.parent.parent

# Need to do this upfront because we'll be switching branches
# Load index.jinja file and render it into schema.bacalhau.org/index.md
env = Environment(loader=FileSystemLoader(Path(__file__).parent / "templates/"))
template = env.get_template("index.jinja")


# If --rebuild-all is passed, we will rebuild all schema files, even if they
# already exist in the schema.bacalhau.org directory
rebuild_all = False

if len(argv) > 1 and argv[1] == "--rebuild-all":
    rebuild_all = True

repo = git.Repo(rootPath)

actor = git.Actor("Bacalhau JSONSchema Builder Actor", "")
commit = git.Actor("Bacalhau JSONSchema Builder Committer", "")
subprocess.call(["go", "mod", "vendor"], cwd=rootPath)
repo.git.add(all=True)
repo.index.commit("Running JSONSchema Builder", author=actor, committer=commit)

repo.heads.main.checkout()

tagList = repo.git.ls_remote("--tags", "origin").split("refs/tags/")[1:]

listOfTagsToBuild = []

for longTag in tagList:
    splitValues = longTag.strip().split("\n")
    if len(splitValues) > 1:
        tag, commit = splitValues
    else:
        tag = splitValues[0]
        commit = None

    if tag.startswith("v"):
        tag = tag[1:]

    try:
        semVerTag = version.parse(tag)
        if semVerTag > STARTING_SEMVER and not semVerTag.is_prerelease:
            listOfTagsToBuild.append(tag)
    except ValueError as ve:
        print(f"Skipping {tag} because it is not a valid semver tag: {ve}")
        continue

most_recent_tag = max(listOfTagsToBuild)

jsonFileContents = {}

if not rebuild_all:
    listOfTagsToBuild = [listOfTagsToBuild.pop()]

for tag in listOfTagsToBuild:
    repo.git.checkout(f"v{tag}")
    print(f"Building schema files for {tag}")
    subprocess.call(["go", "mod", "vendor"], cwd=rootPath)
    subprocess.call(["make", "build-ci"], cwd=rootPath)

    GOOS = subprocess.check_output(["go", "env", "GOOS"], cwd=rootPath).decode("utf-8").strip()
    GOARCH = subprocess.check_output(["go", "env", "GOARCH"], cwd=rootPath).decode("utf-8").strip()

    proc = subprocess.Popen(
        [f"bin/{GOOS}_{GOARCH}/bacalhau", "validate", "--output-schema"], cwd=rootPath, stdout=subprocess.PIPE
    )
    jsonFileContents[tag] = proc.stdout.read().decode("utf-8")

repo.heads.main.checkout()

for jsonFile in jsonFileContents:
    schemaFile = SCHEMA_DIR / "jsonschema" / f"v{jsonFile}.json"
    with open(schemaFile, "w") as f:
        f.write(jsonFileContents[jsonFile])

jsonSchemaIndexFile = SCHEMA_DIR / "jsonschema" / "index.md"

# Render the template and write it to the index.md file
jsonSchemas = []
maxSchema = version.parse("0.0.0")
for schemaFile in SCHEMA_DIR.glob("jsonschema/v*.json"):
    currentSchema = version.parse(schemaFile.stem.lstrip("v"))
    jsonSchemas.append({"schemaVersion": str(currentSchema), "file": schemaFile.name})

    # Get the file name without the v prefix
    if currentSchema > maxSchema:
        maxSchema = currentSchema

jsonSchemas = sorted(jsonSchemas, key=lambda x: x["schemaVersion"], reverse=True)
jsonSchemas.insert(0, {"schemaVersion": "LATEST", "file": f"v{maxSchema}.json"})

jsonSchemaIndexFile = SCHEMA_DIR / "index.md"

jsonSchemaIndex = template.render(jsonSchemas=jsonSchemas)

jsonSchemaIndexFile.write_text(jsonSchemaIndex)
