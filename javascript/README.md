
# Bacalhau JS🟨


```js
const { submit } = require('@daggle/bacalhau-js')
const { Payload } =  require("@daggle/bacalhau-js/models");

let data = new Payload({
	ClientID: ...,
	spec: ...
})

submit(data)
```
[Check examples↗️](https://github.com/dagglexyz/bacalhau-js-example)

## Installation

This is a [Node.js](https://nodejs.org/en/) module available through the [npm registry](https://www.npmjs.com/).

Before installing, [download and install Node.js](https://nodejs.org/en/download/). Node.js 0.10 or higher is required.

If this is a brand new project, make sure to create a `package.json` first with the [`npm init` command](https://docs.npmjs.com/creating-a-package-json-file).

Installation is done using the [`npm install` command](https://docs.npmjs.com/getting-started/installing-npm-packages-locally):

```console
$ npm install @daggle/bacalhau-js
```

## Example
```js
const { submit } =  require("@daggle/bacalhau-js");

const {
Payload,
Spec,
PublisherSpec,
StorageSpec,
Deal,
JobSpecDocker,
} =  require("@daggle/bacalhau-js/models");

async function submitJob() {
	let data = new Payload({
	ClientID: getClientId(),
	spec: new Spec({
	deal: new Deal(),
	docker: new JobSpecDocker({
			image: "ubuntu",
			entrypoint: ["echo", "Hello World!"],
		}),
		engine: "Docker",
		publisher_spec: new PublisherSpec({ type:  "Estuary" }),
		timeout: 1800,
		verifier: "Noop",
		outputs: [
			new  StorageSpec({
				path: "/outputs",
				storage_source: "IPFS",
				name: "outputs",
			}),
		],
		}),
	});

	const  response = await submit(data);
	console.log(response);
}
submitJob()
```
[Check complete example↗️](https://github.com/dagglexyz/bacalhau-js-example)

## Local Testing

Follow the below steps to run it locally.

### Clone repo⚙️

1. Clone Repo.
	```console
	$ git clone https://github.com/dagglexyz/bacalhau-js bacalhau-js
	```
	**Change Host**
	Initialize the SDK by passing in the host URL as parameter.
	***./src/sdk/config.js***
	```js
	...
	} = require("./config");
	// Change host here
	const  config = initializeSDK("http://127.0.0.1:1234");
	const  jobApi = new  JobApi(config);
	
	async  function  submit(data  =  new  Payload()) {
	...
	```
2. Create a simple node project
	```console
	$ mkdir example-project
	$ cd example-project
	$ npm init
	```	 
3. Install package
	```console
	$ npm install ../bacalhau-js
	```
4. Testing, create a file at root, *index.js*.
	```js
	const { list} =  require("@daggle/bacalhau-js");

	async function listJobs() {
		const response = await list();
		console.log(response.jobs);
	}
	listJobs()
	```

## Note⚠️
 This is not a  production ready SDK, this repo will be moved to [Bacalhau](https://github.com/bacalhau-project) organization's GitHub and  maintained by [Bacalhau](https://github.com/bacalhau-project) in the future.
