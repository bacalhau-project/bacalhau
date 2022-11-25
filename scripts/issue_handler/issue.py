import os
from dotenv import load_dotenv

from pathlib import Path

from sgqlc.endpoint.http import HTTPEndpoint
from operations import Operations

e = Path(__file__) / ".env"
load_dotenv()

gh_token = os.getenv("GITHUB_TOKEN")

endpoint = HTTPEndpoint(
    "https://api.github.com/graphql",
    {
        "Authorization": "bearer " + os.environ["GH_TOKEN"],
    },
)

owner = os.environ["OWNER"]
repo = os.environ["REPO"]

# op = Operations.query.list_issues

# # you can print the resulting GraphQL
# print(op)  # noqa: T001

# # Call the endpoint:
# data = endpoint(op, {"owner": owner, "name": repo})

# # Interpret results into native objects
# repo = (op + data).repository
# for issue in repo.issues.nodes:
#     print(issue)

op = Operations.query.get_issue
print(op)

data = endpoint(op, {"owner": owner, "name": repo, "number": 1375})
issue = (op + data).repository.issue

print(issue)
