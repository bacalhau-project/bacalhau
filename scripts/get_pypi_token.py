#!/usr/bin/env python

# If PYPI_TOKEN is in the environment, output it to .secrets file
import os

PYPI_TOKEN = ""

# If the PYPI_TOKEN is in the .secrets file, and the user doesn't add the --update flag, we're done
if os.path.exists(".secret") and "--update" not in os.sys.argv:
    with open(".secret", "r") as f:
        lines = f.readlines()
        for line in lines:
            if line.startswith("PYPI_TOKEN"):
                print("PYPI_TOKEN already in .secrets file, exiting")
                exit(0)

# First check if the PYPI_TOKEN is in the environment
if "PYPI_TOKEN" in os.environ:
    PYPI_TOKEN = os.environ["PYPI_TOKEN"]
# If not, check if it's in the .env file
elif os.path.exists(".env"):
    with open(".env", "r") as f:
        lines = f.readlines()
        for line in lines:
            if line.startswith("PYPI_TOKEN"):
                PYPI_TOKEN = line.split("=")[1].strip()

# If we found the PYPI_TOKEN, write it to the .secret file, and delete the line if it's already there
if PYPI_TOKEN:
    with open(".secret", "+tw") as f:
        if os.path.exists(".secret"):
            lines = f.readlines()
            for line in lines:
                if line.startswith("PYPI_TOKEN="):
                    lines.remove(line)
        lines.append(f"PYPI_TOKEN={PYPI_TOKEN}")
        f.writelines(lines)
else:
    print("PYPI_TOKEN not found in environment or .env file")
    exit(1)
