---
sidebar_label: Coresets On Bacalhau
sidebar_position: 2
---
# Coresets On Bacalhau

[Coreset ](https://arxiv.org/abs/2011.09384)is a data subsetting method. Since the uncompressed datasets can get very large when compressed, it becomes much harder to train them as training time increases with the dataset size. To reduce training time and cut costs, we employ the coreset method; the coreset method can also be applied to other datasets. In this case, we use the coreset method which can lead to a fast speed in solving the k-means problem among the big data with high accuracy in the meantime.

We construct a small coreset for arbitrary shapes of numerical data with a decent time cost. The implementation was mainly based on the coreset construction algorithm that was proposed by Braverman et al. (SODA 2021).

:::info
For a deeper understanding of the core concepts, it's recommended to explore:    
[1] [Coresets for Ordered Weighted Clustering](http://proceedings.mlr.press/v97/braverman19a/braverman19a.pdf)  
[2] [Efficient Implementation of Coreset-based K-Means Methods](https://aaltodoc.aalto.fi/bitstream/handle/123456789/108293/master_Wu_Xiaobo_2021.pdf?sequence=2)
:::


In this tutorial example, we will run compressed dataset with Bacalhau

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)



## Running Locally

Clone the repo which contains the code



```bash
%%bash
git clone https://github.com/js-ts/Coreset
```

### Downloading the dataset

To download the dataset you should open Street Map, which is a public repository that aims to generate and distribute accessible geographic data for the whole world. Basically, it supplies detailed position information, including the longitude and latitude of the places around the world.

The dataset is a osm.pbf (compressed format for .osm file), the file can be downloaded from [Geofabrik Download Server](https://download.geofabrik.de/)



```bash
%%bash
wget https://download.geofabrik.de/europe/monaco-latest.osm.pbf
```

### Installing Dependencies

The following command is installing Linux dependencies:



```bash
%%bash
sudo apt-get -y update \
sudo apt-get -y install osmium-tool \
sudo apt-get -y install libpq-dev gdal-bin libgdal-dev libxml2-dev libxslt-dev
```

Ensure that the `requirements.txt` file contains the following dependencies: 

```python
%%writefile requirements.txt
certifi==2020.12.5
chardet==4.0.0
cycler==0.10.0
idna==2.10
joblib
kiwisolver==1.3.1
lxml==4.6.2
matplotlib==3.3.3
numpy==1.19.4
overpy==0.4
pandas==1.1.4
Pillow==8.0.1
pyparsing==2.4.7
python-dateutil==2.8.1
pytz==2020.4
requests==2.25.1
scikit-learn
scipy
six==1.15.0
threadpoolctl
tqdm==4.56.0
urllib3==1.26.2
geopandas
```

The following command is installing Python dependencies:


```bash
%%bash
pip3 install -r Coreset/requirements.txt
```


### Running the Script

To run coreset locally, you need to convert from compressed pbf format to geojson format:


```bash
%%bash
osmium export monaco-latest.osm.pbf -o monaco-latest.geojson
```

The following command is to run the Python script to generate the coreset:


```bash
%%bash
python Coreset/python/coreset.py -f monaco-latest.geojson
```

:::info
`coreset.py` contains the following script [here](https://github.com/js-ts/Coreset/blob/master/Coreset/python/coreset.py)
:::

## Containerize Script using Docker

To build your own docker container, create a `Dockerfile`, which contains instructions on how the image will be built, and what extra requirements will be included.

```
FROM python:3.8

RUN apt-get -y update && apt-get -y install osmium-tool && apt update && apt-get -y install libpq-dev gdal-bin libgdal-dev libxml2-dev libxslt-dev

ADD Coreset Coreset

ADD monaco-latest.geojson .

RUN cd Coreset && pip3 install -r requirements.txt
```

We will use the `python:3.8` image, we run the same commands for installing dependencies that we used locally.

:::info
See more information on how to containerize your script/app [here](https://docs.docker.com/get-started/02_our_app/)
:::


### Build the container

We will run `docker build` command to build the container:

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace:

**`hub-user`** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

**`repo-name`** with the name of the container, you can name it anything you want

**`tag`** this is not required but you can use the latest tag

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

After the repo image has been pushed to Docker Hub, we can now use the container for running on Bacalhau. We've already converted the `monaco-latest.osm.pbf` file from compressed pbf format to geojson format  [here](https://github.com/js-ts/Coreset/blob/master/monaco-latest.geojson). To submit a job, run the following Bacalhau command:

```
bacalhau docker run \
    --input https://github.com/js-ts/Coreset/blob/master/monaco-latest.geojson \
    jsace/coreset \
    -- /bin/bash -c 'python Coreset/python/coreset.py -f monaco-latest.geojson -o outputs'
```
### Structure of the command

Let's look closely at the command above:

1. `bacalhau docker run`: call to bacalhau
1. `--input https://github.com/js-ts/Coreset/blob/master/monaco-latest.geojson`: mount the `monaco-latest.geojson` file inside the container so it can be used by the script
1. `jsace/coreset`:  the name of the docker image we are using
1. `python Coreset/python/coreset.py -f monaco-latest.geojson -o outputs`: the script initializes cluster centers, creates a coreset using these centers, and saves the results to the specified folder.


**Additional parameters:**  

**`-k`**: amount of initialized centers (default=5)

**`-n`**: size of coreset (default=50)

**`-o`**: the output folder

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this:

```yaml
name: Coresets On Bacalhau
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        Image: "jsace/coreset" 
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c
          - "osmium export input/liechtenstein-latest.osm.pbf -o /liechtenstein-latest.geojson;python Coreset/python/coreset.py -f /liechtenstein-latest.geojson -o /outputs"
    Publisher:
      Type: ipfs
    ResultPaths:
      - Name: outputs
        Path: /outputs      
    InputSources:
      - Source:
          Type: "s3"
          Params:
            Bucket: "coreset"
            Key: "*"
            Region: "us-east-1"
        Target: "/input"    
```

The job description should be saved in `.yaml` format, e.g. `coreset.yaml`, and then run with the command:
```bash
bacalhau job run coreset.yaml
```

## Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.



```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

**Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe ${JOB_ID}
```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`results`) and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:

```bash
%%bash
ls results/outputs

Expected Output:
centers.csv                       coreset-weights-monaco-latest.csv
coreset-values-monaco-latest.csv  ids.csv
```

To view the output as a CSV file, run:


```bash
%%bash
cat results/outputs/centers.csv | head -n 10

Expected Output:
lat,long
7.423843975787508,43.730621154072196
7.4252607,43.7399135
7.411026970571964,43.72937671121925
7.459404485446199,43.62065587026715
7.429551373022234,43.74042043301333
```


```bash
%%bash
cat results/outputs/coreset-values-monaco-latest.csv | head -n 10

Expected Output:
7.418849799999999384e+00,4.372759140000000144e+01
7.416779063194204547e+00,4.373053835217195484e+01
7.422073648233502574e+00,4.374059957604499971e+01
7.434173206469590234e+00,4.374591689556921636e+01
7.417540100000000081e+00,4.372501400000000160e+01
7.427359010538406636e+00,4.374324133692341121e+01
7.427839200000001085e+00,4.374025220000000758e+01
7.418834173612560257e+00,4.372760402368248833e+01
7.416381731248183229e+00,4.373708812663696932e+01
7.412050699999999992e+00,4.372842109999999849e+01
```


```bash
%%bash
cat results/outputs/coreset-weights-monaco-latest.csv | head -n 10

Expected Output:
7.704359156916230233e+01
2.090893934427382987e+02
1.560611140982714744e+02
2.516557569411126281e+02
7.714605094768158722e+01
2.640808776415075840e+02
2.326085291610944523e+02
7.704841021255269595e+01
2.089705263763523249e+02
1.728105655128551632e+02
```

<<<<<<< HEAD
## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
=======

#### Sources

[1] [http://proceedings.mlr.press/v97/braverman19a/braverman19a.pdf](http://proceedings.mlr.press/v97/braverman19a/braverman19a.pdf)

[2][https://aaltodoc.aalto.fi/bitstream/handle/123456789/108293/master_Wu_Xiaobo_2021.pdf?sequence=2](https://aaltodoc.aalto.fi/bitstream/handle/123456789/108293/master_Wu_Xiaobo_2021.pdf?sequence=2)
>>>>>>> main
