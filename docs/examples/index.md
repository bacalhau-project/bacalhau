---
sidebar_label: "Examples"
sidebar_position: 20
---
# Examples

Bacalhau comes pre-loaded with exciting examples to showcase its abilities and help get you started.

## Organisation

Examples are organised by task. Each task has a number of examples that showcase different ways to solve the same problem.

:::tip

We're adding examples all the time. Check back soon for more!

:::

## Usage

Each example is a self-contained [Jupyter notebook](https://docs.jupyter.org/en/latest/) that can be run locally or on your favourite Jupyter host (Google Colab or binder). The main advantage of Jupiter notebooks is that user can run examples by pressing 'Run All' button; it contains desciptive text next to each code block with output in a single static page.

Instead of `bash` Jupiter notebooks use [iPython magic links](https://ipython.readthedocs.io/en/stable/interactive/magics.html#cell-magics), where command starts with `%%`.

In order:  
* To run on the free cloud, use Collab/binder buttons on the top of each example. 

* To run locally, you need to install [jupiter](https://jupyterlab.readthedocs.io/en/stable/getting_started/installation.html). The `.ipynb` file is where the source code lives in each of our [examples](https://github.com/bacalhau-project/examples). 

Data is typically stored externally in [GCS](https://cloud.google.com/docs) (for remote data examples) or [IPFS](https://docs.ipfs.tech/) (for local data examples).

The examples execute on the Bacalhau public network (a.k.a. `mainnet`).

## Developer Information

All of the examples are open source and available on GitHub. The examples exist in a separate repository located at https://github.com/bacalhau-project/examples/. Please see the [README.md for more instructions on how to contribute](https://github.com/bacalhau-project/examples/README.md).

Note that the code for the rest of the documentation website [is located in a separate repository](https://github.com/bacalhau-project/docs.bacalhau.org/).
