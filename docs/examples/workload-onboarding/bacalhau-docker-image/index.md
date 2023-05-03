---
sidebar_label: "Bacalhau Docker Image"
sidebar_position: 1
description: How to use the Bacalhau Docker image
---
# Bacalhau Docker Image


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

This example shows you how to run some common client-side Bacalhau tasks using the Bacalhau Docker image.

## TD;LR
Running Docker image on Bacalhau

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)

## Pull the Docker image

The first step is to pull the Bacalhau Docker image from the [Github container registry](https://github.com/orgs/bacalhau-project/packages/container/package/bacalhau).


```bash
%%bash
docker pull ghcr.io/bacalhau-project/bacalhau:latest
```

    latest: Pulling from bacalhau-project/bacalhau
    Digest: sha256:d80f1fe751886a29e0d5ae265a88abbfcd5c59e36247473b66aba93ea24f45aa
    Status: Image is up to date for ghcr.io/bacalhau-project/bacalhau:latest
    ghcr.io/bacalhau-project/bacalhau:latest


You can also pull a specific version of the image, e.g.:

```bash
docker pull ghcr.io/bacalhau-project/bacalhau:v0.3.16
```

:::warning
Remember that the "latest" tag is just a string. It doesn't refer to the latest version of the Bacalhau client, it refers to an image that has the "latest" tag. Therefore, if your machine has already downloaded the "latest" image, it won't download it again. To force a download, you can use the `--no-cache` flag.
:::

## Check version

Check the version of the Bacalhau client you are using.



```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest version
```

    Client Version: v0.3.29
    Server Version: v0.3.29


## Running a Bacalhau Job

To submit a bi to Bacalhau, we use the `bacalhau docker run` command. 


```bash
%%bash --out job_id
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        ubuntu:latest -- \
            sh -c 'uname -a && echo "Hello from Docker Bacalhau!"'
```

In this example, I run an ubuntu-based job that echo's some stuff.

### Structure of the command

-  `--id-only......`: Output only the job id

- `ubuntu:latest.` Ubuntu container

- `ghcr.io/bacalhau-project/bacalhau:latest `: Name of the Bacalhau Docker image

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

    env: JOB_ID=a00cadd2-0214-4d57-9eee-363c90cdecb8


To print out the content of the Job ID, run the following command:


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \
        | grep -A 2 "stdout: |"
```

          stdout: |
            Linux c32ddafa1967 5.19.0-1022-gcp #24~22.04.1-Ubuntu SMP Sun Apr 23 09:51:08 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
            Hello from Docker Bacalhau!


## Sumbit a Job With Output Files

One inconvenience that you'll see is that you'll need to mount directories into the container to access files. This is because the container is running in a separate environment to your host machine. Let's take a look at the example below:

The first part of the example should look familiar, except for the Docker commands.


```bash
%%bash --out job_id
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    docker run \
        --id-only \
        --wait \
        --gpu 1 \
        ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 -- \
            python main.py --o ./outputs --p "A Docker whale and a cod having a conversation about the state of the ocean"
```


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.

    env: JOB_ID=30c62ab4-5a77-4ea0-ad67-3cc9ef7387d2


## Checking the State of your Jobs

- **Job status**: You can check the status of the job using `bacalhau list`. 


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    list $JOB_ID \
        | grep -A 2 "stdout: |"
```


    ---------------------------------------------------------------------------

    CalledProcessError                        Traceback (most recent call last)

    Cell In[21], line 1
    ----> 1 get_ipython().run_cell_magic('bash', '', 'bacalhau docker run -t ghcr.io/bacalhau-project/bacalhau:latest \\\n    list $JOB_ID \\\n        | grep -A 2 "stdout: |"\n')


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/interactiveshell.py:2430, in InteractiveShell.run_cell_magic(self, magic_name, line, cell)
       2428 with self.builtin_trap:
       2429     args = (magic_arg_s, cell)
    -> 2430     result = fn(*args, **kwargs)
       2432 # The code below prevents the output from being displayed
       2433 # when using magics with decodator @output_can_be_silenced
       2434 # when the last Python token in the expression is a ';'.
       2435 if getattr(fn, magic.MAGIC_OUTPUT_CAN_BE_SILENCED, False):


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:153, in ScriptMagics._make_script_magic.<locals>.named_script_magic(line, cell)
        151 else:
        152     line = script
    --> 153 return self.shebang(line, cell)


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:305, in ScriptMagics.shebang(self, line, cell)
        300 if args.raise_error and p.returncode != 0:
        301     # If we get here and p.returncode is still None, we must have
        302     # killed it but not yet seen its return code. We don't wait for it,
        303     # in case it's stuck in uninterruptible sleep. -9 = SIGKILL
        304     rc = p.returncode or -9
    --> 305     raise CalledProcessError(rc, cell)


    CalledProcessError: Command 'b'bacalhau docker run -t ghcr.io/bacalhau-project/bacalhau:latest \\\n    list $JOB_ID \\\n        | grep -A 2 "stdout: |"\n'' returned non-zero exit status 1.


When it says `Completed`, that means the job is done, and we can get the results.

- **Job information**: You can find out more information about your job by using `bacalhau describe`.


```bash
%%bash
docker run -t ghcr.io/bacalhau-project/bacalhau:latest \
    describe $JOB_ID \
        | grep -A 2 "stdout: |"
```


    ---------------------------------------------------------------------------

    CalledProcessError                        Traceback (most recent call last)

    Cell In[10], line 1
    ----> 1 get_ipython().run_cell_magic('bash', '', 'docker run -t ghcr.io/bacalhau-project/bacalhau:latest \\\n    describe $JOB_ID \\\n        | grep -A 2 "stdout: |"\n')


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/interactiveshell.py:2430, in InteractiveShell.run_cell_magic(self, magic_name, line, cell)
       2428 with self.builtin_trap:
       2429     args = (magic_arg_s, cell)
    -> 2430     result = fn(*args, **kwargs)
       2432 # The code below prevents the output from being displayed
       2433 # when using magics with decodator @output_can_be_silenced
       2434 # when the last Python token in the expression is a ';'.
       2435 if getattr(fn, magic.MAGIC_OUTPUT_CAN_BE_SILENCED, False):


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:153, in ScriptMagics._make_script_magic.<locals>.named_script_magic(line, cell)
        151 else:
        152     line = script
    --> 153 return self.shebang(line, cell)


    File ~/.pyenv/versions/3.11.1/lib/python3.11/site-packages/IPython/core/magics/script.py:305, in ScriptMagics.shebang(self, line, cell)
        300 if args.raise_error and p.returncode != 0:
        301     # If we get here and p.returncode is still None, we must have
        302     # killed it but not yet seen its return code. We don't wait for it,
        303     # in case it's stuck in uninterruptible sleep. -9 = SIGKILL
        304     rc = p.returncode or -9
    --> 305     raise CalledProcessError(rc, cell)


    CalledProcessError: Command 'b'docker run -t ghcr.io/bacalhau-project/bacalhau:latest \\\n    describe $JOB_ID \\\n        | grep -A 2 "stdout: |"\n'' returned non-zero exit status 1.


- **Job download**: You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory and downloaded our job output to be stored in that directory.


```bash
%%bash
docker run -t -v $(pwd)/results:/results ghcr.io/bacalhau-project/bacalhau:latest \
    get $JOB_ID --output-dir /results
```

After the download has finished you should see the following contents in results directory. 




    
![png](index_files/index_24_0.png)
    



## Need Support?

If have questions or need support or guidance, please reach out to the [Bacalhau team via Slack (#bacalhau channel)](https://filecoin.io/slack)

