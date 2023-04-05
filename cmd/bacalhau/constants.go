package bacalhau

const inputUsageMsg = `Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
		Examples:
		# Mount IPFS CID to /inputs directory
		-i src=ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

		# Mount S3 object to a specific path
		-i src=s3://bucket/key,dst=/my/input/path

		# Mount S3 object with specific endpoint and region
		-i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=http://s3.example.com,opt=region=us-east-1
`
