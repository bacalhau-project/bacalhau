---
sidebar_label: "R Script"
sidebar_position: 9
---
# Running a Simple R Script on Bacalhau


You can use official Docker containers for each language like R or Python. In this example, we will use the official R container and run it on Bacalhau.

In this tutorial example, we will run a "hello world" R script on Bacalhau.

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## 1. Running an R Script Locally

To install R follow these instructions [A Installing R and RStudio | Hands-On Programming with R](https://rstudio-education.github.io/hopr/starting.html). After R and RStudio are installed, create and run a script called `hello.R`:


```python
%%writefile hello.R
print("hello world")
```

Run the script:


```bash
%%bash
Rscript hello.R
```

Next, upload the script to your public storage (in our case, IPFS).  We've already uploaded the script to IPFS and the CID is: `QmVHSWhAL7fNkRiHfoEJGeMYjaYZUsKHvix7L54SptR8ie`. You can look at this by browsing to one of the HTTP IPFS proxies like [ipfs.io](https://ipfs.tech/) or [w3s.link](https://github.com/web3-storage/w3link).

## 2. Running a Job on Bacalhau

Now it's time to run the script on Bacalhau:


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

`bacalhau docker run`: call to Bacalhau

`i ipfs://QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R`: Mounting the uploaded dataset at `/inputs` in the execution. It takes two arguments, the first is the IPFS CID (`QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk`) and the second is file path within IPFS (`/hello.R`)

`r-base`: docker official image we are using

`Rscript hello.R`: execute the R script


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on:


```python
%env JOB_ID={job_id}
```
### Declarative job description

The same job can be presented in the [declarative](../../../setting-up/jobs/job-specification/job.md) format. In this case, the description will look like this:

```yaml
name: Running a Simple R Script
type: batch
count: 1
tasks:
  - name: My main task
    Engine:
      type: docker
      params:
        Image: r-base:latest
        Entrypoint:
          - /bin/bash
        Parameters:
          - -c        
          - Rscript /hello.R
    InputSources:
      - Target: "/"
        Source:
          Type: urlDownload
          Params:
            URL: https://raw.githubusercontent.com/bacalhau-project/examples/main/scripts/hello.R
            Path: /hello.R
```

The job description should be saved in `.yaml` format, e.g. `rhello.yaml`, and then run with the command:
```bash
bacalhau job run rhello.yaml
```

## 3. Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.



```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
```

When it says `Published` or `Completed`, that means the job is done, and we can get the results.

**Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
bacalhau describe  ${JOB_ID}
```

**Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`results`) and downloaded our job output to be stored in that directory.


```bash
%%bash
rm -rf results && mkdir results
bacalhau get ${JOB_ID} --output-dir results
```

## 4. Viewing your Job Output

To view the file, run the following command:

```bash
%%bash
cat results/stdout
```

### Futureproofing your R Scripts

You can generate the job request using `bacalhau describe` with the `--spec` flag. This will allow you to re-run that job in the future:


```bash
%%bash
bacalhau describe ${JOB_ID} --spec > job.yaml
```


```bash
%%bash
cat job.yaml
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
