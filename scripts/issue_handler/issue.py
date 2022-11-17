from ghapi.all import GhApi
import os
from dotenv import load_dotenv

from pathlib import Path

e = Path(__file__) / ".env"
load_dotenv()

gh_token = os.getenv("GITHUB_TOKEN")
api = GhApi(owner="filecoin-project", repo="bacalhau", token=gh_token)

# Get all issues from bacalhau using the api
issues = api.projects.list_for_repo(org="filecoin-project", repo="bacalhau")

# Bacalhau project IDs
#

for i in issues:
    print(i.id)



projects = api.ProjectV2.list_for_org(org="filecoin-project")

for p in projects:
    print(f"{p.name} - {p.id}")

# Get project ID for filecoin-project - bacalhau
columns = api.projects.list_columns(project_id=65)

for c in columns:
    print(c)

# columns = api.projects.get_project_columns(project_id=1)

# cards = api.projects.list_cards(column_id=1)
# for card in cards:
#     # If the card points to an issue, grab its list of labels
#     if "content_url" not in card:
#         continue
#     _, org, repo, _, num = card["content_url"].rsplit("/", 4)
#     issue_labels = api.issues.list_labels_on_issue(org, repo, num)
