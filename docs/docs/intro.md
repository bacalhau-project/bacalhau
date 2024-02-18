---
title: About Bacalhau
sidebar_label: 'About Bacalhau'
sidebar_position: 1
slug: /
hide_title: true
---

<p align="center">
<img src="img/logo/Bacalhau-horizontal.svg" alt="Bacalhau" width="300" />
</p>

# What is Bacalhau?

Bacalhau is a platform for fast, cost efficient, and secure computation by running jobs where the data is generated and stored. With Bacalhau, you can streamline your existing workflows without the need of extensive rewriting by running arbitrary Docker containers and WebAssembly (wasm) images as tasks. This architecture is also referred to as **Compute Over Data** (or CoD). _[Bacalhau](https://translate.google.com/?sl=pt&tl=en&text=bacalhau&op=translate) was coined from the Portuguese word for salted Cod fish_.

Bacalhau seeks to transform data processing for large-scale datasets to improve cost and efficiency, and to open up data processing to larger audiences. Our goals is to create an open, collaborative compute ecosystem that enables unparalleled collaboration. We ([Expanso.io](https://expanso.io)) offer a demo network so you can try out jobs without even installing. Give it a shot!

## Why Bacalhau?

⚡️ Bacalhau simplifies the process of managing compute jobs by providing a **unified platform** for managing jobs across different regions, clouds, and edge devices.

🤝 Bacalhau provides **reliable and network-partition** resistant orchestration, ensuring that your jobs will complete even if there are network disruptions.

🚨 Bacalhau provides a **complete and permanent audit log** of exactly what happened, so you can be confident that your jobs are being executed securely.

🔐 You can run [private workloads](https://docs.bacalhau.org/next-steps/private-cluster) to **reduce the chance of leaking private information** or inadvertently sharing your data outside of your organization.

💸 Bacalhau **reduces ingress/egress costs** since jobs are processed closer to the source.

🤓 You can [mount your data anywhere](https://docs.bacalhau.org/#how-it-works) on your machine, and Bacalhau will be able to run against that data.

💥 You can integrate with services running on nodes to run a jobs, such as on [DuckDB](https://docs.bacalhau.org/examples/data-engineering/DuckDB/).

📚 Bacalhau operates at scale over parallel jobs. You can batch process petabytes (quadrillion bytes) of data.

## How it works

Bacalhau concists of a peer-to-peer network of nodes that enables decentralized communication between computers. The network consists of two types of nodes:

  **Requester Node:** responsible for handling user requests, discovering and ranking compute nodes, forwarding jobs to compute nodes, and monitoring the job lifecycle.

  **Compute Node:** responsible for executing jobs and producing results. Different compute nodes can be used for different types of jobs, depending on their capabilities and resources.

:::info
For a more detailed tutorial, check out our [Getting Started Tutorial](https://docs.bacalhau.org/getting-started/installation).
:::

The goal of the Bacalhau project is to make it easy to perform distributed computation next to where the data resides. In order to do this, first you need to ingest some data.

### Data ingestion
Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. Here are some options that can help you mount your data:

- [Copy data from a URL to public storage](https://docs.bacalhau.org/data-ingestion/from-url)
- [Pin Data to public storage](https://docs.bacalhau.org/data-ingestion/pin)
- [Copy Data from S3 Bucket to public storage](https://docs.bacalhau.org/data-ingestion/s3)

:::info
The options are not limited to the above mentioned. You can mount your data anywhere on your machine, and Bacalhau will be able to run against that data
:::

### Security in Bacalhau
All workloads run under restricted Docker or WASM permissions on the node. Additionally, you can use existing (locked down) binaries that are pre-installed through Pluggable Executors.

Best practices in [12-factor apps](https://12factor.net/) is to use environment variables to store sensitive data such as access tokens, API keys, or passwords. These variables can be accessed by Bacalhau at runtime and are not visible to anyone who has access to the code or the server.

Alternatively, you can pre-provision credentials to the nodes and access those on a node by node basis.

Finally, endpoints (such as vaults) can also be used to provide secure access to Bacalhau. This way, the client can authenticate with Bacalhau using the token without exposing their credentials.

### Workloads Bacalhau is best suited for
Bacalhau can be used for a variety of data processing workloads, including machine learning, data analytics, and scientific computing. It is well-suited for workloads that require processing large amounts of data in a distributed and parallelized manner.

### Use Cases
Once you have more than 10 devices generating or storing around 100GB of data, you're likely to face challenges with processing that data efficiently. Traditional computing approaches may struggle to handle such large volumes, and that's where distributed computing solutions like Bacalhau can be extremely useful. Bacalhau can be used in various industries, including security, web serving, financial services, IoT, Edge, Fog, and multi-cloud. Bacalhau shines when it comes to data-intensive applications like [data engineering](https://docs.bacalhau.org/examples/data-engineering/), [model training](https://docs.bacalhau.org/examples/model-training/), [model inference](https://docs.bacalhau.org/examples/model-inference/), [molecular dynamics](https://docs.bacalhau.org/examples/molecular-dynamics/), etc.

Here are some example tutorials on how you can process your data with Bacalhau:
- [Stable Diffusion AI](https://docs.bacalhau.org/examples/model-inference/stable-diffusion-gpu/)
- [Generate Realistic Images using StyleGAN3 and Bacalhau](https://docs.bacalhau.org/examples/model-inference/StyleGAN3/)
- [Object Detection with YOLOv5 on Bacalhau](https://docs.bacalhau.org/examples/model-inference/object-detection-yolo5/)
- [Running Genomics on Bacalhau](https://docs.bacalhau.org/examples/miscellaneous/Genomics/)
- [Training Pytorch Model with Bacalhau](https://docs.bacalhau.org/examples/model-training/Training-Pytorch-Model/)

:::info
For more tutorials, visit our [example page](https://docs.bacalhau.org/examples/)
:::

## Community

Bacalhau has a very friendly community and we are always happy to help you get started:

- [GitHub Discussions](https://github.com/bacalhau-project/bacalhau/discussions) – ask anything about the project, give feedback or answer questions that will help other users.
- [Join the Slack Community](https://bit.ly/bacalhau-project-slack) and go to **#bacalhau** channel – it is the easiest way engage with other members in the community and get help.
- [Contributing](https://docs.bacalhau.org/community/ways-to-contribute) – learn how to contribute to the Bacalhau project.

## Next Steps

👉 Continue with [Bacalhau Getting Started guide](https://docs.bacalhau.org/getting-started/installation) to learn how to install and run a job with the Bacalhau client.

👉 Or jump directly to try out the different [Examples](/docs/examples/index.md) that showcases Bacalhau abilities.
