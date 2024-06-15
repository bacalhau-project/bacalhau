#!/usr/bin/env python3

import argparse
import os
import random
import string
import subprocess
import sys

from github import Github, GithubException


def run_command(command):
    result = subprocess.run(
        command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    if result.returncode != 0:
        print(f"Error: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(result.returncode)
    return result.stdout.strip()


def remote_exists(remote_name):
    result = subprocess.run(
        f"git remote get-url {remote_name}",
        shell=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.returncode == 0


def get_pull_request_info(github_token, upstream_repo, source_gh_user, source_branch):
    g = Github(github_token)
    repo = g.get_repo(f"{upstream_repo}")
    pull_requests = repo.get_pulls(
        state="open", head=f"{source_gh_user}:{source_branch}"
    )

    if pull_requests.totalCount == 0:
        print(
            f"No open pull request found for branch {source_branch} in repo {upstream_repo}"
        )
        sys.exit(1)

    pr = pull_requests[0]
    return pr.title, pr.body


def create_pull_request(
    github_token, upstream_repo, source_branch, new_branch_name, title, body
):
    g = Github(github_token)
    repo = g.get_repo(upstream_repo)

    try:
        # Ensure the new branch exists before creating a pull request
        repo.get_branch(new_branch_name)
    except GithubException:
        print(f"Error: Branch {new_branch_name} does not exist in the repository.")
        sys.exit(1)

    if body is None:
        body = ""

    try:
        pr = repo.create_pull(
            title=title, body=body, head=new_branch_name, base="main", draft=False
        )
        print(f"Pull request created: {pr.html_url}")
    except GithubException as e:
        print(f"Failed to create pull request: {e}")
        raise


def main():
    parser = argparse.ArgumentParser(
        description="Push a forked branch to an upstream repository."
    )
    parser.add_argument(
        "--upstream_full_name",
        type=str,
        required=True,
        help="The upstream full name (e.g.: '<org>/<repo>' with the '/')",
    )
    parser.add_argument(
        "--branch_spec",
        type=str,
        required=True,
        help="The fork username and branch name in the format <fork_username>:<fork_branchname>",
    )
    parser.add_argument(
        "--gpf_upstream_branch",
        type=str,
        required=True,
        help="The upstream branch name",
    )
    parser.add_argument(
        "--github_token", type=str, required=True, help="GitHub token for API access"
    )
    args = parser.parse_args()

    upstream_full_name = args.upstream_full_name
    branch_spec = args.branch_spec
    gpf_upstream_branch = args.gpf_upstream_branch
    github_token = args.github_token

    if branch_spec.count(":") != 1:
        parser.error(
            "branch_spec must be in the format <fork_username>:<fork_branchname>"
        )

    source_gh_user, source_branch = branch_spec.split(":")
    repo_name = run_command(
        "git remote get-url --push origin | awk -F/ '{print $NF}' | sed 's/\\.git$//'"
    )

    random_string = "".join(random.choices(string.ascii_letters + string.digits, k=6))
    fork_to_test_name = f"fork-to-test-{random_string}"

    if remote_exists(fork_to_test_name):
        run_command(f"git remote remove {fork_to_test_name}")

    gpf_use_ssh = os.getenv("GPF_USE_SSH", "")

    if gpf_use_ssh:
        run_command(
            f"git remote add {fork_to_test_name} git@github.com:{source_gh_user}/{repo_name}.git"
        )
    else:
        run_command(
            f"git remote add {fork_to_test_name} https://github.com/{source_gh_user}/{repo_name}.git"
        )

    run_command("git fetch --all")

    new_branch_name = f"{gpf_upstream_branch}-pr-{random_string}"
    run_command(
        f"git push --force https://github.com/{upstream_full_name} refs/remotes/{fork_to_test_name}/{source_branch}:refs/heads/{new_branch_name}"
    )

    run_command(f"git remote remove {fork_to_test_name}")

    title, body = get_pull_request_info(
        github_token, upstream_full_name, source_gh_user, source_branch
    )
    create_pull_request(
        github_token, upstream_full_name, source_branch, new_branch_name, title, body
    )

    print(
        f"Forked branch '{branch_spec}' has been pushed to {upstream_full_name} branch '{new_branch_name}' and a pull request has been created."
    )


if __name__ == "__main__":
    main()
