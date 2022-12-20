import pem
import base64
import pprint

from Crypto.PublicKey import RSA
from Crypto.Hash import SHA256, SHA3_256
from Crypto.Signature import pkcs1_15, pss
from cryptography import x509

import json

from bacalhau_apiclient import ApiClient

key_file = "/Users/enricorotundo/.bacalhau/user_id.pem"
with open(key_file, 'rb') as f:
   certs = pem.parse(f.read())
private_key = RSA.import_key(certs[0].as_bytes())


# SignForClient signs a message with the user's private ID key.
def sign_payload(payload, client: ApiClient):    
    job_create_payload_json = json.dumps(client.sanitize_for_serialization(payload), indent=None, separators=(',', ':'),sort_keys=True).encode()
    # DATA='{"job_create_payload":{"ClientID":"bae9c3b2adfa04cc647a2457e8c0c605cef8ed93bdea5ac5f19f94219f722dfe","APIVersion":"V1beta1","Spec":{"Engine":"Docker","Verifier":"Noop","Publisher":"Estuary","Docker":{"Image":"ubuntu","Entrypoint":["date"]},"Language":{"JobContext":{}},"Wasm":{},"Resources":{"GPU":""},"Timeout":1800,"outputs":[{"StorageSource":"IPFS","Name":"outputs","path":"/outputs"}],"Sharding":{"BatchSize":1,"GlobPatternBasePath":"/inputs"},"ExecutionPlan":{},"Deal":{"Concurrency":1}}},"signature":"GPkRGWQMWXjhm6Xh+4EhzYuA8RCyB2/TLNEj33M7vtzv6ESKRVnLLjuRGyDC5DdSgvM3q7J/SDqfPG5GViZ6Mw0Zem33p1ZtwwPQrC/yrE/p0FwiReNWI727Ze9BLkwUU1bTlNp4rAdDc63o7I8kbZs4ibgnl/r7aG9XQQA6xXu2TBZw4m1hp0NgNTK9Dm6y+lI1QIon1yc7C9Qe7ZHWBQG1qdvSosoeVrawuOXI5zhATS/gSVwveo0KNLXtZ+Hc/CHSglUoii9IHvk6ECCIpSR8YpmgnlC6zpbOBSiIsIspVHelvhDVDS/RkNGBmOXA+h/ECQ07rNYpj1oqLg7uqw==","client_public_key":"MIIBCgKCAQEArc7/yPio/awcazNgYdLoCNYsuowTJz7QCs2JhrbED3Kv2swNlxa0YqZfvXNfTjq2povLFgxgOG1nUkKeTgLjB5871in3XiYm9yizQonPsXr4UMpj7wCNo+QpQsI4JjyO/yNW/9l46+ZcqEn1WvQGWpDER703U59vberasW+Yu7Y75dyEU4Sn07XRyPLfCwoR7JUVhChvvjw9xnttwrigj+m3wq65h2KqLUQpGQAp3Ulnces++uuaV9uZS430MU4D+ooWJSWLsxzhEcpHB/pHWVxf8wIF3ozmNq+WR6YTmKt0vY+548EdVrgG/YptTkUu8QTjRtzpv4fATs/z1W7GVQIDAQAB"}'
    # job_create_payload_json = DATA.encode()
    # job_create_payload_json = b'{"ClientID":"bae9c3b2adfa04cc647a2457e8c0c605cef8ed93bdea5ac5f19f94219f722dfe","APIVersion":"V1beta1","Spec":{"Engine":"Docker","Verifier":"Noop","Publisher":"Estuary","Docker":{"Image":"ubuntu","Entrypoint":["date"]},"Language":{"JobContext":{}},"Wasm":{},"Resources":{"GPU":""},"Timeout":1800,"outputs":[{"StorageSource":"IPFS","Name":"outputs","path":"/outputs"}],"Sharding":{"BatchSize":1,"GlobPatternBasePath":"/inputs"},"ExecutionPlan":{},"Deal":{"Concurrency":1}}}'

    print()
    print("job_create_payload_json")
    print(job_create_payload_json)
    print("job_create_payload_json hex")
    print(job_create_payload_json.hex())
    print()
    print()
    
    
    # https://pycryptodome.readthedocs.io/en/latest/src/signature/signature.html#signing-a-message
    signer = pkcs1_15.new(private_key)
    hash_obj = SHA256.new()
    hash_obj.update(job_create_payload_json)

    print()
    print("hash_obj")
    print(hash_obj.hexdigest())
    print()
    print()
    
    signed_payload = signer.sign(hash_obj)

    # 344 chars
    signature = base64.b64encode(signed_payload).decode()
    print()
    print("signature")
    print(signature)
    print()
    print()

    return signature




def get_publickey():
    public_key = private_key.public_key()


    client_public_key = public_key.export_key('DER')
    print("base64.encodebytes(client_public_key)")
    print(base64.b64encode(client_public_key))
    

    return base64.b64encode(client_public_key).decode()[32:]
