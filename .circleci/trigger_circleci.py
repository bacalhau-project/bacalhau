import argparse
import json
import os
from pathlib import Path

import requests
from dotenv import load_dotenv


def main():
    branch = os.getenv("BRANCH")
    circle_token = os.getenv("CIRCLE_TOKEN")

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
        },
    }
    data.update(target)

    print(f"Full data: {data}")

    response = requests.post(
        "https://circleci.com/api/v2/project/gh/bacalhau-project/bacalhau/pipeline",
        headers=headers,
        data=json.dumps(data),
    )

    # If response code not in 2xx, raise an exception
    if response.status_code < 200 or response.status_code >= 300:
        print(f"Failed to trigger CircleCI pipeline: {response.status_code}")
        print(response.text)
        response.raise_for_status()
    else:
        print("Successfully triggered CircleCI pipeline")


if __name__ == "__main__":
    # Get .env file as flag
    argsp = argparse.ArgumentParser()
    argsp.add_argument("--env", type=str, default=".env")
    args = argsp.parse_args()

    if args.env:
        if Path(args.env).exists():
            load_dotenv(args.env)
        else:
            print(f"File {args.env} does not exist. Exiting.")
            exit

    main()
