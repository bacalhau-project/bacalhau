"""Main module."""

import bacalhau_apiclient
from bacalhau_apiclient.api import job_api
from bacalhau_apiclient.api import utils_api
from bacalhau_apiclient.models.version_request import VersionRequest
from bacalhau_apiclient.models.list_request import ListRequest
from bacalhau_apiclient.models.submit_request import SubmitRequest
from bacalhau_apiclient.models.job_create_payload import JobCreatePayload
from bacalhau_apiclient.models.spec import Spec
from bacalhau_apiclient.models.job_spec_docker import JobSpecDocker

from bacalhau_apiclient.models.job_spec_language import JobSpecLanguage
from bacalhau_apiclient.models.storage_spec import StorageSpec
from bacalhau_apiclient.models.spec_sharding import SpecSharding
from bacalhau_apiclient.models.job_sharding_config import JobShardingConfig
from bacalhau_apiclient.models.spec_deal import SpecDeal
from bacalhau_apiclient.models.deal import Deal

import json
import pprint

from bacalhau_sdk.utils import sign_payload, get_publickey


conf = bacalhau_apiclient.Configuration()
conf.host = "http://0.0.0.0:20002"
client_id = "bae9c3b2adfa04cc647a2457e8c0c605cef8ed93bdea5ac5f19f94219f722dfe"
client = job_api.ApiClient(conf)
jobapi_instance = job_api.JobApi(client)



job_create_payload=JobCreatePayload(
        api_version='V1beta1',
        client_id=client_id, 
        spec=Spec(
            engine="Docker",
            verifier="Noop",
            publisher= "Estuary",
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
            execution_plan=None,
            deal=Deal(concurrency=1)
        ),
)

# job_create_payload.
signature = sign_payload(job_create_payload, client)
client_public_key = get_publickey()

# DATA='{"job_create_payload":{"ClientID":"bae9c3b2adfa04cc647a2457e8c0c605cef8ed93bdea5ac5f19f94219f722dfe","APIVersion":"V1beta1","Spec":{"Engine":"Docker","Verifier":"Noop","Publisher":"Estuary","Docker":{"Image":"ubuntu","Entrypoint":["date"]},"Language":{"JobContext":{}},"Wasm":{},"Resources":{"GPU":""},"Timeout":1800,"outputs":[{"StorageSource":"IPFS","Name":"outputs","path":"/outputs"}],"Sharding":{"BatchSize":1,"GlobPatternBasePath":"/inputs"},"ExecutionPlan":{},"Deal":{"Concurrency":1}}},"signature":"GPkRGWQMWXjhm6Xh+4EhzYuA8RCyB2/TLNEj33M7vtzv6ESKRVnLLjuRGyDC5DdSgvM3q7J/SDqfPG5GViZ6Mw0Zem33p1ZtwwPQrC/yrE/p0FwiReNWI727Ze9BLkwUU1bTlNp4rAdDc63o7I8kbZs4ibgnl/r7aG9XQQA6xXu2TBZw4m1hp0NgNTK9Dm6y+lI1QIon1yc7C9Qe7ZHWBQG1qdvSosoeVrawuOXI5zhATS/gSVwveo0KNLXtZ+Hc/CHSglUoii9IHvk6ECCIpSR8YpmgnlC6zpbOBSiIsIspVHelvhDVDS/RkNGBmOXA+h/ECQ07rNYpj1oqLg7uqw==","client_public_key":"MIIBCgKCAQEArc7/yPio/awcazNgYdLoCNYsuowTJz7QCs2JhrbED3Kv2swNlxa0YqZfvXNfTjq2povLFgxgOG1nUkKeTgLjB5871in3XiYm9yizQonPsXr4UMpj7wCNo+QpQsI4JjyO/yNW/9l46+ZcqEn1WvQGWpDER703U59vberasW+Yu7Y75dyEU4Sn07XRyPLfCwoR7JUVhChvvjw9xnttwrigj+m3wq65h2KqLUQpGQAp3Ulnces++uuaV9uZS430MU4D+ooWJSWLsxzhEcpHB/pHWVxf8wIF3ozmNq+WR6YTmKt0vY+548EdVrgG/YptTkUu8QTjRtzpv4fATs/z1W7GVQIDAQAB"}'

submit_req = SubmitRequest(
    client_public_key=client_public_key, 
    job_create_payload=job_create_payload,
    signature=signature
    # signature='GPkRGWQMWXjhm6Xh+4EhzYuA8RCyB2/TLNEj33M7vtzv6ESKRVnLLjuRGyDC5DdSgvM3q7J/SDqfPG5GViZ6Mw0Zem33p1ZtwwPQrC/yrE/p0FwiReNWI727Ze9BLkwUU1bTlNp4rAdDc63o7I8kbZs4ibgnl/r7aG9XQQA6xXu2TBZw4m1hp0NgNTK9Dm6y+lI1QIon1yc7C9Qe7ZHWBQG1qdvSosoeVrawuOXI5zhATS/gSVwveo0KNLXtZ+Hc/CHSglUoii9IHvk6ECCIpSR8YpmgnlC6zpbOBSiIsIspVHelvhDVDS/RkNGBmOXA+h/ECQ07rNYpj1oqLg7uqw=='
)
pprint.pprint(submit_req.to_dict())

print(jobapi_instance.submit(submit_req))










# utilsapi_instance = utils_api.UtilsApi(client)
# print(utilsapi_instance.version(VersionRequest(client_id="test")))


# list_req = ListRequest(
#     client_id=client_id, 
#     exclude_tags=[], 
#     id=None, 
#     include_tags=[], 
#     max_jobs=5,
#     return_all=False, 
#     sort_by="created_at", 
#     sort_reverse=True
# )
# list_res = jobapi_instance.list(list_req)
# print(len(list_res.jobs))