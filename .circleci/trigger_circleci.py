import json
import os

import requests


def main():
    branch = os.getenv("BRANCH")
    circle_token = os.getenv("CIRCLE_TOKEN")
    name = os.getenv("NAME")

    if not circle_token:
        print("CIRCLE_TOKEN is not set. Exiting.")
        exit(1)

    if not branch:
        target = {"branch": "main"}
    elif "refs/tags" in branch:
        tag = branch.replace("refs/tags/", "")
        target = {"tag": tag}
    else:
        target = {"branch": branch}

    headers = {
        "Content-Type": "application/json",
        "Circle-Token": circle_token,
    }

    data = {
        "parameters": {
            "GHA_Action": "trigger_pipeline",
            "Name": name,
        }
    }
    data.update(target)

    response = requests.post(
        "https://circleci.com/api/v2/project/gh/bacalhau-project/bacalhau/pipeline",
        headers=headers,
        data=json.dumps(data),
    )

    if response.status_code != 200:
        print(f"Failed to trigger CircleCI pipeline: {response.status_code}")
        print(response.text)
        response.raise_for_status()
    else:
        print("Successfully triggered CircleCI pipeline")


if __name__ == "__main__":
    main()
