#!/usr/bin/env python3

import argparse
import asyncio
import json
import logging
import os
import subprocess
from typing import List, TypedDict

import semver
from alive_progress import alive_bar
from tabulate import tabulate

import docker

DOCKER_REGISTRY_URL = "docker.io"
GCR_REGISTRY_URL = "gcr.io"
DOCKER_REPO = "bacalhauproject"
PROJECT = "bacalhau-infra"


class Image(TypedDict):
    name: str
    version: str
    from_image: str


class RepetitiveLogFilter(logging.Filter):
    def __init__(self):
        self.last_log = None

    def filter(self, record):
        current_log = (record.levelname, record.msg)
        if current_log != self.last_log:
            self.last_log = current_log
            return True
        return False


logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
root_logger = logging.getLogger()
repetitive_log_filter = RepetitiveLogFilter()
root_logger.addFilter(repetitive_log_filter)


def check_docker_authentication():
    config_path = os.path.expanduser("~/.docker/config.json")
    if not os.path.exists(config_path):
        print("Error: Docker config file not found. Please log in to Docker first.")
        print("To log in, run: docker login")
        return False

    with open(config_path, "r") as config_file:
        config_data = json.load(config_file)
        if "auths" in config_data and (
            DOCKER_REGISTRY_URL in config_data["auths"]
            or "https://index.docker.io/v1/" in config_data["auths"]
        ):
            return True
        else:
            print("Error: Docker Hub credentials not found in Docker config file.")
            print("To log in, run: docker login")
            return False


def check_gcr_authentication():
    try:
        result = subprocess.run(
            ["gcloud", "auth", "print-access-token"],
            capture_output=True,
            text=True,
            check=True,
        )
        if result.stdout.strip():
            return True
    except subprocess.CalledProcessError:
        pass
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


def increment_version(version) -> str:
    v = semver.VersionInfo.parse(version)
    return str(v.bump_patch())


def build_docker_image(docker_client, image: Image):
    docker_image = f"{DOCKER_REGISTRY_URL}/{DOCKER_REPO}/{image['name']}"
    gcr_image = f"{GCR_REGISTRY_URL}/{PROJECT}/{image['name']}"

    if image["version"] is None:
        new_version = "0.0.1"  # Set a default version if None
    else:
        new_version = increment_version(image["version"])

    version_file_path = os.path.join(image["name"], "VERSION")
    with open(version_file_path, "w") as version_file:
        version_file.write(str(new_version))

    tag_list = [
        f"{docker_image}:{new_version}",
        f"{docker_image}:latest",
    ]

    if check_gcr_authentication():
        tag_list.extend(
            [
                f"{gcr_image}:{new_version}",
                f"{gcr_image}:latest",
            ]
        )

    try:
        dockerfile_path = os.path.join(image["name"], "Dockerfile")
        if not os.path.exists(dockerfile_path):
            raise FileNotFoundError(f"Dockerfile not found at {dockerfile_path}")

        print(f"Building image {docker_image}:{new_version}")
        build_params = {
            "path": image["name"],
            "dockerfile": "Dockerfile",
            "tag": f"{docker_image}:{new_version}",
            "platform": "linux/amd64",
        }

        with alive_bar(title="Building", unknown="bubbles") as bar:
            try:
                for line in docker_client.api.build(**build_params, decode=True):
                    if "stream" in line:
                        bar()
                    elif "error" in line:
                        raise Exception(line["error"])
            except docker.errors.BuildError as e:
                print(f"Build failed: {str(e)}")
                return None

        for tag in tag_list:
            docker_client.images.get(f"{docker_image}:{new_version}").tag(tag)

        docker_client.images.get(f"{docker_image}:{new_version}")
    except Exception as e:
        print(f"Error during build: {e}")
        return None

    return tag_list


async def push_docker_image(docker_client, tag_list, timeout=300):
    async def push_single_image(tag):
        try:
            repository, version = tag.split(":")
            response = docker_client.images.push(repository, tag=version, stream=True)
            with alive_bar(
                title=f"Pushing {tag}", bar="bubbles", unknown="stars"
            ) as bar:
                buffer = ""
                for chunk in response:
                    buffer += chunk.decode("utf-8")
                    while "\n" in buffer:
                        line, buffer = buffer.split("\n", 1)
                        if line:
                            try:
                                chunk_data = json.loads(line)
                                if "error" in chunk_data:
                                    raise Exception(chunk_data["error"])
                                if "status" in chunk_data:
                                    if "progress" in chunk_data:
                                        bar.text(chunk_data["progress"])
                                    bar()
                            except json.JSONDecodeError as e:
                                print(f"Error decoding JSON for {tag}: {e}")
        except Exception as e:
            print(f"Error pushing image {tag}: {e}")
            return False
        return True

    print("Pushing images...")
    tasks = [push_single_image(tag) for tag in tag_list]
    results = await asyncio.gather(*tasks)

    return all(results)


def push_docker_image_sync(docker_client, tag_list, timeout=300):
    loop = asyncio.get_event_loop()
    return loop.run_until_complete(push_docker_image(docker_client, tag_list, timeout))


def get_images_to_build() -> List[Image]:
    images_to_build: List[Image] = []

    def search_directories(current_dir):
        for item in os.listdir(current_dir):
            item_path = os.path.join(current_dir, item)
            if os.path.isdir(item_path) and not item.startswith("."):
                dockerfile_path = os.path.join(item_path, "Dockerfile")
                if os.path.exists(dockerfile_path):
                    version, from_image = check_versions(item_path, dockerfile_path)
                    images_to_build.append(
                        {
                            "name": os.path.relpath(item_path, start="."),
                            "version": version,
                            "from_image": from_image,
                        }
                    )

                search_directories(item_path)

    search_directories(".")
    return images_to_build


def check_versions(image_dir, dockerfile_path):
    # Get the version from the VERSION file
    version_file_path = os.path.join(image_dir, "VERSION")
    version = get_latest_version_from_file(version_file_path)

    # Get the FROM image from the first line of the Dockerfile
    with open(dockerfile_path, "r") as dockerfile:
        first_line = dockerfile.readline().strip()
    from_image = first_line.split("FROM")[1].strip()

    return version, from_image


def get_docker_client():
    return docker.from_env()


def main():
    parser = argparse.ArgumentParser(description="Build and push Docker images.")
    parser.add_argument("--all", action="store_true", help="Build and push all images.")
    parser.add_argument(
        "--check-versions",
        action="store_true",
        help="Check the versions of all images - both VERSION and FROM tags.",
    )
    parser.add_argument(
        "--build",
        nargs="?",
        const="menu",
        help="Build a specific image or show menu if no argument is provided.",
    )
    args = parser.parse_args()

    if args.all:
        images_to_build = get_images_to_build()
        for image in images_to_build:
            tag_list = build_docker_image(get_docker_client(), image)
            if tag_list:
                push_docker_image(get_docker_client(), tag_list)

    elif args.check_versions:
        images_to_check = get_images_to_build()
        images: List[Image] = []
        for image in images_to_check:
            images.append(image)

        table = tabulate(
            [
                [
                    image["name"],
                    image["version"],
                    image["from_image"],
                ],
            ],
            headers=["Image", "Version", "From"],
            tablefmt="grid",
        )
        print(table)

    elif args.build:
        all_images = get_images_to_build()
        if args.build == "menu":
            print("Select an image to build:")
            for i, image in enumerate(all_images):
                print(f"{i + 1}. {image['name']}")
            choice = input("Enter the number of the image to build: ")
            try:
                choice = int(choice) - 1
                if 0 <= choice < len(all_images):
                    tag_list = build_docker_image(
                        get_docker_client(), all_images[choice]
                    )
                    if tag_list:
                        success = push_docker_image_sync(get_docker_client(), tag_list)
                        if not success:
                            print("Image push was not completed successfully")
                else:
                    print("Invalid selection.")
            except ValueError:
                print("Invalid input. Please enter a number.")
        else:
            image_to_build = next(
                (img for img in all_images if img["name"] == args.build), None
            )
            if image_to_build:
                tag_list = build_docker_image(get_docker_client(), image_to_build)
                if tag_list:
                    push_docker_image_sync(get_docker_client(), tag_list)
            else:
                print(f"Image '{args.build}' not found.")
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
