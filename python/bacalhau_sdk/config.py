"""Utils for the SDK."""

import base64
import logging
import os
import stat
from pathlib import Path
from typing import Union

import pem
from bacalhau_apiclient import Configuration
from Crypto.Hash import SHA256
from Crypto.PublicKey import RSA
from Crypto.Signature import pkcs1_15

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


def set_user_id_key(user_id_key: RSA.RsaKey):
    """Set the user ID key."""
    global __user_id_key
    __user_id_key = user_id_key


def init_config():
    """Initialize the config."""
    config_dir = __ensure_config_dir()
    log.debug("config_dir: {}".format(config_dir))
    __ensure_config_file()
    key_path = __ensure_user_id_key(config_dir)
    log.debug("key_path: {}".format(key_path))
    set_user_id_key(__load_user_id_key(key_path))
    log.debug("user_id_key set")
    set_client_id(__load_client_id(key_path))
    log.debug("client_id: {}".format(get_client_id()))

    conf = Configuration()
    if os.getenv("BACALHAU_API_HOST") and os.getenv("BACALHAU_API_PORT"):
        conf.host = "http://{}:{}".format(
            os.getenv("BACALHAU_API_HOST"), os.getenv("BACALHAU_API_PORT")
        )
        log.debug(
            "Using BACALHAU_API_HOST and BACALHAU_API_PORT to set host: %s", conf.host
        )

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
        config_dir.mkdir(mode=700, parents=True, exist_ok=True)
    else:
        os.stat(config_dir_str)
        config_dir = Path(config_dir_str)
    log.debug("Using config dir: %s", config_dir.absolute().resolve())
    return config_dir.absolute().resolve()


def __ensure_config_file() -> str:
    """Ensure that BACALHAU_DIR/config.yaml exists."""
    # warnings.warn("Not implemented - the BACALHAU_DIR/config.yaml config file is not used yet.")
    return ""


def __ensure_user_id_key(config_dir: Path) -> Path:
    """Ensure that a default user ID key exists in the config dir."""
    key_file_name = "user_id.pem"
    user_id_key = config_dir.joinpath(key_file_name)
    if not os.path.isfile(user_id_key):
        log.info("User ID key not found at %s, generating one.", user_id_key)
        key = RSA.generate(bits_per_key)
        with open(user_id_key, "wb") as f:
            f.write(key.export_key("PEM", pkcs=1))
        os.chmod(user_id_key, stat.S_IRUSR | stat.S_IWUSR)
    else:
        log.info("Found user ID key at %s", user_id_key)
    return user_id_key


def __load_user_id_key(key_file: Path) -> RSA.RsaKey:
    """Return the private key."""
    log.debug("Loading user ID key from %s", key_file)
    with open(key_file, "rb") as f:
        certs = pem.parse(f.read())
        private_key = RSA.import_key(certs[0].as_bytes())
    return private_key


def __convert_to_client_id(key: RSA.RsaKey) -> str:
    """Return the client ID from a public key.

    e.g. `bae8c1b2adfa04cc647a2457e8c0c605cef8ed11bdea5ac1f19f94219d722dfz`.
    """
    der_key = key.public_key()
    hash_obj = SHA256.new()
    hash_obj.update(der_key.n.to_bytes(der_key.size_in_bytes(), byteorder="big"))

    return hash_obj.hexdigest()


def __load_client_id(key_path: Path) -> str:
    """Return the client ID.

    Should be callable without the need of invoking init_config() first.
    """
    key = __load_user_id_key(key_path)
    return __convert_to_client_id(key)


def sign_for_client(msg: bytes) -> str:
    """sign_for_client signs a message with the user's private ID key.

    Must be called after init_config().
    """
    signer = pkcs1_15.new(get_user_id_key())
    hash_obj = SHA256.new()
    hash_obj.update(msg)

    signed_payload = signer.sign(hash_obj)
    signature = base64.b64encode(signed_payload).decode()

    return signature


def __clean_pem_pub_key(pem_pub_key: str) -> str:
    """Prepare a public key in the format expected by the API.

    - remove the header and footer
    - remove the newlines
    - remove the first 32 characters i.e. `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A`. \
        https://stackoverflow.com/questions/8784905/command-line-tool-to-export-rsa-private-key-to-rsapublickey
    """
    pem_public_key = (
        pem_pub_key.replace("-----BEGIN PUBLIC KEY-----", "")
        .replace("-----END PUBLIC KEY-----", "")
        .replace("\n", "")[32:]
    )
    return pem_public_key


def get_client_public_key() -> str:
    """Return the client public key."""
    public_key = get_user_id_key().publickey()
    pem_public_key = public_key.export_key("PEM").decode()
    return __clean_pem_pub_key(pem_public_key)
