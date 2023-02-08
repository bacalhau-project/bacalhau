#!/usr/bin/env python

"""The setup script."""

from setuptools import setup, find_packages

with open('README.md') as readme_file:
    readme = readme_file.read()


requirements = []

test_requirements = []

setup(
    author="Enrico Rotundo",
    author_email='team@bacalhau.org',
    python_requires='>=3.7',
    classifiers=[
        'Development Status :: 2 - Pre-Alpha',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: Apache Software License',
        'Natural Language :: English',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
    ],
    description="Python Boilerplate contains all the boilerplate you need to create a Python package.",
    install_requires=requirements,
    license="Apache Software License 2.0",
    long_description=readme,
    include_package_data=True,
    keywords='bacalhau',
    name='bacalhau',
    packages=find_packages(include=['bacalhau', 'bacalhau.*']),
    test_suite='tests',
    tests_require=test_requirements,
    url='https://github.com/enricorotundo/bacalhau',
    version='0.1.0',
    zip_safe=False,
)
