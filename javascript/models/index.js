const { Payload } = require("./payload");
const { Spec } = require("./spec");
const { PublisherSpec } = require("./publisherSpec");
const { JobSpecLanguage } = require("./jobSpecLanguage");
const { JobSpecDocker } = require("./jobSpecDocker");
const { StorageSpec } = require("./storageSpec");
const { Deal } = require("./deal");

module.exports = {
	Spec,
	Payload,
	PublisherSpec,
	JobSpecLanguage,
	JobSpecDocker,
	StorageSpec,
	Deal,
};
