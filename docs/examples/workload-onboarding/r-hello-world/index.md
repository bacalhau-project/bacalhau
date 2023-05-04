---
sidebar_label: "R Script"
sidebar_position: 9
---
# Running a Simple R Script in Bacalhau


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

You can use official Docker containers for each language like R or Python. In this example, we will use the official R container and run it on bacalhau. 

## TD;LR
A quick guide on how to run a hello world script on Bacalhau

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Running an R Script Locally

To install R follow these instructions [A Installing R and RStudio | Hands-On Programming with R](https://rstudio-education.github.io/hopr/starting.html). After R and RStudio is installed, create and run a script called hello.R.


```python
%%writefile hello.R
print("hello world")
```

Run the script:


```bash
%%bash
Rscript hello.R
```

Next, upload the script to your public storage in our case IPFS.  We've already uploaded the script to IPFS and the CID is: `QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie`. You can look at this by browsing to one of the HTTP IPFS proxies like [ipfs.io](https://cloudflare-ipfs.com/ipfs/QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie/) or [w3s.link](https://w3s.link/ipfs/QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie).

## Running a Job on Bacalhau

Now it's time to run the script on the Bacalhau network. To run a job on Bacalhau, run the following command:


```bash
%%bash --out job_id
bacalhau docker run \
--wait \
--id-only \
-i ipfs://QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R \
r-base \
-- Rscript hello.R
```

### Structure of the command

Let's look closely at the command above:

* `bacalhau docker run`: call to bacalhau 
  
* `-i ipfs://QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk`: CIDs to use on the job. Mounts them at '/inputs' in the execution.

* `:/hello.R`: the name and the tag of the docker image we are using

* `Rscript hello.R`: execute the R script


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`.



```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe  ${JOB_ID}
```

- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
ls results/
```

Viewing the result


```bash
%%bash
cat results/stdout
```

### Futureproofing your R Scripts

You can generate the the job request with the following command. This will allow you to re-run that job in the future.


```bash
%%bash
bacalhau describe ${JOB_ID} --spec > job.yaml
```


```bash
%%bash
cat job.yaml
```
