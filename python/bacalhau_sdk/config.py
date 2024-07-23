"""Utils for the SDK."""

import logging
import os
import stat
from pathlib import Path
from typing import Union
from urllib.parse import urlparse

from bacalhau_apiclient import Configuration


__client_id = None
__user_id_key = None
bits_per_key = 2048
log = logging.getLogger(__name__)
log.setLevel(logging.DEBUG)


def get_client_id() -> Union[str, None]:
    """Return the client ID."""
    global __client_id
    return __client_id


def set_client_id(client_id: str):
    """Set the client ID."""
    global __client_id
    __client_id = client_id
    log.debug("set client_id to %s", __client_id)


def get_user_id_key():
    """Return the user ID key."""
    global __user_id_key
    return __user_id_key


def init_config():
    """Initialize the config."""
    config_dir = __ensure_config_dir()
    log.debug("config_dir: {}".format(config_dir))
    __ensure_config_file()
    conf = Configuration()
    # Parse out defaults and override with environment variables if they exist
    # before setting the configuration host.
    u = urlparse(conf.host)
    api_scheme: str = "http"
    scheme: str = os.getenv("BACALHAU_HTTPS", "")
    if scheme:
        api_scheme = "https"

    api_host: str = os.getenv("BACALHAU_API_HOST", u.hostname)
    api_port: str = os.getenv("BACALHAU_API_PORT", str(u.port))

    conf.host = "{}://{}:{}".format(api_scheme, api_host, api_port)
    log.debug("Host is set to: %s", conf.host)

    # Remove trailing slash from host
    if conf.host[-1] == "/":
        conf.host = conf.host[:-1]

    log.debug("init config done")
    return conf


def __ensure_config_dir() -> Path:
    """Ensure the config directory exists and return its path."""
    config_dir_str = os.getenv("BACALHAU_DIR")
    if config_dir_str == "" or config_dir_str is None:
        log.debug("BACALHAU_DIR not set, using default of ~/.bacalhau")
        home_path = Path.home()
        config_dir = home_path.joinpath(".bacalhau")
        config_dir.mkdir(mode=stat.S_IRWXU, parents=True, exist_ok=True)
    else:
        os.stat(config_dir_str)
        config_dir = Path(config_dir_str)
    log.debug("Using config dir: %s", config_dir.absolute().resolve())
    return config_dir.absolute().resolve()


def __ensure_config_file() -> str:
    """Ensure that BACALHAU_DIR/config.yaml exists."""
    # warnings.warn("Not implemented - the BACALHAU_DIR/config.yaml config file is not used yet.")
    return ""
