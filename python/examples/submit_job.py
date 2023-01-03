"""Example of submitting a docker job to the API.

For production network:
```
python examples/submit_job.py
```

For devstack:
```
BACALHAU_API_HOST=0.0.0.0 BACALHAU_API_PORT=20002 python examples/submit_job.py
```
"""

import logging
import pprint
import json

from bacalhau_apiclient.api import job_api
from bacalhau_apiclient.models.deal import Deal
from bacalhau_apiclient.models.job_execution_plan import JobExecutionPlan
from bacalhau_apiclient.models.job_sharding_config import JobShardingConfig
from bacalhau_apiclient.models.job_spec_docker import JobSpecDocker
from bacalhau_apiclient.models.job_spec_language import JobSpecLanguage
from bacalhau_apiclient.models.spec import Spec
from bacalhau_apiclient.models.storage_spec import StorageSpec
from bacalhau_apiclient.models.submit_request import SubmitRequest

from bacalhau_sdk.config import init_config, get_client_id, sign_for_client, get_client_public_key

log = logging.getLogger(__name__)
log.setLevel(logging.DEBUG)

conf = init_config()

client = job_api.ApiClient(conf)

jobapi_instance = job_api.JobApi(client)

data = dict(
    APIVersion='V1beta1',
    ClientID=get_client_id(),
    Spec=Spec(
        engine="Docker",
        verifier="Noop",
        publisher="Estuary",
        docker=JobSpecDocker(
            image="ubuntu",
            entrypoint=["echo", "123"],
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


sanitized_data = client.sanitize_for_serialization(data)
json_data = json.dumps(sanitized_data, indent = None, separators=(', ', ': '))
json_bytes = json_data.encode('utf-8')
print(json_bytes)


signature = sign_for_client(json_bytes)


client_public_key = get_client_public_key()
submit_req = SubmitRequest(
    client_public_key=client_public_key,
    job_create_payload=sanitized_data,
    signature=signature
)
print(submit_req)
print()


print(jobapi_instance.submit(submit_req))