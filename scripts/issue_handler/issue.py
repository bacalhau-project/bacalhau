from ghapi.all import GhApi
import os
from dotenv import load_dotenv

from pathlib import Path

e = Path(__file__) / ".env"
load_dotenv()

gh_token = os.getenv("GITHUB_TOKEN")
api = GhApi(owner="filecoin-project", token=gh_token)

p = api.projectsv2.list_cards(column_id="f17de28f")
print(f"{p.title} - {p.id}")

# # Get project ID for filecoin-project - bacalhau
# columns = api.projects.list_columns()q

# for c in columns:
#     print(c)

# columns = api.projects.get_project_columns(project_id=1)

# cards = api.projects.list_cards(column_id=1)
# for card in cards:
#     # If the card points to an issue, grab its list of labels
#     if "content_url" not in card:
#         continue
#     _, org, repo, _, num = card["content_url"].rsplit("/", 4)
#     issue_labels = api.issues.list_labels_on_issue(org, repo, num)


# gh api graphql -f query='
#   query{
#   node(id: "PVT_kwDOAU_qk84AHJ4X") {
#     ... on ProjectV2 {
#       fields(first: 20) {
#         nodes {
#           ... on ProjectV2Field {
#             id
#             name
#           }
#           ... on ProjectV2IterationField {
#             id
#             name
#             configuration {
#               iterations {
#                 startDate
#                 id
#               }
#             }
#           }
#           ... on ProjectV2SingleSelectField {
#             id
#             name
#             options {
#               id
#               name
#             }
#           }
#         }
#       }
#     }
#   }
# }'


# "options": [
#   {
#     "id": "f17de28f",
#     "name": "Triage"
#   },
#   {
#     "id": "f75ad846",
#     "name": "Todo"
#   },
#   {
#     "id": "1e2a6912",
#     "name": "Must Have for Next Event"
#   },
#   {
#     "id": "47fc9ee4",
#     "name": "In Progress"
#   },
#   {
#     "id": "e3b53dda",
#     "name": "To Celebrate"
#   },
#   {
#     "id": "98236657",
#     "name": "Done"
#   }
