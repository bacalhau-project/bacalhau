class JobSpecLanguage {
	constructor({
		command,
		deterministic_execution,
		job_context,
		language,
		language_version,
		program_path,
		requirements_path,
	} = {}) {
		this.command = command;
		this.deterministic_execution = deterministic_execution;
		this.job_context = job_context;
		this.language = language;
		this.language_version = language_version;
		this.program_path = program_path;
		this.requirements_path = requirements_path;
	}

	get toJson() {
		return {
			command: this.command,
			deterministic_execution: this.deterministic_execution,
			job_context: this.job_context,
			language: this.language,
			language_version: this.language_version,
			program_path: this.program_path,
			requirements_path: this.requirements_path,
		};
	}
}

module.exports = { JobSpecLanguage };
