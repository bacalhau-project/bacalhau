#!/usr/bin/env python3

import sys

print(sys.version)

# Test to see if the python version is the same as the one in .tool-versions
# If not, print an error message and exit with a non-zero status code

# Get the version from the .tool-versions file
with open(".tool-versions") as f:
    lines = f.readlines()
    for line in lines:
        if "python" in line:
            version = line.split(" ")[1].strip()
            break

# Compare the versions
if sys.version.split(" ")[0] != version:
    print(
        f"Error: Python version is {sys.version.split(' ')[0]}, but should be {version}"
    )
    sys.exit(1)
