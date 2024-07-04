#!/usr/bin/env python
"""Tests for `bacalhau_sdk` package."""

import logging
import os
from pathlib import Path
from tempfile import TemporaryDirectory

log = logging.getLogger(__name__)


def test_ensure_config_dir():
    """Test ensure_config_dir()."""
    from bacalhau_sdk.config import __ensure_config_dir

    # test it creates a default config dir
    config_dir = __ensure_config_dir()
    home_path = Path.home()
    assert config_dir.exists()
    assert config_dir.is_dir()
    assert config_dir.is_absolute()
    assert config_dir == home_path.joinpath(".bacalhau")

    # test it detects an existing config dir
    with TemporaryDirectory() as config_dir:
        os.environ["BACALHAU_DIR"] = config_dir
        config_dir = __ensure_config_dir()
        assert config_dir.exists()
        assert config_dir.is_dir()
        assert config_dir.is_absolute()
        # clean up
        os.environ["BACALHAU_DIR"] = ""


def test_ensure_config_file():
    """Test ensure_config_file()."""
    from bacalhau_sdk.config import __ensure_config_file

    assert __ensure_config_file() == ""


def test_init_config():
    """Test init_config()."""
    from bacalhau_apiclient.configuration import Configuration
    from bacalhau_sdk.config import init_config

    conf = init_config()
    assert isinstance(conf, Configuration)
    assert conf.host == "http://bootstrap.production.bacalhau.org:1234"

    os.environ["BACALHAU_HTTPS"] = "1"
    conf = init_config()
    assert isinstance(conf, Configuration)
    assert conf.host == "https://bootstrap.production.bacalhau.org:1234"
    del os.environ["BACALHAU_HTTPS"]

    os.environ["BACALHAU_API_HOST"] = "1.1.1.1"
    os.environ["BACALHAU_API_PORT"] = "9999"
    conf = init_config()
    assert isinstance(conf, Configuration)
    assert conf.host == "http://1.1.1.1:9999"

    del os.environ["BACALHAU_API_HOST"]
    os.environ["BACALHAU_API_PORT"] = "4321"
    conf = init_config()
    assert isinstance(conf, Configuration)
    assert conf.host == "http://bootstrap.production.bacalhau.org:4321"

    os.environ["BACALHAU_API_HOST"] = "mycluster.com"
    del os.environ["BACALHAU_API_PORT"]
    conf = init_config()
    assert isinstance(conf, Configuration)
    assert conf.host == "http://mycluster.com:1234"
