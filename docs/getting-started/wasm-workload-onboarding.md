---
sidebar_label: 'Onboard WebAssembly Workload'
sidebar_position: 3
---
# Onboarding Your WebAssembly Workloads

Bacalhau supports running programs that are compiled to [WebAssembly (WASM)](https://webassembly.org/). 

WASM programs can be uploaded using the Bacalhau client or can be retrieved from IPFS. They can read and write data, receive program arguments and access environment variables.

## Prerequisites and Limitations

Bacalhau can run compiled WASM programs that expect the WebAssembly System Interface (WASI) Snapshot 1. Through this interface, WebAssembly programs can access data, environment variables and program arguments.

All ingress/egress networking is disabled – you won't be able to pull data/code/weights/etc from an external source. Instead, data input and output is implemented using [Bacalhau's input/output volumes](../about-bacalhau/architecture.md#input--output-volumes). WASM jobs can say what data they need using URLs or IPFS CIDs and can then access the data by reading from the filesystem.

There is no multi-threading as WASI does not expose any interface for it.

## Onboarding

### Step 1: Replace network operations with filesystem reads and writes

If your program would normally read and write to network endpoints, you'll need to replace this with filesystem operations.

For example, instead of making an HTTP request to `example.com`, instead read from the `/inputs` folder. You can then specify the URL to Bacalhau when you run the job using `--input-urls example.com`.

You can write results to standard out or standard error pipes or to the filesystem into an output mount. For example, WASM jobs by default will have access to a folder at `/outputs` that will be persisted when the job ends.

:::tip
You can specify more or different output mounts using the `-o` flag.
:::

### Step 2: Configure your compiler to output WASI-compliant WebAssembly

You will need to compile your program to WebAssembly that expects WASI. Check the instructions for your compiler to see how to do this.

For example, Rust users can specify the `wasm32-wasi` target to `rustup` and `cargo` to get programs compiled for WASI WebAssembly. See [the Rust example](../examples/workload-onboarding/rust-wasm/index.md) for more information on this.

### Step 3: Upload the input data to IPFS (optional)

We recommend uploading your data to IPFS for persistent storage, because:

* Bacalhau is designed to perform the computation next to the data
* Distributing data across the solar system with IPFS distributes the Bacalhau computation
* Distributing computation improves performance by scaling, and improves resiliency via redundancy
* Using IPFS CIDs as inputs enables repeatable and cacheable execution

:::tip
The following guides explain how to store data on the IPFS network.

- Leverage an IPFS “pinning service” such as:
  - [Web3.Storage](https://web3.storage/account/)
  - [Estuary](https://estuary.tech/sign-in)
  - [Manually pin your files to IPFS](https://docs.ipfs.io/how-to/pin-files/) with your own IPFS server.
- If uploading a folder of input files, consider [uploading with this script](https://web3.storage/docs/#create-the-upload-script). However, please note that any content uploaded to Web3.storage is [also wrapped in a parent directory](https://web3.storage/docs/how-tos/store/#directory-wrapping). You will need to take care to reference the inner directory CID in your bacalhau command.
:::

### Step 4: Run your program

You can run a WebAssembly program on Bacalhau using the `bacalhau wasm run` command.

To run a locally compiled WASM program, specify it as an argument. For example, running `bacalhau wasm run main.wasm` will upload and execute the `main.wasm` program.

:::caution
The program you specify will be uploaded to a Bacalhau IPFS node and will be publicly available.
:::

Alternatively, you can specify a WASM program already on IPFS by using a CID, like `bacalhau wasm run Qmajb9T3jBdMSp7xh2JruNrqg3hniCnM6EUVsBocARPJRQ`.

Make sure to specify any input data using the `--input-volumes` or `--input-urls` flags.

:::tip
The `--input-urls` flag can only be used once, and will make the contents of the URL available at the `/inputs` directory.
:::

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

Please reach out to the [Bacalhau team via Slack](https://filecoinproject.slack.com/archives/C02RLM3JHUY) if you would like help pinning data to IPFS for your job or for any issues you encounter.
