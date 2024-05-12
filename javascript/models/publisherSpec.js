class PublisherSpec {
	constructor({ params, type } = {}) {
		this.params = params;
		this.type = type;
	}

	get toJson() {
		return {
			params: this.params,
			Type: this.type,
		};
	}
}

module.exports = { PublisherSpec };
