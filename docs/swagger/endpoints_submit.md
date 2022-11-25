Description:

* `client_public_key`: The base64-encoded public key of the client.
* `signature`: A base64-encoded signature of the `data` attribute, signed by the client.
* `data`
    * `ClientID`: Request must specify a `ClientID`. To retrieve your `ClientID`, you can do the following: (1) submit a dummy job to Bacalhau (or use one you created before), (2) run `bacalhau describe <job-id>` and fetch the `ClientID` field.
    * `Job`: see example below.

Example request
```json
{
	"data": {
		"ClientID": "ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51",
		"Job": {
			"APIVersion": "V1beta1",
			"Spec": {
				"Engine": "Docker",
				"Verifier": "Noop",
				"Publisher": "Estuary",
				"Docker": {
					"Image": "ubuntu",
					"Entrypoint": [
						"date"
					]
				},
				"Timeout": 1800,
				"outputs": [
					{
						"StorageSource": "IPFS",
						"Name": "outputs",
						"path": "/outputs"
					}
				],
				"Sharding": {
					"BatchSize": 1,
					"GlobPatternBasePath": "/inputs"
				}
			},
			"Deal": {
				"Concurrency": 1
			}
		}
	},
	"signature": "...",
	"client_public_key": "..."
}
```