from setuptools import setup

PLUGIN_NAME = "bacalhau"

microlib_name = f"flytekitplugins-{PLUGIN_NAME}"

# TODO add additional requirements
plugin_requires = [
    "flytekit>=1.1.0b0,<=1.8.1",
    "bacalhau-sdk>=1.0.3",
    "dataclasses-json",
    "marshmallow",
    "marshmallow-dataclass",
    "marshmallow-enum>=1.5.1",
]

__version__ = ""

setup(
    name=microlib_name,
    version=__version__,
    author="flyteorg",
    author_email="admin@flyte.org",
    # TODO Edit the description
    description="My awesome plugin.....",
    # TODO alter the last part of the following URL
    url="https://github.com/flyteorg/flytekit/tree/master/plugins/flytekit-...",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    namespace_packages=[
        "flytekitplugins"
    ],
    packages=[f"flytekitplugins.{PLUGIN_NAME}"],
    install_requires=plugin_requires,
    license="apache2",
    python_requires=">=3.8",
    classifiers=[
        "Intended Audience :: Science/Research",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: Apache Software License",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Topic :: Scientific/Engineering",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "Topic :: Software Development",
        "Topic :: Software Development :: Libraries",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    # TODO OPTIONAL
    # FOR Plugins where auto-loading on installation is desirable, please uncomment this line and ensure that the
    # __init__.py has the right modules available to be loaded, or point to the right module
    # entry_points={"flytekit.plugins": [f"{PLUGIN_NAME}=flytekitplugins.{PLUGIN_NAME}"]},
)