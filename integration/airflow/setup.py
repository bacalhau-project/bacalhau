#!/usr/bin/env python

"""The setup script."""
import os

from setuptools import find_packages, setup

with open("README.md") as readme_file:
    readme = readme_file.read()

pypi_version = os.getenv("PYPI_VERSION", "0.0.0")


requirements = [
    "bacalhau_sdk",
    "apache-airflow>=2.3.0",
]

test_requirements = []

setup(
    author="Enrico Rotundo",
    author_email="team@bacalhau.org",
    python_requires=">=3.8",
    classifiers=[
        "Development Status :: 2 - Pre-Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: Apache Software License",
        "Natural Language :: English",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
    ],
    description="An Apache Airflow provider for Bacalhau.",
    install_requires=requirements,
    license="Apache Software License 2.0",
    long_description=readme,
    long_description_content_type="text/markdown",
    include_package_data=True,
    keywords=["bacalhau", "airflow", "provider"],
    name="bacalhau_airflow",
    packages=find_packages(include=["bacalhau_airflow", "bacalhau_airflow.*"]),
    test_suite="tests",
    tests_require=test_requirements,
    url="https://github.com/filecoin-project/bacalhau/tree/main/integration/airflow",
    version=pypi_version,
    zip_safe=False,
)
