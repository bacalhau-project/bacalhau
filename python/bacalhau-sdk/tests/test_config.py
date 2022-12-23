#!/usr/bin/env python
"""Tests for `bacalhau_sdk` package."""

import logging
import os
from pathlib import Path
from tempfile import TemporaryDirectory

import pytest
from Crypto.PublicKey import RSA

log = logging.getLogger(__name__)


@pytest.fixture
def response():
    """Sample pytest fixture.

    See more at: http://doc.pytest.org/en/latest/fixture.html
    """


def test_ensure_config_dir():
    """Test ensure_config_dir()."""
    from bacalhau_sdk.config import __ensure_config_dir

    config_dir = __ensure_config_dir()
    home_path = Path.home()
    assert config_dir.exists()
    assert config_dir.is_dir()
    assert config_dir.is_absolute()
    assert config_dir == home_path.joinpath(".bacalhau")


def test_ensure_config_file():
    """Test ensure_config_file()."""
    from bacalhau_sdk.config import __ensure_config_file

    assert __ensure_config_file() == ""


def test_ensure_user_id_key():
    """Test ensure_user_id_key()."""
    from bacalhau_sdk.config import __ensure_user_id_key

    home_path = Path.home()
    config_dir = home_path.joinpath(".bacalhau")
    user_id_key_path = __ensure_user_id_key(config_dir)
    assert user_id_key_path.exists()
    assert user_id_key_path.is_file()
    assert user_id_key_path.is_absolute()
    assert user_id_key_path == config_dir.joinpath("user_id.pem")


def test_load_user_id_key():
    """Test load_user_id_key()."""
    from bacalhau_sdk.config import __load_user_id_key

    # test generated key
    with TemporaryDirectory(prefix="bacalhau_sdk_test-") as tmpdirname:
        tmpdir = Path(tmpdirname)
        key = RSA.generate(2048)
        with open(tmpdir.joinpath('mykey.pem'), 'wb') as f:
            f.write(key.export_key('PEM'))
        loaded_key = __load_user_id_key(tmpdir.joinpath('mykey.pem'))
        assert type(loaded_key) == RSA.RsaKey
        assert loaded_key.export_key('PEM') == key.export_key('PEM')


def test_load_client_id():
    """Test load_client_id()."""
    from bacalhau_sdk.config import __load_client_id

    with TemporaryDirectory(prefix="bacalhau_sdk_test-") as tmpdirname:
        tmpdir = Path(tmpdirname)
        key = RSA.generate(2048)
        with open(tmpdir.joinpath('mykey.pem'), 'wb') as f:
            f.write(key.export_key('PEM'))
        client_id = __load_client_id(tmpdir.joinpath('mykey.pem'))
        assert len(client_id) == 64


def test_init_config():
    """Test init_config()."""
    from bacalhau_apiclient.configuration import Configuration

    from bacalhau_sdk.config import init_config

    os.environ['BACALHAU_API_HOST'] = "1.1.1.1"
    os.environ['BACALHAU_API_PORT'] = "9999"
    conf = init_config()
    assert type(conf) == Configuration
    assert conf.host == "http://1.1.1.1:9999"

    os.environ['BACALHAU_API_HOST'] = ""
    os.environ['BACALHAU_API_PORT'] = ""
    conf = init_config()
    assert type(conf) == Configuration
    assert conf.host == "http://bootstrap.production.bacalhau.org:1234"


def test_sign_for_client():
    """Test sign_for_client()."""
    import base64

    from bacalhau_apiclient.api import job_api
    from bacalhau_apiclient.models.deal import Deal
    from bacalhau_apiclient.models.job_create_payload import JobCreatePayload
    from bacalhau_apiclient.models.job_execution_plan import JobExecutionPlan
    from bacalhau_apiclient.models.job_sharding_config import JobShardingConfig
    from bacalhau_apiclient.models.job_spec_docker import JobSpecDocker
    from bacalhau_apiclient.models.job_spec_language import JobSpecLanguage
    from bacalhau_apiclient.models.spec import Spec
    from bacalhau_apiclient.models.storage_spec import StorageSpec
    from Crypto.Hash import SHA256
    from Crypto.Signature import pkcs1_15

    from bacalhau_sdk.config import get_client_id, get_user_id_key, hashable_value, init_config, sign_for_client

    _ = init_config()

    test_payload = JobCreatePayload(
        api_version='V1beta1',
        client_id=get_client_id(),
        spec=Spec(
            engine="Docker",
            verifier="Noop",
            publisher="Estuary",
            docker=JobSpecDocker(
                image="ubuntu",
                entrypoint=["date"],
            ),
            language=JobSpecLanguage(job_context=None),
            wasm=None,
            resources=None,
            timeout=1800,
            outputs=[
                StorageSpec(
                    storage_source="IPFS",
                    name="outputs",
                    path="/outputs",
                )
            ],
            sharding=JobShardingConfig(
                batch_size=1,
                glob_pattern_base_path="/inputs",
            ),
            execution_plan=JobExecutionPlan(shards_total=0),
            deal=Deal(concurrency=1, confidence=0, min_bids=0),
            do_not_track=False,
        ),
    )

    signature = sign_for_client(test_payload)
    assert signature is not None
    assert len(signature) == 344
    assert signature.endswith("==")

    client = job_api.ApiClient()
    sanitized_payload = client.sanitize_for_serialization(test_payload)
    hashable_payload = hashable_value(sanitized_payload)

    # check returned signature and generated signature match
    signer = pkcs1_15.new(get_user_id_key())
    hash_obj = SHA256.new()
    hash_obj.update(hashable_payload)
    signed_payload = signer.sign(hash_obj)
    assert signature == base64.b64encode(signed_payload).decode()

    # verify signature has been generated with the public key
    verifier = pkcs1_15.new(get_user_id_key())
    hash_obj = SHA256.new()
    hash_obj.update(hashable_payload)
    verifier.verify(hash_obj, base64.b64decode(signature.encode()))


def test_get_client_public_key():
    """Test __clean_pem_pub_key()."""
    from bacalhau_sdk.config import get_client_public_key

    pub_key = get_client_public_key()
    assert pub_key is not None
    assert type(pub_key) == str
    assert len(pub_key) == 360
    assert "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A" not in pub_key
    assert "BEGIN PUBLIC KEY" not in pub_key
    assert "END PUBLIC KEY" not in pub_key


def test_hashable_value():
    """Test hashable_value()."""
    # from bacalhau_sdk.config import hashable_value
    # TODO @enricorotundo https://github.com/filecoin-project/bacalhau/issues/1555
    pass
