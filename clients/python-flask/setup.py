# coding: utf-8

import sys
from setuptools import setup, find_packages

NAME = "bacalhau-client"
VERSION = "0.3.13"
# To install the library, run the following
#
# python setup.py install
#
# prerequisite: setuptools
# http://pypi.python.org/pypi/setuptools

REQUIRES = [
    "connexion",
    "swagger-ui-bundle>=0.0.2"
]

setup(
    name=NAME,
    version=VERSION,
    description="Bacalhau API",
    author_email="team@bacalhau.org",
    url="",
    keywords=["Swagger", "Bacalhau API"],
    install_requires=REQUIRES,
    packages=find_packages(),
    package_data={'': ['swagger/swagger.yaml']},
    include_package_data=True,
    entry_points={
        'console_scripts': ['bacalhau-client=bacalhau-client.__main__:main']},
    long_description="""\
    This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/filecoin-project/bacalhau.
    """
)
