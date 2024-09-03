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
    {"v1.2.3-4-g5c6d7e8": "1.2.3.dev4+g5c6d7e8"},
    {"v1.2.3-4-g5c6d7e8-dirty": "1.2.3.dev4+g5c6d7e8.dirty"},
    {"v1.2.2-rc1-113-g5b54992f2-dirty": "1.2.2.rc1.dev113+g5b54992f2.dirty"},
    {"v1.5.0-dev2": "1.5.0.dev2"},
    {"v1.5.0-alpha1": "1.5.0.alpha1"},
    {"v1.5.0-beta2": "1.5.0.beta2"},
    {"v1.5.0-alpha1-dev3": "1.5.0.alpha1.dev3"},
    {"v1.5.0-beta2-5-g1234567": "1.5.0.beta2.dev5+g1234567"},
]


def convert_to_pep440(version_string):
    # Remove the leading 'v' if present
    if version_string.startswith("v"):
        version_string = version_string[1:]

    # Regular expressions to match different parts of the version string
    main_version_pattern = r"^(\d+\.\d+\.\d+)"
    pre_release_pattern = r"(?:-(alpha|beta|rc)(\d+))?"
    dev_release_pattern = r"(?:-dev(\d+))?"
    commit_count_pattern = r"(?:-(\d+))?"
    commit_hash_pattern = r"(?:-g([^\-\.]+))?"
    dirty_pattern = r"(-dirty)?$"

    # Combine the patterns
    full_pattern = (
        main_version_pattern
        + pre_release_pattern
        + dev_release_pattern
        + commit_count_pattern
        + commit_hash_pattern
        + dirty_pattern
    )

    # Match the version string against the pattern
    match = re.match(full_pattern, version_string)

    if not match:
        raise ValueError(f"Invalid version string: {version_string}")

    # Extract the components
    main_version = match.group(1)
    pre_release_type = match.group(2)
    pre_release_num = match.group(3)
    dev_release = match.group(4)
    commit_count = match.group(5)
    commit_hash = match.group(6)
    dirty = match.group(7)

    # Construct the PEP440 compatible version
    pep440_version = main_version

    if pre_release_type and pre_release_num:
        pep440_version += f".{pre_release_type}{pre_release_num}"

    if dev_release:
        pep440_version += f".dev{dev_release}"
    elif commit_count:
        pep440_version += f".dev{commit_count}"

    if commit_hash:
        pep440_version += f"+g{commit_hash}"

    if dirty:
        pep440_version += ".dirty"

    return pep440_version


def test_convert_git_version_to_pep440_compatible():
    print("Testing convert_git_version_to_pep440_compatible")

    for example in EXAMPLE_STRINGS:
        git_version, expected_pep440_version = list(example.items())[0]
        print(f"Testing {git_version} -> {expected_pep440_version}")

        converted = convert_to_pep440(git_version)

        assert (
            expected_pep440_version == converted
        ), f"Failed: \n Expected: {expected_pep440_version}\n Got: {converted}"

    print("All tests passed successfully!")


if __name__ == "__main__":
    # If the script has an arg then use it as the git version
    import sys

    if len(sys.argv) > 1:
        print(convert_to_pep440(sys.argv[1]))
    else:
        test_convert_git_version_to_pep440_compatible()
