---
sidebar_label: 'Home'
sidebar_position: 1
slug: /
hide_title: true
---

<p align="center">
<img src="img/bacalhau-horizontal.jpg" alt="Bacalhau Logo" width="300" />
</p>

## Overview

Bacalhau is a platform for fast, cost efficient, and secure computation by running jobs where the data is generated and stored. With Bacalhau, you can streamline your existing workflows without the need of extensive rewriting by running  arbitrary Docker containers and WebAssembly (wasm) images as tasks. This architecture is also referred to as **Compute Over Data** (or CoD). _[Bacalhau](https://translate.google.com/?sl=pt&tl=en&text=bacalhau&op=translate) was coined from the Portuguese word for salted Cod fish_. 

Bacalhau seeks to transform data processing for large-scale datasets to improve cost and efficiency, and to open up data processing to larger audiences. Our goals with the project center around creating an open, collaborative Compute ecosystem. We believe that the same benefits of open collaboration on datasets should be available to generic storage compute tasks.  **At the moment we are free volunteer network, enjoy;)**

## Why Bacalhau?

⚡️ Jobs in Bacalhau are processed where the data was created and all **jobs are parallel by default**.

🔐 You can run [private workloads](https://docs.bacalhau.org/next-steps/private-cluster) to **reduce the chance of leaking private information** or inadvertently sharing your data outside of your organization.

💸 Bacalhau **eliminates ingress/egress costs** since jobs are processed closer to the source. 

🤓  You can [mount your data anywhere](https://docs.bacalhau.org/#how-it-works) on your machine, and Bacalhau will be able to run against that data 

💥 You can integrate with Bacalhau and run a job on a [database](https://docs.bacalhau.org/examples/data-engineering/DuckDB/)

📚 Bacalhau operates on a network of **open compute resources** made available to serve any data processing workload. With Bacalhau you can batch process petabytes (quadrillion bytes) of data.


## Fast Track ⏱️

Understand Bacalhau in 1 minute 

Go to the folder directory that you want to store your job results

Install the bacalhau client

```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

Submit a "Hello World" job

```bash
bacalhau docker run ubuntu echo Hello World
```

Download your result

```bash
bacalhau get 63d08ff0..... # make sure to use the right job id from the docker run command
```

:::info
For a more detailed tutorial, check out our [Getting Started tutorial](https://docs.bacalhau.org/getting-started/installation).
:::


## How it works

The goal of the Bacalhau project is to make it easy to perform distributed, decentralised computation next to where the data resides. So a key step in this process is making your data accessible. Data is identified by its content identifier (CID) and can be accessed by anyone who knows the CID. Here are some options that can help you mount your data:

- [Copy data from a URL to public storage](https://docs.bacalhau.org/examples/data-ingestion/from-url/)
- [Pin Data to public storage](https://docs.bacalhau.org/examples/data-ingestion/pinning/)
- [Copy Data from S3 Bucket to public storage](https://docs.bacalhau.org/examples/data-ingestion/s3-to-ipfs/)

:::info
The options are not limited to the above mentioned. You can mount your data anywhere on your machine, and Bacalhau will be able to run against that data
:::

### Use Cases

Bacalhau shines when it comes to data-intensive applications like [data engineering](https://docs.bacalhau.org/examples/data-engineering/), [model training](https://docs.bacalhau.org/examples/model-training/), [model inference](https://docs.bacalhau.org/examples/model-inference/), [model training](https://docs.bacalhau.org/examples/model-training/), [model dynanmics](https://docs.bacalhau.org/examples/molecular-dynamics/), etc.

Here are some example tutorials on how you can process your data with Bacalhau:
- [Image processing with Bacalhau](https://docs.bacalhau.org/examples/model-inference/stable-diffusion-gpu/)
- [Generate Realistic Images using StyleGAN3 and Bacalhau](https://docs.bacalhau.org/examples/model-inference/StyleGAN3/)
- [Object Detection with YOLOv5 on Bacalhau](https://docs.bacalhau.org/examples/model-inference/object-detection-yolo5/)
- [Running Genomics on Bacalhau](https://docs.bacalhau.org/examples/miscellaneous/Genomics/)
- [Training Pytorch Model with Bacalhau](https://docs.bacalhau.org/examples/model-training/Training-Pytorch-Model/)

:::info
For more tutorials, visit our [example page](https://docs.bacalhau.org/examples/)
:::

## Roadmap

Initially, the Bacalhau project will focus on serving data processing and analytics use cases. Over time, Bacalhau will expand to other Compute workloads. You can find Bacalhau's [Public Roadmap here](https://starmap.site/roadmap/github.com/bacalhau-project/bacalhau/issues/1151)!

## Community

Bacalhau has a very friendly community and we are always happy to help you get started:

- [GitHub Discussions](https://github.com/bacalhau-project/bacalhau/discussions) – ask anything about the project, give feedback or answer questions that will help other users.
- [Join the Slack Community](https://filecoin.io/slack) and go to **#bacalhau** channel – it is the easiest way engage with other members in the community and get help.
- [Contributing](https://docs.bacalhau.org/community/ways-to-contribute) – learn how to contribute to the Bacalhau project.

## Next Steps

👉 Continue with [Getting Started guide](/docs/getting-started/installation.md) to learn how to install and run a job with the Bacalhau client.

👉 Or jump directly to try out the different [Examples](/docs/examples/index.md) that showcases Bacalhau abilities.
