---
sidebar_label: 'Onboard WebAssembly Workload'
sidebar_position: 3
---
# Onboarding Your WebAssembly Workloads

Bacalhau supports running programs that are compiled to [WebAssembly (WASM)](https://webassembly.org/). With Bacalhau client, you can upload WASM programs, retrieve data from public storage, read and write data, receive program arguments and access environment variables.

## Prerequisites and Limitations

Bacalhau can run compiled WASM programs that expect the WebAssembly System Interface (WASI) Snapshot 1. Through this interface, WebAssembly programs can access data, environment variables and program arguments.

All ingress/egress networking is disabled – you won't be able to pull data/code/weights/etc from an external source. WASM jobs can say what data they need using URLs or CIDs (Content IDentifier) and can then access the data by reading from the filesystem.

There is no multi-threading as WASI does not expose any interface for it.

## Onboarding

### Step 1: Replace network operations with filesystem reads and writes

If your program would normally read and write to network endpoints, you'll need to replace this with filesystem operations.

For example, instead of making an HTTP request to `example.com`, instead read from the `/inputs` folder. You can then specify the URL to Bacalhau when you run the job using `--input http://example.com`.

You can write results to standard out or standard error pipes or to the filesystem into an output mount. For example, WASM jobs by default will have access to a folder at `/outputs` that will be persisted when the job ends.

:::tip
You can specify more or different output mounts using the `-o` flag.
:::

### Step 2: Configure your compiler to output WASI-compliant WebAssembly

You will need to compile your program to WebAssembly that expects WASI. Check the instructions for your compiler to see how to do this.

For example, Rust users can specify the `wasm32-wasi` target to `rustup` and `cargo` to get programs compiled for WASI WebAssembly. See [the Rust example](../examples/workload-onboarding/rust-wasm/index.md) for more information on this.

### Step 3: Upload the input data

Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. You can use either of these methods to upload your data:

- [Copy data from a URL to public storage](https://docs.bacalhau.org/examples/data-ingestion/from-url/)
- [Pin Data to public storage](https://docs.bacalhau.org/examples/data-ingestion/pinning/)
- [Copy Data from S3 Bucket to public storage](https://docs.bacalhau.org/examples/data-ingestion/s3-to-ipfs/)

:::info
You can mount your data anywhere on your machine, and Bacalhau will be able to run against that data
:::

### Step 4: Run your program

You can run a WebAssembly program on Bacalhau using the `bacalhau wasm run` command.

To run a locally compiled WASM program, specify it as an argument. For example, running `bacalhau wasm run main.wasm` will upload and execute the `main.wasm` program.

:::caution
The program you specify will be uploaded to a Bacalhau storage node and will be publicly available.
:::

Alternatively, you can specify a WASM program by 
using a CID, like `bacalhau wasm run Qmajb9T3jBdMSp7xh2JruNrqg3hniCnM6EUVsBocARPJRQ`.

Make sure to specify any input data using `--input` flag.


#### Program arguments

You can give the WASM program arguments by specifying them after the program path or CID. 

```shell
$ bacalhau wasm run echo.wasm hello world
```

:::tip

Write your program to use program arguments to specify input and output paths. This makes your program more flexible at handling different configurations of input and output volumes.

For example, instead of hard-coding your program to read from `/inputs/data.txt`, accept a program argument that should contain the path and then specify the path as an argument to `bacalhau wasm run`:

```shell
$ bacalhau wasm run prog.wasm /inputs/data.txt
```

Your language of choice should contain a standard way of reading program arguments that will work with WASI.
:::

#### Environment variables

You can also specify environment variables using the `-e` flag.

```shell
$ bacalhau wasm run prog.wasm -e HELLO=world
```

## Examples

See [the Rust example](../examples/workload-onboarding/rust-wasm/index.md) for a workload that leverages WebAssembly support.

## Support


If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/archives/C02RLM3JHUY)(#bacalhau channel)
