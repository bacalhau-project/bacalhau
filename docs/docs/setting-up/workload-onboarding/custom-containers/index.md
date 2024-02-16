---
sidebar_label: "Custom Containers"
sidebar_position: 11
---
# How To Work With Custom Containers in Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

Bacalhau operates by executing jobs within containers. This example shows you how to build and use a custom docker container.

### Prerequisite

1. To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)
2. This example requires Docker. If you don't have Docker installed, you can install it from [here](https://docs.docker.com/install/). Docker commands will not work on hosted notebooks like Google Colab, but the Bacalhau commands will.


##  1. Running Containers

### Docker Command
You're likely familiar with executing Docker commands to start a container:


```bash
%%bash
docker run docker/whalesay cowsay sup old fashioned container run
```

This command runs a container from the `docker/whalesay` image.
The container executes the `cowsay sup old fashioned container run` command:

```shell
Expected output:
_________________________________
< sup old fashioned container run >
 ---------------------------------
    \
     \
      \
                    ##        .
              ## ## ##       ==
           ## ## ## ##      ===
       /""""""""""""""""___/ ===
  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
       \______ o          __/
        \    \        __/
          \____\______/
```

### Bacalhau Command

```bash
%%bash --out job_id
bacalhau docker run --wait --id-only docker/whalesay -- bash -c 'cowsay hello web3 uber-run'
```

This command also runs a container from the `docker/whalesay` image, using Bacalhau. We use the `bacalhau docker run` command to start a job in a Docker container.
It contains additional flags such as `--wait` to wait for job completion and `--id-only` to return only the job identifier.
Inside the container, the `bash -c 'cowsay hello web3 uber-run'` command is executed.

When a job is submitted, Bacalhau prints out the related `job_id` (`7e41b9b9-a9e2-4866-9fce-17020d8ec9e0`):

```shell
7e41b9b9-a9e2-4866-9fce-17020d8ec9e0
```
We store that in an environment variable so that we can reuse it later on.

```python
%env JOB_ID={job_id}
```

You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`results`) and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get ${JOB_ID}  --output-dir results
```

Viewing your job output


```bash
%%bash
cat ./results/stdout

Expected output:

 _____________________
< hello web3 uber-run >
 ---------------------
    \
     \
      \
                    ##        .
              ## ## ##       ==
           ## ## ## ##      ===
       /""""""""""""""""___/ ===
  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
       \______ o          __/
        \    \        __/
          \____\______/
```

Both commands execute cowsay in the `docker/whalesay` container, but Bacalhau provides additional features for working with jobs at scale.

### Bacalhau Syntax
Bacalhau uses a syntax that is similar to Docker, and you can use the same containers. The main difference is that input and output data is passed to the container via IPFS, to enable planetary scale. In the example above, it doesn't make too much difference except that we need to download the stdout.

The `--wait` flag tells Bacalhau to wait for the job to finish before returning. This is useful in interactive sessions like this, but you would normally allow jobs to complete in the background and use the `bacalhau list` command to check on their status.

Another difference is that by default Bacalhau overwrites the default entry point for the container, so you have to pass all shell commands as arguments to the `run` command after the `--` flag.




## 2. Building Your Own Custom Container For Bacalhau

To use your own custom container, you must publish the container to a container registry that is accessible from the Bacalhau network. At this time, only public container registries are supported.

To demonstrate this, you will develop and build a simple custom container that comes from an old Docker example. I remember seeing cowsay at a Docker conference about a decade ago. I think it's about time we brought it back to life and distribute it across the Bacalhau network.


```python
%%writefile cod.cow
$the_cow = <<"EOC";
   $thoughts
    $thoughts
                               ,,,,_
                            ┌Φ▓╬▓╬▓▓▓W      @▓▓▒,
                           ╠▓╬▓╬╣╬╬▓╬▓▓   ╔╣╬╬▓╬╣▓,
                    __,┌╓═╠╬╠╬╬╬Ñ╬╬╬Ñ╬╬¼,╣╬╬▓╬╬▓╬▓▓▓┐        ╔W_             ,φ▓▓
               ,«@▒╠╠╠╠╩╚╙╙╩Ü╚╚╚╚╩╙╙╚╠╩╚╚╟▓▒╠╠╫╣╬╬╫╬╣▓,   _φ╬▓╬╬▓,        ,φ╣▓▓╬╬
          _,φÆ╩╬╩╙╚╩░╙╙░░╩`=░╙╚»»╦░=╓╙Ü1R░│░╚Ü░╙╙╚╠╠╠╣╣╬≡Φ╬▀╬╣╬╬▓▓▓_   ╓▄▓▓▓▓▓▓╬▌
      _,φ╬Ñ╩▌▐█[▒░░░░R░░▀░`,_`!R`````╙`-'╚Ü░░Ü░░░░░░░│││░╚╚╙╚╩╩╩╣Ñ╩╠▒▒╩╩▀▓▓╣▓▓╬╠▌
     '╚╩Ü╙│░░╙Ö▒Ü░░░H░░R ▒¥╣╣@@@▓▓▓  := '`   `░``````````````````````````]▓▓▓╬╬╠H
       '¬═▄ `\░╙Ü░╠DjK` Å»»╙╣▓▓▓▓╬Ñ     -»`       -`      `  ,;╓▄╔╗∞  ~▓▓▓▀▓▓╬╬╬▌
             '^^^`   _╒Γ   `╙▀▓▓╨                     _, ⁿD╣▓╬╣▓╬▓╜      ╙╬▓▓╬╬▓▓
                 ```└                           _╓▄@▓▓▓╜   `╝╬▓▓╙           ²╣╬▓▓
                        %φ▄╓_             ~#▓╠▓▒╬▓╬▓▓^        `                ╙╙
                         `╣▓▓▓              ╠╬▓╬▓╬▀`
                           ╚▓▌               '╨▀╜
EOC
```

Next, the Dockerfile adds the script and sets the entry point.


```python
%%writefile Dockerfile
FROM debian:stretch
RUN apt-get update && apt-get install -y cowsay
# "cowsay" installs to /usr/games
ENV PATH $PATH:/usr/games
RUN echo '#!/bin/bash\ncowsay "${@:1}"' > /usr/bin/codsay && \
    chmod +x /usr/bin/codsay
COPY cod.cow /usr/share/cowsay/cows/default.cow
```

Now let's build and test the container locally.


```bash
%%bash
docker build -t ghcr.io/bacalhau-project/examples/codsay:latest . 2> /dev/null
```


```bash
%%bash
docker run --rm ghcr.io/bacalhau-project/examples/codsay:latest codsay I like swimming in data
```

Once your container is working as expected then you should push it to a public container registry. In this example, I'm pushing to Github's container registry, but we'll skip the step below because you probably don't have permission. Remember that the Bacalhau nodes expect your container to have a `linux/amd64` architecture.


```bash
%%bash
# docker buildx build --platform linux/amd64,linux/arm64 --push -t ghcr.io/bacalhau-project/examples/codsay:latest .
```

## 3. Running Your Custom Container on Bacalhau

Now we're ready to submit a Bacalhau job using your custom container. This code runs a job, downloads the results, and prints the stdout.

:::tip
The `bacalhau docker run` command strips the default entry point, so don't forget to run your entry point in the command line arguments.
:::


```bash
%%bash --out job_id
bacalhau docker run \
    --wait \
    --id-only \
    ghcr.io/bacalhau-project/examples/codsay:v1.0.0 \
    -- bash -c 'codsay Look at all this data'
```

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

Download your job results directly by using `bacalhau get` command.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get ${JOB_ID}  --output-dir results
```

View your job output


```bash
%%bash
cat ./results/stdout

Expected output:

_______________________
< Look at all this data >
 -----------------------
   \
    \
                               ,,,,_
                            ┌Φ▓╬▓╬▓▓▓W      @▓▓▒,
                           ╠▓╬▓╬╣╬╬▓╬▓▓   ╔╣╬╬▓╬╣▓,
                    __,┌╓═╠╬╠╬╬╬Ñ╬╬╬Ñ╬╬¼,╣╬╬▓╬╬▓╬▓▓▓┐        ╔W_             ,φ▓▓
               ,«@▒╠╠╠╠╩╚╙╙╩Ü╚╚╚╚╩╙╙╚╠╩╚╚╟▓▒╠╠╫╣╬╬╫╬╣▓,   _φ╬▓╬╬▓,        ,φ╣▓▓╬╬
          _,φÆ╩╬╩╙╚╩░╙╙░░╩`=░╙╚»»╦░=╓╙Ü1R░│░╚Ü░╙╙╚╠╠╠╣╣╬≡Φ╬▀╬╣╬╬▓▓▓_   ╓▄▓▓▓▓▓▓╬▌
      _,φ╬Ñ╩▌▐█[▒░░░░R░░▀░`,_`!R`````╙`-'╚Ü░░Ü░░░░░░░│││░╚╚╙╚╩╩╩╣Ñ╩╠▒▒╩╩▀▓▓╣▓▓╬╠▌
     '╚╩Ü╙│░░╙Ö▒Ü░░░H░░R ▒¥╣╣@@@▓▓▓  := '`   `░``````````````````````````]▓▓▓╬╬╠H
       '¬═▄ `░╙Ü░╠DjK` Å»»╙╣▓▓▓▓╬Ñ     -»`       -`      `  ,;╓▄╔╗∞  ~▓▓▓▀▓▓╬╬╬▌
             '^^^`   _╒Γ   `╙▀▓▓╨                     _, ⁿD╣▓╬╣▓╬▓╜      ╙╬▓▓╬╬▓▓
                 ```└                           _╓▄@▓▓▓╜   `╝╬▓▓╙           ²╣╬▓▓
                        %φ▄╓_             ~#▓╠▓▒╬▓╬▓▓^        `                ╙╙
                         `╣▓▓▓              ╠╬▓╬▓╬▀`
                           ╚▓▌               '╨▀╜
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
