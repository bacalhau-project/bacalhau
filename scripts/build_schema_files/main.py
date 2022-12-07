import git
from pathlib import Path
import semver
from sys import argv

import subprocess

from jinja2 import Environment, FileSystemLoader

STARTING_SEMVER = semver.parse("0.3.11")
SCHEMA_DIR = Path(__file__).parent.parent.parent / "schema.bacalhau.org"
LATEST_SEMVER = None
rootPath = Path(__file__).parent.parent.parent

# Need to do this upfront because we'll be switching branches
# Load index.jinja file and render it into schema.bacalhau.org/index.md
env = Environment(loader=FileSystemLoader(rootPath))
template = env.get_template(rootPath / "index.jinja")


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
        semVerTag = semver.VersionInfo.parse(tag)
        print(semVerTag)
        if semVerTag > STARTING_SEMVER and semVerTag.prerelease is None:
            listOfTagsToBuild.append(tag)
    except ValueError as ve:
        print(f"Skipping {tag} because it is not a valid semver tag: {ve}")
        continue

for tag in listOfTagsToBuild:
    if not rebuild_all:
        if (SCHEMA_DIR / tag).exists():
            print(f"Skipping {tag} because it already exists")
            continue

    repo.git.checkout(f"v{tag}")

most_recent_tag = max(listOfTagsToBuild)

if rebuild_all:
    for tag in listOfTagsToBuild[0:-1]:
        repo.git.checkout(f"v{tag}")
        print(f"Building schema files for {tag}")
        subprocess.call(["go", "mod", "vendor"], cwd=rootPath)
        subprocess.call(["make", "build"], cwd=rootPath)
        proc = subprocess.Popen(
            ["bin/darwin_arm64/bacalhau", "validate", "--output-schema"], cwd=rootPath, stdout=subprocess.PIPE
        )
        repo.heads.main.checkout()
        schemaFile = SCHEMA_DIR / "jsonschema" / f"v{tag}.json"
        with open(schemaFile, "w") as f:
            f.write_text(proc.stdout.read().decode("utf-8"))
else:
    proc = subprocess.Popen(["bacalhau", "version"], stdout=subprocess.PIPE)
    versionOutput = proc.stdout.read().decode("utf-8")

    proc = subprocess.Popen(["bacalhau", "validate", "--output-schema"], cwd=rootPath, stdout=subprocess.PIPE)
    schemaFile = SCHEMA_DIR / "jsonschema" / f"v{most_recent_tag}.json"
    with open(schemaFile, "w") as f:
        f.write_text(proc.stdout.read().decode("utf-8"))

jsonSchemaIndex = SCHEMA_DIR / "jsonschema" / "index.md"

# Render the template and write it to the index.md file
jsonSchemas = []
maxSchema = semver.parse("0.0.0")
for schemaFile in SCHEMA_DIR.glob("jsonschema/v*.json"):
    jsonSchemas.append({"name": schemaFile.name, "file": f"{schemaFile.name}.json"})
    if semver.parse(schemaFile.name) > maxSchema:
        maxSchema = schemaFile.name

jsonSchemas = sorted(jsonSchemas, key=lambda x: semver.parse(x[0].name), reverse=True)
jsonSchemas.push(("LATEST", f"v{maxSchema}.json"))

template.render(jsonSchemas=jsonSchemas)
