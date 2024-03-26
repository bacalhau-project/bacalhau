class StorageSpec {
	constructor({
		cid,
		metadata = {},
		name,
		repo,
		s3,
		source_path,
		StorageSource,
		url,
		path,
	} = {}) {
		this.cid = cid;
		this.metadata = metadata;
		this.name = name;
		this.repo = repo;
		this.s3 = s3;
		this.source_path = source_path;
		this.StorageSource = StorageSource;
		this.url = url;
		this.path = path;
	}

	get toJson() {
		return {
			cid: this.cid,
			metadata: this.metadata,
			name: this.name,
			repo: this.repo,
			s3: this.s3,
			source_path: this.source_path,
			StorageSource: this.StorageSource,
			url: this.url,
			path: this.path,
		};
	}
}

module.exports = { StorageSpec };
