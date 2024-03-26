class JobSpecDocker {
	constructor({
		entrypoint = [],
		environment_variables = [],
		image,
		working_directory,
	} = {}) {
		this.entrypoint = entrypoint;
		this.environment_variables = environment_variables;
		this.image = image;
		this.working_directory = working_directory;
	}

	get toJson() {
		return {
			entrypoint: this.entrypoint,
			environment_variables: this.environment_variables,
			image: this.image,
			working_directory: this.working_directory,
		};
	}
}

module.exports = { JobSpecDocker };
