const { default: axios } = require("axios");

class HeakthApi {
	constructor(config) {
		this.config = config;
	}

	async apiServerdebug() {
		return axios.get(this.config.base_url + "/debug");
	}

	async list(body) {
		if (!body)
			throw Error("Missing the required parameter `body` when calling `list`");
		return axios.post(this.config.base_url + "/requester/list", body);
	}

	async results(body) {
		if (!body)
			throw Error(
				"Missing the required parameter `body` when calling `results`"
			);
		return axios.post(this.config.base_url + "/requester/results", body);
	}

	async states(body) {
		if (!body)
			throw Error(
				"Missing the required parameter `body` when calling `states`"
			);
		return axios.post(this.config.base_url + "/requester/states", body);
	}

	async events(body) {
		if (!body)
			throw Error(
				"Missing the required parameter `body` when calling `events`"
			);
		return axios.post(this.config.base_url + "/requester/events", body);
	}
}

module.exports = { HeakthApi };
