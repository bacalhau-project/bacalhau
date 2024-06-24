#!/usr/bin/env python3

import argparse
import json
import logging
import os
import re
import subprocess
from typing import List, Tuple, TypedDict

import semver
from tabulate import tabulate

import docker

DOCKER_REGISTRY_URL = "docker.io"
GCR_REGISTRY_URL = "gcr.io"
DOCKER_REPO = "bacalhauproject"
PROJECT = "bacalhau-infra"


# Define an image type
class Image(TypedDict):
    name: str
    version: str
    from_image: str


# Define a filter to suppress repetitive log messages
class RepetitiveLogFilter(logging.Filter):
    def __init__(self):
        self.last_log = None

    def filter(self, record):
        current_log = (record.levelname, record.msg)
        if current_log != self.last_log:
            self.last_log = current_log
            return True
        return False


# Set up the logging configuration
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)

# Get the root logger
root_logger = logging.getLogger()

# Add the filter to suppress repetitive log messages
repetitive_log_filter = RepetitiveLogFilter()
root_logger.addFilter(repetitive_log_filter)


def authenticate_gcr():
    try:
        subprocess.run(["gcloud", "auth", "configure-docker"], check=True)
        print("Configured docker with GCR successfully.")
    except subprocess.CalledProcessError:
        print(
            "Error: Failed to authenticate with GCR. Please ensure you have the correct credentials."
        )
        print("To authenticate, run: gcloud auth login && gcloud auth configure-docker")


def check_docker_authentication():
    config_path = os.path.expanduser("~/.docker/config.json")
    if not os.path.exists(config_path):
        print("Error: Docker config file not found. Please log in to Docker first.")
        print("To log in, run: docker login")
        return False

    with open(config_path, "r") as config_file:
        config_data = json.load(config_file)
        if "credHelpers" in config_data and "gcr.io" in config_data["credHelpers"]:
            return True
        else:
            print("Error: GCR credentials not found in Docker config file.")
            print("To configure GCR credentials, run: gcloud auth configure-docker")
            return False


def get_latest_version_from_file(filepath="VERSION"):
    if not os.path.exists(filepath):
        return None

    with open(filepath, "r") as version_file:
        version = version_file.read().strip()

    if not semver.VersionInfo.is_valid(version):
        print(f"Invalid version found in {filepath} file: {version}")
        return None

    return version


def increment_version(version) -> semver.VersionInfo:
    if isinstance(version, str):
        version = semver.VersionInfo.parse(version)
    return version.bump_patch()


def build_docker_image(docker_client, repo, image_name, version, base_image_name):
    docker_image = f"{DOCKER_REGISTRY_URL}/{repo}/{image_name}"
    gcr_image = f"{GCR_REGISTRY_URL}/{PROJECT}/{image_name}"
    tag_list = [
        f"{docker_image}:{version}",
        f"{docker_image}:latest",
        f"{gcr_image}:{version}",
        f"{gcr_image}:latest",
    ]

    root_logger.info(f"Building image {docker_image}:{version}")
    root_logger.info(
        f"Environment: GOOS={os.getenv('GOOS')}, GOARCH={os.getenv('GOARCH')}"
    )

    root_logger.info(f"Building base image {docker_image}:{version}")

    try:
        result = docker_client.api.build(
            path=".",
            tag=f"{docker_image}:{version}",
            platform="linux/amd64",
            decode=True,
            cache_from=[f"{GCR_REGISTRY_URL}/{PROJECT}/{image_name}:latest"],
        )
        for chunk in result:
            if "stream" in chunk:
                for line in chunk["stream"].splitlines():  # Split into lines
                    root_logger.info(line.strip())
            if "errorDetail" in chunk:  # Handle detailed error messages
                root_logger.error(chunk["errorDetail"]["message"])
                raise docker.errors.BuildError(chunk["errorDetail"]["message"], result)
            if "error" in chunk:  # Handle other errors
                root_logger.error(chunk["error"])
                raise docker.errors.BuildError(chunk["error"], result)

        # Build and tag the remaining images
        for img_name in [
            f"{docker_image}:latest",
            f"{gcr_image}:{version}",
            f"{gcr_image}:latest",
        ]:
            docker_client.api.tag(f"{docker_image}:{version}", img_name)

    except docker.errors.BuildError as e:
        root_logger.error(f"Error building image: {e}")
        root_logger.error("Build logs:")
        for line in e.build_log:
            # Check if the line is a string or a dictionary before logging
            if isinstance(line, str):
                root_logger.error(line.strip())
            elif "stream" in line:
                root_logger.error(line["stream"].strip())
        return None
    root_logger.info("Image built successfully")
    return tag_list


def push_docker_image(docker_client, tag_list):
    for tag in tag_list:
        try:
            print(f"Pushing image {tag}")
            # Capture and log push output in real-time
            for line in docker_client.images.push(tag, stream=True, decode=True):
                if "error" in line:
                    root_logger.error(line["error"].strip())
                elif "status" in line:
                    root_logger.info(line["status"].strip())
        except docker.errors.APIError as e:
            print(f"Error pushing image {tag}: {e}")


def get_dirs():
    return [d for d in os.listdir() if os.path.isdir(d)]


def get_images_to_build() -> List[Tuple[str, str]]:
    images_to_build = []

    def search_directories(current_dir):
        for item in os.listdir(current_dir):
            item_path = os.path.join(current_dir, item)
            if os.path.isdir(item_path) and not item.startswith("."):
                dockerfile_path = os.path.join(item_path, "Dockerfile")
                if os.path.exists(dockerfile_path):
                    process_dockerfile(item_path, dockerfile_path)
                else:
                    search_directories(item_path)

    def process_dockerfile(directory, dockerfile_path):
        with open(dockerfile_path, "r") as dockerfile:
            content = dockerfile.read()
            from_match = re.search(r"^FROM\s+(\S+)", content, re.MULTILINE)
            if not from_match:
                return

            from_image = from_match.group(1)
            base_image_dir = from_image.split("/")[-1].split(":")[0]

            # Check if base image is in any parent directory
            current_dir = directory
            while current_dir != os.path.dirname(current_dir):
                if base_image_dir in os.listdir(current_dir):
                    base_version_file = os.path.join(
                        current_dir, base_image_dir, "VERSION"
                    )
                    if os.path.exists(base_version_file):
                        with open(base_version_file, "r") as version_file:
                            base_version = version_file.read().strip()

                        if f"{base_image_dir}:{base_version}" not in from_image:
                            print(
                                f"Warning: {directory} Dockerfile FROM tag ({from_image}) "
                                f"doesn't match the version in {base_image_dir}/VERSION ({base_version})"
                            )
                            return
                    break
                current_dir = os.path.dirname(current_dir)

        version_file = os.path.join(directory, "VERSION")
        if os.path.exists(version_file):
            with open(version_file, "r") as version_file:
                version = version_file.read().strip()
            images_to_build.append((directory, version))

    search_directories(".")
    return images_to_build


def check_versions(image, version) -> Tuple[str, str]:
    root_logger.debug(f"Checking versions for {image}:{version}")

    # Get the VERSION tag from the VERSION file
    version_file = os.path.join(image, "VERSION")
    with open(version_file, "r") as version_file:
        version = version_file.read().strip()
    root_logger.debug(f"VERSION tag: {version}")

    # Get the FROM image from the first line of the Dockerfile
    dockerfile = os.path.join(image, "Dockerfile")
    with open(dockerfile, "r") as dockerfile:
        first_line = dockerfile.readline().strip()
    from_image = first_line.split("FROM")[1].strip()
    root_logger.debug(f"FROM image: {from_image}")

    return version, from_image


def main():
    parser = argparse.ArgumentParser(
        description="Build and push the base image and canary images."
    )
    parser.add_argument("--all", action="store_true", help="Build and push all images.")
    parser.add_argument(
        "--check-versions",
        action="store_true",
        help="Check the versions of all images - both VERSION and FROM tags.",
    )
    args = parser.parse_args()

    if not any(vars(args).values()):
        parser.print_help()
        print()
        all_dirs = get_dirs()
        print("Select image to build and push:")
        for idx, image in enumerate(all_dirs, start=1):
            print(f"{idx}. {image}")
        choice = int(input("Enter the number of the image to build: ")) - 1
        if 0 <= choice < len(all_dirs):
            build_docker_image(all_dirs[choice])
        else:
            print("Invalid selection.")
        return

    if args.all:
        images_to_build = get_images_to_build()
        for image, version in images_to_build:
            build_docker_image(image, version)

    if args.check_versions:
        images_to_check = get_images_to_build()

        # Array of Image
        images: List[Image] = []

        for image, version in images_to_check:
            version, from_image = check_versions(image, version)
            images.append(
                {
                    "Name": image,
                    "Latest version": version,
                    "FROM line (first line only)": from_image,
                }
            )

        # Print out images in a pretty table
        print(tabulate(images, headers="keys", showindex=False))


if __name__ == "__main__":
    main()
