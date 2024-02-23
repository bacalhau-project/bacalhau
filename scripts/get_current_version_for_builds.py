#!/usr/bin/env python3

# Write a script that gets the current version for builds based on the following rules:
# 1. If the build is a release build, the version is the build number.
# 2. If the build is a pre-release build, the version is the build number with the pre-release tag.
# 3. If the build is a nightly build, the version is the build number with the nightly tag.

# The script should read from the entire codebase

import os
from pathlib import Path


def get_current_version_for_builds(for_python=False):
    # Get the current version from the environment variable
    version = os.environ.get("PYPI_VERSION")
    if version:
        return version

    # Get the current version from the git tags
    version = os.popen("git describe --tags --always --dirty").read().strip()
    if version:
        return version

    # Get the current version from the build number
    version = os.environ.get("BUILD_NUMBER")
    if version:
        return version

    return "0.0.0-alpha.0"


if __name__ == "__main__":
    # If there is a flag for --pep440, convert the version to PEP440 compatible version
    # Otherwise, print the version as is. Use the convert_to_pep440 function from the
    # convert_git_version_to_pep440_compatible.py script
    v = get_current_version_for_builds()
    if "--pep440" in os.sys.argv:
        current_directory = os.path.dirname(os.path.realpath(__file__))
        convertor = (
            Path(current_directory) / "convert_git_version_to_pep440_compatible.py"
        )
        v = os.popen(f"python3 {convertor} {v}").read().strip()

    print(v)
