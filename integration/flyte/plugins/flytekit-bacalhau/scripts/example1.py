import pprint

from bacalhau_sdk.api import submit
from bacalhau_sdk.config import get_client_id
from bacalhau_apiclient.models.spec import Spec


data = dict(
    APIVersion="V1beta1",
    ClientID=get_client_id(),
    Spec=Spec(
        engine="Docker",
        verifier="Noop",
        publisher_spec={"type": "Estuary"},
        docker={
            "image": "ubuntu",
            "entrypoint": ["echo", "Hello World!"],
        },
        language={"job_context": None},
        wasm=None,
        resources=None,
        timeout=1800,
        outputs=[
            {
                "storage_source": "IPFS",
                "name": "outputs",
                "path": "/outputs",
            },
        ],
        deal={"concurrency": 1},
        do_not_track=False,
    ),
)

pprint.pprint(submit(data))
