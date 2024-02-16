---
sidebar_label: Coresets On Bacalhau
sidebar_position: 2
---
# Coresets On Bacalhau



[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

[Coreset ](https://arxiv.org/abs/2011.09384)is a data subsetting method. Since the uncompressed datasets can get very large when compressed, it becomes much harder to train them as training time increases with the dataset size. To reduce the training time to save costs we use the coreset method; the coreset method can also be applied to other datasets. In this case, we use the coreset method which can lead to a fast speed in solving the k-means problem among the big data with high accuracy in the meantime.

We construct a small coreset for arbitrary shapes of numerical data with a decent time cost. The implementation was mainly based on the coreset construction algorithm that was proposed by Braverman et al. (SODA 2021).

## TD:LR
Running compressed dataset with Bacalhau


## Running Locally

Clone the repo which contains the code



```bash
%%bash
git clone https://github.com/js-ts/Coreset
```


To download the dataset you should open Street Map, which is a public repository that aims to generate and distribute accessible geographic data for the whole world. Basically, it supplies detailed position information, including the longitude and latitude of the places around the world.

The dataset is a osm.pbf (compressed format for .osm file), the file can be downloaded from [Geofabrik Download Server](https://download.geofabrik.de/)



```bash
%%bash
wget https://download.geofabrik.de/europe/liechtenstein-latest.osm.pbf -o liechtenstein-latest.osm.pbf
```


The following command is installing Linux dependencies:



```bash
%%bash
sudo apt-get -y update \
sudo apt-get -y install osmium-tool \
sudo apt update \
sudo apt-get -y install libpq-dev gdal-bin libgdal-dev libxml2-dev libxslt-dev
```

The following command is installing Python dependencies:



```bash
%%bash
pip3 install -r Coreset/requirements.txt
```

To run coreset locally, you need to convert from compressed pbf format to geojson format:


```bash
%%bash
osmium export liechtenstein-latest.osm.pbf -o liechtenstein-latest.geojson
```

The following command is to run the Python script to generate the coreset:


```bash
%%bash
python Coreset/python/coreset.py -f liechtenstein-latest.geojson
```

## Containerize Script using Docker

To build your own docker container, create a `Dockerfile`, which contains instructions on how the image will be built, and what extra requirements will be included.

```
FROM python:3.8

RUN apt-get -y update && apt-get -y install osmium-tool && apt update && apt-get -y install libpq-dev gdal-bin libgdal-dev libxml2-dev libxslt-dev

ADD Coreset Coreset

ADD monaco-latest.geojson .

RUN cd Coreset && pip3 install -r requirements.txt
```


We will use the `python:3.8` image, and we will choose the src directory in the container as our work directory, we run the same commands for installing dependencies that we used locally, but we also add files and directories which are present on our local machine, we also run a test command, in the end, to check whether the script works

:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

We will run `docker build` command to build the container;

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace;

- **hub-user** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

- **repo-name** with the name of the container, you can name it anything you want

- **tag** this is not required but you can use the latest tag

In our case

```bash
docker build -t jsace/coreset
```

### Push the container

Next, upload the image to the registry. This can be done by using the Docker hub username, repo name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

In our case

```bash
docker push jsace/coreset
```


## Running a Bacalhau Job

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. To submit a job, run the following Bacalhau command:

```
bacalhau docker run \
-i ipfs://QmXuatKaWL24CwrBPC9PzmLW8NGjgvBVJfk6ZGCWUGZgCu:/input \
jsace/coreset \
-- /bin/bash -c 'osmium export input/liechtenstein-latest.osm.pbf -o liechtenstein-latest.geojson;
python Coreset/python/coreset.py -f input/liechtenstein-latest.geojson -o outputs'
```


Backend: Docker backend here for running the job

* `input/liechtenstein-latest.osm.pbf`: Upload the .osm.pbf file

* `-i ipfs://QmXuatKaWL24CwrBPC9PzmLW8NGjgvBVJfk6ZGCWUGZgCu:/input`: mount dataset to the folder inside the container so it can be used by the script

* `jsace/coreset`:  the name and the tag of the docker image we are using


The following command converts the osm.pbf dataset to geojson (the dataset is stored in the input volume folder):

```
osmium export input/.osm.pbf -o liechtenstein-latest.geojson
```

Let's run the script, we use flag `-f` to determine the path of the output geojson file from the step above.

```
python Coreset/python/coreset.py -f liechtenstein-latest.geojson -o outputs
```

We get the output in stdout

Additional parameters:
* `-k`: amount of initialized centers (default=5)

* `-n`: size of coreset (default=50)

* `-o`: the output folder

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.



```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
ls results/
```

To view the output as a CSV file, run:


```bash
%%bash
cat results/outputs/centers.csv | head -n 10
```


```bash
%%bash
cat results/outputs/coreset-values-liechtenstein-latest.csv | head -n 10
```


```bash
%%bash
cat results/outputs/coreset-weights-liechtenstein-latest.csv | head -n 10
```


#### Sources

[1] [http://proceedings.mlr.press/v97/braverman19a/braverman19a.pdf](http://proceedings.mlr.press/v97/braverman19a/braverman19a.pdf)

[2][https://aaltodoc.aalto.fi/bitstream/handle/123456789/108293/master_Wu_Xiaobo_2021.pdf?sequence=2](https://aaltodoc.aalto.fi/bitstream/handle/123456789/108293/master_Wu_Xiaobo_2021.pdf?sequence=2)
