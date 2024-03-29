import "strings"

#Job: {
    Name:       string
    Namespace:  string
    Type:       "daemon" | "ops" | "batch" | "service"
    Priority:   int & >=0 | *1
    Count:      int & >0
    Constraints?: [...#LabelSelectorRequirement]
    Labels?: [string]: string
    Tasks: [...#Task]
}

#LabelSelectorRequirement: {
    Key: string
    Operator: "!" | "=" | "==" | "in" | "!=" | "notin" | "exists" | "gt" | "lt"
    Values: [...string]
}

#Task: {
    Name: string
    Engine: #EngineSpecConfig

    Publisher?: #PublisherSpecConfig
    Env?: [string]: string
    InputSources?: [...#InputSource]
    ResultPaths?: [...#ResultPath]
    Resources?: #ResourcesConfig
    Network?: #NetworkConfig
    Timeouts?: #TimeoutConfig
}

#ResultPath: {
    Name: string
    Path: string
}

#ResourcesConfig: {
    CPU?: number
    Memory?: string
    Disk?: string
    GPU?: string
}

#NetworkConfig: {
    Type: #Network

    Domains?: [...string]
}

#Network: "none" | "full" | "http"

#TimeoutConfig: {
    ExecutionTimeout?: int64
}

#EngineSpecConfig: {
    Type: string
    Params: #EngineParams[strings.ToLower(Type)]
}

#EngineParams: {
    // engines
    "docker": #DockerEngineSpec
    "wasm":   #WasmEngineSpec
}

#DockerEngineSpec: {
    Image:                string

    Entrypoint?:           [...string]
    Parameters?:           [...string]
    EnvironmentVariables?: [...string]
    WorkingDirectory?:     string
}

#WasmEngineSpec: {
    EntryModule: #InputSource

    EntryPoint?: string
    Parameters?: [...string]
    EnvironmentVariables?: [string]: string
    ImportModules?: [#InputSource]
}


#PublisherSpecConfig: {
    Type: string
    Params: #PublisherParams[strings.ToLower(Type)]
}

#PublisherParams: {
    "s3": #S3PublisherSpec
    "ipfs": #IPFSPublisherSpec
    "local": #LocalPublisherSpec
}

#S3PublisherSpec: {
    Bucket:   string
    Key:      string
    // TODO make this required
    Endpoint?: string

    Region?:   string
}

#IPFSPublisherSpec: {}
#LocalPublisherSpec: {}

#StorageSpecConfig: {
    Type: string
    Params: #StorageParams[strings.ToCamel(Type)]
}

#StorageParams: {
    "ipfs": #IPFSStorageSpec
    "urlDownload": #URLStorageSpec
    "inline": #URLStorageSpec
    "localDirectory": #LocalDirectoryStorageSpec
    "s3": #S3StorageSpec

    // TODO deprecate
    "repoclone": #RepoStorageSpec
    "repoclonelfs": #RepoStorageSpec
}

#URLStorageSpec: {
    URL: string
}

#IPFSStorageSpec: {
    CID: string
}

#RepoStorageSpec: {
    Repo: string
}

#LocalDirectoryStorageSpec: {
    SourcePath: string
    ReadWrite: bool | *false
}

#S3StorageSpec: {
    Bucket: string
    Key: string
    // TODO make this required
    Endpoint?: string

    Filter?: string
    Region?: string
    VersionID?: string
    ChecksumSHA256?: string
}

#InputSource: {
    Source: #StorageSpecConfig
    Target: string

    Alias?: string
}
