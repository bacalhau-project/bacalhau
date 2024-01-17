---
sidebar_position: 4
sidebar_label: "Internet Access"
---

# Internet access

By default, Bacalhau jobs do not have any access to the internet. This is to keep both compute providers and users safe from malicious activities.

However, you can access your data before or during the execution of a job:
- Using Data Volumes to download the input data and upload the results
- Using `--network` flag to allow job to access the internet 
:::info
For more details about flags and recent updates see the [CLI Guide](../../dev/cli-reference/all-flags.md) or execute `bacalhau help`. Also feel free to contact us on [Slack](https://bacalhauproject.slack.com).
:::

## Using Data Volumes

If you need to process data located on a certain network resource and/or save the result of the job execution to such resource - you will need to specify the internet locations to download data from and write results to when creating the job. Both [Docker](../../getting-started/docker-workload-onboarding.md) and [WebAssembly](../../getting-started/wasm-workload-onboarding.md) jobs support these features.

When submitting a Bacalhau job, you can specify the CID (Content IDentifier) or HTTP(S) URL to download data from. The data will be retrieved before the job starts and made available to the job as a directory on the filesystem. When running Bacalhau jobs, you can specify as many CIDs or URLs as needed using `--input` which is accepted by both `bacalhau docker run` and `bacalhau wasm run`. See [command line flags](../../dev/cli-reference/all-flags.md) for more information. Make sure the nodes on the network have enough resources to download and process specified data.

You can write back results from your Bacalhau jobs to your public storage location. By default, jobs will write results to the storage provider using the `--publisher` command line flag. See [command line flags](../../dev/cli-reference/all-flags.md) on how to configure this.


## Specifying Jobs to Access the Internet

For some workloads, the required data is computed as part of the job if the purpose of the job is to process web results. In these cases, networking may be possible during job execution. To run Docker jobs on Bacalhau with internet access, you'll need to use a `--netwwork` flag and specify one of the following options:

* **full**: unfiltered networking for any protocol `--network=full`
* **http**: HTTP(S)-only networking to a specified list of domains `--network=http`
* **none**: no networking at all, the default `--network=none`

:::tip
Specifying `none` will still allow Bacalhau to download and upload data before and after the job.
:::

Jobs using `http` must specify the domains they want to access when the job is submitted. When the job runs, only HTTP(S) requests to those domains will be possible and data transfer will be rate limited to 10Mbit/sec in either direction to prevent ddos.

Jobs will be provided with [`http_proxy` and `https_proxy` environment variables](https://about.gitlab.com/blog/2021/01/27/we-need-to-talk-no-proxy/) which contain a TCP address of an HTTP proxy to connect through. Most tools and libraries will use these environment variables by default. If not, they must be used by user code to configure HTTP proxy usage.

The required networking can be specified using the `--network` flag. For `http` networking, the required domains can be specified using the `--domain` flag, multiple times for as many domains as required. Specifying a domain starting with a `.` means that all sub-domains will be included. For example, specifying `.example.com` will cover `some.thing.example.com` as well as `example.com`.

:::caution
Bacalhau jobs are explicitly prevented from starting other Bacalhau jobs, even if a Bacalhau requester node is specified on the HTTP allowlist.
:::

## Support for networked jobs on the public network

Bacalhau supports *describing* jobs that can access the internet during job execution. The ability of a public network to run jobs, that require internet access depends on what compute nodes are currently part of the network.

Compute nodes that join the Bacalhau network do not accept networked jobs by default (i.e. they only accept jobs that specify `--network=none`, which is also the default).

The public compute nodes provided by the Bacalhau network will accept jobs that require HTTP networking as long as the domains are from [this allowlist](https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/scripts/http-domain-allowlist.txt).

If you need to access a domain that isn't on the allowlist, you can make a request to the [Bacalhau Project team](https://github.com/bacalhau-project/bacalhau/discussions) to include your required domains. You can also set up your own compute node that implements the allowlist you need.
