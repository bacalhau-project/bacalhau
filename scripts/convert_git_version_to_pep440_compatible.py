#!/usr/bin/env python3

# Convert git version to PEP440 compatible version
# This script is used to convert git version to PEP440 compatible version
# The git version is in the format of "v1.2.3-4-g5c6d7e8"
# The PEP440 compatible version is in the format of "1.2.3.post4+g5c6d7e8"
# The "v" prefix is removed and the last commit hash is appended to the version
# The commit count is appended to the version as a post release version
# The script will print the PEP440 compatible version to stdout

import re

EXAMPLE_STRINGS = [
    {"v1.2.3": "1.2.3"},
    {"v1.2.3-4-g5c6d7e8": "1.2.3.dev4.g5c6d7e8"},
    {"v1.2.3-4-g5c6d7e8-dirty": "1.2.3.dev4.g5c6d7e8.dirty"},
    {"v1.2.2-rc1-113-g5b54992f2-dirty": "1.2.2.rc1.dev113.g5b54992f2.dirty"},
]


def convert_to_pep440(version_string):
    # Remove the leading 'v'
    version_string = version_string[1:]

    # Regular expressions to match different parts of the version string
    main_version_pattern = r"^(\d+\.\d+\.\d+)"
    pre_release_pattern = r"(?:-rc(\d+))?"
    dev_release_pattern = r"(?:-(\d+)+)?"
    commit_hash_pattern = r"(?:-g([^\-\.]+))?"
    dirty_pattern = r"([-.]dirty)?$"

    # Combine the patterns
    full_pattern = (
        main_version_pattern
        + pre_release_pattern
        + dev_release_pattern
        + commit_hash_pattern
        + dirty_pattern
    )

    # Match the version string against the pattern
    match = re.match(full_pattern, version_string)

    if not match:
        raise ValueError(f"Invalid version string: {version_string}")

    components = {}

    # Extract the components
    main_version = match.group(1)
    if len(match.groups()) > 1:
        components["pre_release"] = match.group(2)
    if len(match.groups()) > 2:
        components["dev_release"] = match.group(3)
    if len(match.groups()) > 3:
        components["commit_hash"] = match.group(4)
    if len(match.groups()) > 4:
        components["dirty"] = match.group(5)

    pep440_version = main_version
    if "dev_release" in components and components["dev_release"] is not None:
        pep440_version += f".dev{components['dev_release']}"

    return pep440_version


def test_convert_git_version_to_pep440_compatible():
    print("Testing convert_git_version_to_pep440_compatible")

    # Test the example strings
    for example in EXAMPLE_STRINGS:
        git_version, pep440_version = example.popitem()
        print(f"Testing {git_version} -> {pep440_version}")

        converted = convert_to_pep440(git_version)

        assert pep440_version == converted or print(
            f"Failed: \n Expected: {pep440_version}\n Got: {converted}"
        )


if __name__ == "__main__":
    # If the script has an arg then use it as the git version
    import sys

    if len(sys.argv) > 1:
        print(convert_to_pep440(sys.argv[1]))
    else:
        test_convert_git_version_to_pep440_compatible()
