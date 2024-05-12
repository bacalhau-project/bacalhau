const { Payload } = require("../../models");
const { JobApi } = require("../api/job");
const {
	getClientPublicKey,
	signForClient,
	initializeSDK,
	getClientId,
} = require("./config");

const config = initializeSDK();
const jobApi = new JobApi(config);

async function submit(data = new Payload()) {
	try {
		data = data.toJson;
		const clientPublicKey = getClientPublicKey(),
			signature = signForClient(data);

		let body = {
			payload: data,
			signature: signature,
			client_public_key: clientPublicKey,
		};

		const response = await jobApi.submit(body);

		if (response.status === 200) {
			return response.data;
		} else {
			console.log("error: " + response.data);
			console.log("response.statusCode: " + response.statusCode);
			console.log("response.statusText: " + response.statusText);
		}
	} catch (error) {
		if (error.response) {
			console.log(error.response.data);
		} else {
			console.log(error);
		}
	}
}

async function list() {
	try {
		const clientId = getClientId();

		let body = {
			sort_reverse: false,
			sort_by: "created_at",
			return_all: false,
			max_jobs: 5,
			client_id: clientId,
		};

		const response = await jobApi.list(body);
		return response.data;
	} catch (error) {
		if (error.response) {
			console.log(error.response.data);
		} else {
			console.log(error);
		}
	}
}

async function results(jobId) {
	try {
		const clientId = getClientId();

		let body = {
			client_id: clientId,
			job_id: jobId,
		};

		const response = await jobApi.results(body);
		return response.data;
	} catch (error) {
		if (error.response) {
			console.log(error.response.data);
		} else {
			console.log(error);
		}
	}
}

async function states(jobId) {
	try {
		const clientId = getClientId();

		let body = {
			client_id: clientId,
			job_id: jobId,
		};

		const response = await jobApi.states(body);
		return response.data;
	} catch (error) {
		if (error.response) {
			console.log(error.response.data);
		} else {
			console.log(error);
		}
	}
}

async function events(jobId) {
	try {
		const clientId = getClientId();

		let body = {
			client_id: clientId,
			job_id: jobId,
		};

		const response = await jobApi.events(body);
		return response.data;
	} catch (error) {
		if (error.response) {
			console.log(error.response.data);
		} else {
			console.log(error);
		}
	}
}

module.exports = { submit, list, results, states, events };
