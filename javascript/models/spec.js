const { Deal } = require("./deal");
const { JobSpecDocker } = require("./jobSpecDocker");
const { JobSpecLanguage } = require("./jobSpecLanguage");
const { PublisherSpec } = require("./publisherSpec");

class Spec {
	constructor({
		annotations = [],
		deal = new Deal(),
		do_not_track = false,
		docker = new JobSpecDocker(),
		engine = "Docker",
		language = new JobSpecLanguage(),
		network,
		node_selectors,
		publisher,
		publisher_spec = new PublisherSpec(),
		resources,
		timeout,
		verifier,
		wasm,
		inputs = [],
		outputs = [],
	} = {}) {
		this.annotations = annotations;
		this.deal = deal;
		this.do_not_track = do_not_track;
		this.outputs = outputs;
		this.docker = docker;
		this.engine = engine;
		this.language = language;
		this.network = network;
		this.node_selectors = node_selectors;
		this.publisher = publisher;
		this.PublisherSpec = publisher_spec;
		this.resources = resources;
		this.timeout = timeout;
		this.verifier = verifier;
		this.wasm = wasm;
		this.inputs = inputs;
	}

	get toJson() {
		this.outputs = this.outputs.map((o) => o.toJson);
		this.inputs = this.inputs.map((i) => i.toJson);

		return {
			annotations: this.annotations,
			deal: this.deal.toJson,
			do_not_track: this.do_not_track,
			outputs: this.outputs,
			docker: this.docker.toJson,
			engine: this.engine,
			language: this.language.toJson,
			network: this.network,
			node_selectors: this.node_selectors,
			publisher: this.publisher,
			PublisherSpec: this.PublisherSpec.toJson,
			resources: this.resources,
			timeout: this.timeout,
			verifier: this.verifier,
			wasm: this.wasm,
			inputs: this.inputs,
		};
	}
}

module.exports = { Spec };
