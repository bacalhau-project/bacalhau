const { Spec } = require("./spec");

class Payload {
	constructor({ APIVersion = "V1beta1", ClientID, spec = new Spec() } = {}) {
		this.APIVersion = APIVersion;
		this.ClientID = ClientID;
		this.Spec = spec;
	}

	get toJson() {
		return {
			APIVersion: this.APIVersion,
			ClientID: this.ClientID,
			Spec: this.Spec.toJson,
		};
	}
}

module.exports = { Payload };
