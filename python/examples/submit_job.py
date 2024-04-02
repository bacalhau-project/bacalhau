"""Example of submitting a docker job to the API."""

import pprint

from bacalhau_apiclient.models.deal import Deal
from bacalhau_apiclient.models.job_spec_docker import JobSpecDocker
from bacalhau_apiclient.models.job_spec_language import JobSpecLanguage
from bacalhau_apiclient.models.publisher_spec import PublisherSpec
from bacalhau_apiclient.models.spec import Spec
from bacalhau_apiclient.models.storage_spec import StorageSpec

from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id

data = dict(
    APIVersion="V1beta1",
    ClientID=get_client_id(),
    Spec=Spec(
        engine="Docker",
        publisher_spec=PublisherSpec(type="ipfs"),
        docker=JobSpecDocker(
            image="ubuntu",
            entrypoint=["echo", "Hello World!"],
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
        deal=Deal(concurrency=1),
        do_not_track=False,
    ),
)

pprint.pprint(submit(data))
