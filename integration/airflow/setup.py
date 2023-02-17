#!/usr/bin/env python

"""The setup script."""

from setuptools import find_packages, setup

with open("README.md") as readme_file:
    readme = readme_file.read()


requirements = [
    # "Package-A @ git+https://example.net/package-a.git@main",
    # "bacalhau_sdk==0.1.2",
    # "bacalhau_sdk @ git+https://github.com/filecoin-project/bacalhau.git@main#egg=bacalhau_sdk&subdirectory=python"
    "bacalhau_sdk @ git+https://github.com/filecoin-project/bacalhau.git@7c2b6208538a28f558f5de21c34a49e4c58c0f76#egg=bacalhau_sdk&subdirectory=python",
    "apache-airflow>=2.3.0",
]

test_requirements = []

setup(
    author="Enrico Rotundo",
    author_email="team@bacalhau.org",
    python_requires=">=3.7",
    classifiers=[
        "Development Status :: 2 - Pre-Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: Apache Software License",
        "Natural Language :: English",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
    ],
    description="Python Boilerplate contains all the boilerplate you need to create a Python package.",
    install_requires=requirements,
    license="Apache Software License 2.0",
    long_description=readme,
    include_package_data=True,
    keywords="bacalhau",
    name="bacalhau",
    packages=find_packages(include=["bacalhau", "bacalhau.*"]),
    test_suite="tests",
    tests_require=test_requirements,
    url="https://github.com/enricorotundo/bacalhau",
    version="0.1.0",
    zip_safe=False,
)
