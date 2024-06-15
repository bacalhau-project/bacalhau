import argparse
import json
import os
from pathlib import Path

import requests
from dotenv import load_dotenv


def main():
    branch = os.getenv("BRANCH")
    circle_token = os.getenv("CIRCLE_TOKEN")
    full_name = os.getenv("FULL_NAME")

    if not circle_token:
        print("CIRCLE_TOKEN is not set. Exiting.")
        exit(1)

    if not branch:
        target = {"PUSH_BRANCH": "main"}
    elif "refs/tags" in branch:
        tag = branch.replace("refs/tags/", "")
        target = {"PUSH_TAG": tag}
    else:
        target = {"PUSH_BRANCH": branch}

    headers = {
        "Content-Type": "application/json",
        "Circle-Token": circle_token,
    }

    data = {
        "parameters": {
            "GHA_Action": "trigger_pipeline",
            "full_name": full_name,
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
    argsp.add_argument("--test", type=bool, default=False, help="Test mode.")
    args = argsp.parse_args()

    if args.env:
        if Path(args.env).exists():
            load_dotenv(args.env)
        else:
            print(f"File {args.env} does not exist. Exiting.")
            exit

    if args.test:
        if Path(args.env).exists():
            load_dotenv(args.env)
        else:
            print(f"File {args.env} does not exist. Exiting.")
            exit

        os.environ["REF"] = "main"
        os.environ["CIRCLE_TOKEN"] = os.environ["CIRCLE_TOKEN"]
        os.environ["FULL_NAME"] = "aronchick/main"

        print("Running in test mode.")
        print(f"Branch: {os.getenv('BRANCH')}")
        print(f"Circle Token: {os.getenv('CIRCLE_TOKEN')}")
        print(f"Full Name: {os.getenv('FULL_NAME')}")

    main()
