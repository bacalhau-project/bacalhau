#!/usr/bin/env python3
import ast
from glob import glob
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile

IGNORE = (
    "*.pyc",
    ".DS_Store",
    "__pycache__",
)

CODE_DIR = "/code"  # The mounted code folder
WORKING_DIR = "/app"  # Created by the shutil.copytree
OUTPUT_DIR = "/outputs"  # The output folder


def main():
    # Unpack the contents of /code to the working directory which
    # will create that working_directory, ignoring the files that
    # match the globs in IGNORE
    ignore_pattern = shutil.ignore_patterns(*IGNORE)
    shutil.copytree(CODE_DIR, WORKING_DIR, ignore=ignore_pattern)
    os.chdir(WORKING_DIR)

    # Figure out how to install requirements
    for f in (
        single_file,
        pyproject,
        requirements_txt,
        setup_py,
    ):
        if f():
            break

    # Run the program in that working directory
    past = False
    args = []
    for a in sys.argv:
        if past:
            args.append(a)
        if a == "--":
            past = True

    cmd = " ".join(args)
    proc = subprocess.run(cmd, capture_output=False, shell=True, cwd=WORKING_DIR)


def to_requirements_log(stdoutBytes, stderrBytes):
    if os.path.exists(OUTPUT_DIR):
        name = os.path.join(OUTPUT_DIR, "requirements.log")
        with open(name, "w") as f:
            f.write("================================== STDOUT\n")
            f.write(stdoutBytes.decode("utf-8"))
            f.write("\n================================== STDERR\n")
            f.write(stderrBytes.decode("utf-8"))


def single_file():
    """
    If we only find a single file ready to be deployed, we'll read pip install instrcutions
    from the module doc (if it exists).
    """
    installed = 0
    doclines = []
    files = glob("*.py", root_dir=WORKING_DIR)
    if len(files) == 1:
        with open(os.path.join(WORKING_DIR, files[0])) as f:
            mod = ast.parse(f.read())
            if not mod:
                return False

            doc = ast.get_docstring(mod)
            if not doc:
                return False

            doclines = doc.split("\n")

    for line in doclines:
        line = line.strip()
        if line.startswith("pip"):
            proc = subprocess.run(
                f"python -m{line}", capture_output=True, shell=True, cwd=WORKING_DIR
            )
            to_requirements_log(proc.stdout, proc.stderr)

            installed = installed + 1

    return installed > 0


def pyproject():
    """
    If there is a pyproject.toml we'll check to see if it is a poetry app, and if
    so then we will get poetry to install dependencies.  If not then we will attempt
    to pip install them.
    """
    pth = os.path.join(WORKING_DIR, "pyproject.toml")
    if not os.path.exists(pth):
        return False

    is_poetry = False

    with open(pth) as f:
        contents = f.read()
        is_poetry = "[tool.poetry]" in contents

    cmd = "poetry install"
    if not is_poetry:
        cmd = f"python -mpip install {pth}"

    proc = subprocess.run(cmd, capture_output=True, shell=True, cwd=WORKING_DIR)
    to_requirements_log(proc.stdout, proc.stderr)

    return True


def requirements_txt():
    """
    Look for a requirements file (or several) based on common names to load the
    dependencies from
    """
    installed = 0
    files = ("dev-requirements.txt", "requirements-dev.txt", "requirements.txt")
    for f in files:
        pth = os.path.join(WORKING_DIR, f)
        if os.path.exists(pth):
            proc = subprocess.run(
                f"python -mpip install -r {f}",
                capture_output=True,
                shell=True,
                cwd=WORKING_DIR,
            )
            to_requirements_log(proc.stdout, proc.stderr)

            installed = installed + 1

    return installed > 0


def setup_py():
    """
    Look for a setup.py file as a last resort and try to install it locally
    """
    pth = os.path.join(WORKING_DIR, "setup.py")
    if os.path.exists(pth):
        proc = subprocess.run(
            f"python -m pip install -e .",
            capture_output=True,
            shell=True,
            cwd=WORKING_DIR,
        )
        to_requirements_log(proc.stdout, proc.stderr)
        return True

    return False


if __name__ == "__main__":
    main()
