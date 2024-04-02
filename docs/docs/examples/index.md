---
sidebar_label: "Examples"
sidebar_position: 3
---
# Examples

Bacalhau comes pre-loaded with exciting examples to showcase its abilities and help get you started.

:::tip

Use the navigation bar on the left to browse through the examples. You can also view the raw codebase by visiting our [examples repository](https://github.com/bacalhau-project/examples).

:::

## Organisation

Examples are organized by task. Each task has a number of examples that showcase different ways to solve the same problem.

:::tip

We're adding examples all the time. Check back soon for more!

:::

## Usage

Each example walks you through a specific task and aims at being as self-contained as possible.
For starters, we recommended starting from examples whose prerequisites include only the Bacalhau client (e.g. [Python Hello World](../setting-up/workload-onboarding/Prolog-Hello-World/index.md)).
First read through an example to grasp its objectives and syntax, once you feel confident with those you may want to run it *interactively*.

### Structure

Right at the top you'll see colorful badges reading "Open in Colab" or "launch binder". These are shortcuts to execute an example in interactive mode (more on this below).

After a brief introduction, each example lists a number of prerequisites. These can span from a simple Bacalhau client installation to installing a Docker daemon or NVIDIA drivers.
Clearly, there's a limit to what you can install on a machine to just try out a Bacalhau example.
For instance, you cannot install NVIDIA driver on a Macbook Pro simply because they don't ship with NVIDIA GPUs.
Luckily, that is no problem because for the vast majority of the examples are provided with pre-packaged cloud runtime environments (more on this in the interactive mode section).

Data is typically stored externally in [Google Cloud Storage (GCS)](https://en.wikipedia.org/wiki/Google_Cloud_Storage) (for remote data examples) or [IPFS](https://docs.ipfs.tech/) (for local data examples).
Sometimes examples ship with datasets stored locally and you may find references of the likes of `./data/train.csv`.


Typically each example ends by downloading your Bacalhau job's outputs locally.
This may feel repetitive but it's helpful to display the actual results of your job!

### Syntax primer

Spread across each example you'll find **blocks** like the one below.
As you go through an example, you'll need to understand the nature of these blocks and how to interpret them.

```
Hello reader, I'm a block!
```


Our examples are written in [Jupyter notebooks](https://jupyterlab.readthedocs.io/en/stable/index.html), a rich format that pulls together descriptive text, various **blocks** and the possibility to run bash commands from within a notebook.
This gives you the power to interactively run our examples (more on this below - last teaser about this, promised :smile:)!
An advantage of notebooks is that once you "run it", it'll store the bash commands' output in a dedicated block.
Thanks to that, the static webpages you find in https://docs.bacalhau.org/examples/ are effectively "snapshots" of previous runs.
This way, you don't necessarily need to run an example to see what it outputs!

:::info

To achieve the above, *some* blocks are annotated with the following [cell magics](https://ipython.readthedocs.io/en/stable/interactive/magics.html#cell-magics): `%%bash` and `%%writefile`.
These tell Jupyter how to run the commands within a block.
If you're just reading through the static webpage, these annotations shall inform you that block aims at either execute bash commands, or write the content of that block to a file on disk.

:::

Thus, blocks can be:

* simple text snippets: these are used to display generic text in a dedicated block.
* bash commands (annotated with `%%bash` in the first line): these can be run in the interactive mode! If you wish to use your own terminal to launch these commands, just ignore the `%%bash` line.
* write to file (annotated with `%%writefile <path/file-name>` in the first line): these inform you the remainder of the example will expect `<path/file-name>` to be stored on disk. We use these blocks to:
  * Firstly, show you what's inside a file - for instance, this may display the content of a Python script.
  * Secondly, when running an example in interactive mode, Jupyter will effectively write out the content of the block to disk, in `<path/file-name>`.

Since our examples are runnable, we render the effective run in static web pages to give you a glimpse of what to expect if you run it yourself.
This means we're able to show you what the output of a command is, right below the command block.
For instance, when you run into two consecutive blocks (see below), the former represents a command block and the latter depicts its output.

```python
%%bash
date
```

    Wed Feb 15 13:21:35 CET 2029


### Interactive mode

If you're trying to run an example by yourself, this section contains the instructions you're looking for.
As stated previously, our examples are written in Jupyter Notebook and this gives you the possibility to run its steps one by one, edit them, or simply run them all in a sequence from top to bottom.
This is convenient because while the accompanying text guides you through the example (and hopefully provides enough context), you have to possibility to edit and re-run each step.

:::tip

Are you not familiar with Jupyter Notebooks but wish to run our examples interactively?
Please stop right here and take a moment to learn more about it in its [official docs](https://jupyterlab.readthedocs.io/en/stable/index.html).
This [video on YouTube (7min)](https://www.youtube.com/watch?v=jZ952vChhuI) is a perfect quick introduction to Jupyter Notebook.
See also [the difference between Jupyter Notebook and JupyterLab](https://stackoverflow.com/questions/50982686/what-is-the-difference-between-jupyter-notebook-and-jupyterlab/).

:::

Finally, how to run our examples interactively?
You have two options:

#### Run examples locally

1. Pick an example you'd like to run
1. Find the corresponding `.ipynb` file at https://github.com/bacalhau-project/examples
1. Pull the example repo locally: `git clone https://github.com/bacalhau-project/examples.git`
1. Install Jupyter Notebook or preferably [install JupyterLab](https://jupyterlab.readthedocs.io/en/stable/getting_started/installation.html) locally
1. Launch JupyterLab, run `jupyter lab` in a terminal (see the [official docs for more details](https://jupyterlab.readthedocs.io/en/stable/getting_started/starting.html))
1. Use the [Jupyter interface](https://jupyterlab.readthedocs.io/en/stable/user/interface.html) to interact with the example of your choice

Can you run notebooks in VS Code? Install [the related extension](https://marketplace.visualstudio.com/items?itemName=ms-toolsai.jupyter), then clone the repo as in the steps above.

:::tip

Spare yourself the hustle of installing all the above and use a hosted Jupyter service instead (see the section below).

:::
#### Use a hosted Jupyter service (recommended!)

Most examples come with badges on the top of the page <img src="https://colab.research.google.com/assets/colab-badge.svg" alt="Google Colab logo" /> <img src="https://mybinder.org/badge.svg" alt="mybinder.org logo" />.
Those badges are clickable and they'll open a Colab/Binder workspace (in the cloud) with the notebook and related files ready to go.
They typically work out of the box and perfectly support installing tools like the Bacalhau client or a Python library.
However, it must be noted they may complain when trying to install advanced prerequisites that require system-level components (e.g. Docker). When you run into that case, you'll have to resort to running the notebook and installing all of its dependencies locally (see section above).

## Developer Information

All of the examples are open source and available on GitHub. The examples exist in a separate repository located at https://github.com/bacalhau-project/examples/. Please see the [README.md for more instructions on how to contribute](https://github.com/bacalhau-project/examples/README.md).

Note that the code for the rest of the documentation (this website) [is located in a separate repository](https://github.com/bacalhau-project/bacalhau/tree/main/docs).
