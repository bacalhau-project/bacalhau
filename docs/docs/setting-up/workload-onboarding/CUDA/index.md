---
sidebar_label: "CUDA"
sidebar_position: 10
# cspell: ignore nvcc, cudnn, Bacalhau, timeit, expt, memcpy, devel
---
# Run CUDA programs on Bacalhau


### What is CUDA

In this tutorial, we will look at how to run CUDA programs on Bacalhau. CUDA (Compute Unified Device Architecture) is an extension of C/C++ programming. It is a parallel computing platform and programming model created by NVIDIA. It helps developers speed up their applications by harnessing the power of GPU accelerators.

In addition to accelerating high-performance computing (HPC) and research applications, CUDA has also been widely adopted across consumer and industrial ecosystems. CUDA also makes it easy for developers to take advantage of all the latest GPU architecture innovations

### Advantage of GPU over CPU
Architecturally, the CPU is composed of just a few cores with lots of cache memory that can handle a few software threads at a time. In contrast, a GPU is composed of hundreds of cores that can handle thousands of threads simultaneously.

Computations like matrix multiplication could be done much faster on GPU than on CPU

### Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## 1. Running CUDA locally

You'll need to have the following installed:
1. NVIDIA GPU
2. CUDA drivers installed
3. nvcc installed

Checking if nvcc is installed:


```python
!nvcc --version
```

Downloading the programs:


```bash
%%bash
mkdir inputs outputs
wget -P inputs https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/00-hello-world.cu
wget -P inputs https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu
```

### Viewing the programs

1. **`00-hello-world.cu`**:

```bash
# View the contents of the standard C++ program
!cat inputs/00-hello-world.cu

# Measure the time it takes to compile and run the program
%%timeit
!nvcc -o ./outputs/hello ./inputs/00-hello-world.cu; ./outputs/hello
```
This example represents a standard C++ program that inefficiently utilizes GPU resources due to the use of non-parallel loops.


2. **`02-cuda-hello-world-faster.cu`**:

```bash
# View the contents of the CUDA program with vector addition
!cat inputs/02-cuda-hello-world-faster.cu

# Remove any previous output
!rm -rf outputs/hello

# Measure the time for compilation and execution
%%timeit
!nvcc --expt-relaxed-constexpr -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello
```

In this example we utilize Vector addition using CUDA and allocate the memory in advance and copy the memory to the GPU using cudaMemcpy so that it can utilize the HBM (High Bandwidth memory of the GPU).
Compilation and execution occur faster (1.39 seconds) compared to the previous example (8.67 seconds).

## 2. Running a Bacalhau Job

To submit a job, run the following Bacalhau command:


```bash
%%bash --out job_id
bacalhau docker run \
    --gpu 1 \
    --timeout 3600 \
    --wait-timeout-secs 3600 \
    -i https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu \
    --id-only \
    --wait \
    nvidia/cuda:11.2.2-cudnn8-devel-ubuntu18.04 \
    -- /bin/bash -c 'nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello '
```

### Structure of the Commands

`bacalhau docker run`: call to Bacalhau

`-i https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu`: URL path of the input data volumes downloaded from a URL source.

`nvidia/cuda:11.2.0-cudnn8-devel-ubuntu18.04`: Docker container for executing CUDA programs (you need to choose the right CUDA docker container). The container should have the tag of "devel" in them.

`nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu`: Compilation using the nvcc compiler and save it to the outputs directory as hello

Note that there is `;` between the commands:
  `-- /bin/bash -c 'nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello ` The ";" symbol allows executing multiple commands sequentially in a single line.

`./outputs/hello`: Execution hello binary:
You can combine compilation and execution commands.

:::info
Note that the CUDA version will need to be compatible with the graphics card on the host machine.
:::

When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on:

```python
%env JOB_ID={job_id}
```

## 3. Checking the State of your Jobs

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

## 4. Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/stdout
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
