---
sidebar_label: "CUDA"
sidebar_position: 10
---
# Run CUDA programs on bacalhau

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/CUDA/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/CUDA/index.ipynb)

## Introduction

### What is CUDA

CUDA stands for Compute Unified Device Architecture. It is an extension of C/C++ programming.

CUDA is a parallel computing platform and programming model created by NVIDIA.
it helps developers speed up their applications by harnessing the power of GPU accelerators.

In addition to accelerating high performance computing (HPC) and research applications, CUDA has also been widely adopted across consumer and industrial ecosystems.


CUDA also makes it easy for developers to take advantage of all the latest GPU architecture innovations

### Advantage of GPU over CPU
Architecturally, the CPU is composed of just a few cores with lots of cache memory that can handle a few software threads at a time. In contrast, a GPU is composed of hundreds of cores that can handle thousands of threads simultaneously.

Computations like matrix multiplication could be done much faster on GPU than on CPU



## Running locally


Prerequisites
- NVIDIA GPU
- CUDA drivers installed
- nvcc installed

checking if nvcc is installed


```python
!nvcc --version
```

    nvcc: NVIDIA (R) Cuda compiler driver
    Copyright (c) 2005-2021 NVIDIA Corporation
    Built on Sun_Feb_14_21:12:58_PST_2021
    Cuda compilation tools, release 11.2, V11.2.152
    Build cuda_11.2.r11.2/compiler.29618528_0


Downloading the programs


```bash
%%bash
mkdir inputs outputs
wget -P inputs https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/00-hello-world.cu
wget -P inputs https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu
```

    --2022-11-14 10:12:12--  https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/00-hello-world.cu
    Resolving raw.githubusercontent.com (raw.githubusercontent.com)... 185.199.110.133, 185.199.108.133, 185.199.111.133, ...
    Connecting to raw.githubusercontent.com (raw.githubusercontent.com)|185.199.110.133|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 517 [text/plain]
    Saving to: ‘inputs/00-hello-world.cu’
    
         0K                                                       100% 21.5M=0s
    
    2022-11-14 10:12:12 (21.5 MB/s) - ‘inputs/00-hello-world.cu’ saved [517/517]
    
    --2022-11-14 10:12:12--  https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu
    Resolving raw.githubusercontent.com (raw.githubusercontent.com)... 185.199.109.133, 185.199.108.133, 185.199.110.133, ...
    Connecting to raw.githubusercontent.com (raw.githubusercontent.com)|185.199.109.133|:443... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 1231 (1.2K) [text/plain]
    Saving to: ‘inputs/02-cuda-hello-world-faster.cu’
    
         0K .                                                     100% 49.1M=0s
    
    2022-11-14 10:12:12 (49.1 MB/s) - ‘inputs/02-cuda-hello-world-faster.cu’ saved [1231/1231]
    


### Viewing the programs


```bash
%%bash
cat inputs/00-hello-world.cu
```

    #include <cmath>
    #include <iostream>
    #include <vector>
    
    int main()
    {
        size_t n = 50000000;
        std::vector<double> a(n);
        std::vector<double> b(n);
        for (int i = 0; i < n; i++) {
            a[i] = sin(i) * sin(i);
            b[i] = cos(i) * cos(i);
        }
    
        std::vector<double> c(n);
        for (int i = 0; i < n; i++) {
            c[i] = a[i] + b[i];
        }
    
        double sum = 0;
        for (int i = 0; i < n; i++) {
            sum += c[i];
        }
    
        std::cout << "final result " << (sum / n) << std::endl;
    
        return 0;
    }


This is a standard c++ program which uses loops which are not parallizable so it dosen't use the most of the processing power of the GPU


```python
%%timeit
!nvcc -o ./outputs/hello ./inputs/00-hello-world.cu; ./outputs/hello
```

    final result 1
    final result 1
    final result 1
    final result 1
    final result 1
    final result 1
    final result 1
    final result 1
    8.6 s ± 72.6 ms per loop (mean ± std. dev. of 7 runs, 1 loop each)



```python
!cat inputs/02-cuda-hello-world-faster.cu
```

    #include <math.h>
    #include <stdio.h>
    #include <stdlib.h>
    
    __global__ void prepData(double *a, double *b, size_t n)
    {
        const int idx = blockIdx.x * blockDim.x + threadIdx.x;
        if (idx < n) {
            a[idx] = sin(idx) * sin(idx);
            b[idx] = cos(idx) * cos(idx);
        }
    }
    
    __global__ void vecAdd(double *a, double *b, double *c, size_t n)
    {
        const int idx = blockIdx.x * blockDim.x + threadIdx.x;
        if (idx < n) {
            c[idx] = a[idx] + b[idx];
        }
    }
    
    int main()
    {
        size_t n = 50000000;
        size_t bytes = n * sizeof(double);
        double *h_c = (double *) malloc(bytes);  // output vector
    
        double *d_a, *d_b, *d_c;
        cudaMalloc(&d_a, bytes);
        cudaMalloc(&d_b, bytes);
        cudaMalloc(&d_c, bytes);
    
        const int blockSize = 1024;
        const int gridSize = (int)ceil((float)n/blockSize);
    
        prepData<<<gridSize, blockSize>>>(d_a, d_b, n);
    
        cudaDeviceSynchronize();
    
        vecAdd<<<gridSize, blockSize>>>(d_a, d_b, d_c, n);
    
        cudaMemcpy(h_c, d_c, bytes, cudaMemcpyDeviceToHost);
    
        double sum = 0;
        for (int i = 0; i < n; i++) {
            sum += h_c[i];
        }
    
        printf("final result: %f\n", sum / n);
    
        cudaFree(d_a);
        cudaFree(d_b);
        cudaFree(d_c);
    
        free(h_c);
    
        return 0;
    }
    


Instead of looping we use Vector addition using CUDA and allocate the memory in advance and copy the memory to the GPU
using cudaMemcpy so that it can utilize the HBM (High Bandwith memory of the GPU)


```python
!rm -rf outputs/hello
```


```python
%%timeit
!nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello
```

    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    final result: 1.000000
    1.48 s ± 46.6 ms per loop (mean ± std. dev. of 7 runs, 1 loop each)


It takes around 8.67s to run 
00-hello-world.cu
while it takes 1.39s to run
02-cuda-hello-world-faster.cu


## Running on bacalhau

Installing bacalhau


```bash
%%bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.11 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.11/bacalhau_v0.3.11_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.3.11
    Server Version: v0.3.11


You can easily execute the same program we ran locally using bacalhau

* The program is mounted by using the `-u` flag you can specify the link there
`-u < Link-To-The-Program >`


* Docker container `nvidia/cuda:11.2.0-cudnn8-devel-ubuntu18.04`
for executing CUDA programs you need to choose the right CUDA docker container the container should have the tag of devel in them

* Running program consists of two parts:
  * Compilation using the nvcc compiler and save it to the outputs directory as hello: `nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu`
  * Execution hello binary:  `./outputs/hello`
  * You can combine compilation and execution commands. Note that there is `;` between the commands:
`-- /bin/bash -c 'nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello `



```bash
%%bash --out job_id
 bacalhau docker run \
--gpu 1 \
--timeout 3600 \
--wait-timeout-secs 3600 \
 -u https://raw.githubusercontent.com/tristanpenman/cuda-examples/master/02-cuda-hello-world-faster.cu \
 --id-only \
 --wait \
 nvidia/cuda:11.2.0-cudnn8-devel-ubuntu18.04 \
-- /bin/bash -c 'nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello '
```


```bash
%%bash
bacalhau list --id-filter ${JOB_ID} --wide
```

    [92;100m CREATED           [0m[92;100m ID                                   [0m[92;100m JOB                                                                                                                                                                        [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED                                            [0m
    [97;40m 22-11-14-12:31:45 [0m[97;40m 22715ef6-759e-488a-9aa3-aaf2c8a79b08 [0m[97;40m Docker nvidia/cuda:11.2.0-cudnn8-devel-ubuntu18.04 /bin/bash -c nvcc --expt-relaxed-constexpr  -o ./outputs/hello ./inputs/02-cuda-hello-world-faster.cu; ./outputs/hello  [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmSFnLwaCdoVGpyfjFZDpQ72AS5hTgzzCKmfznnVrH8SgH [0m


Where it says "Completed", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:



```bash
%%bash
bacalhau describe ${JOB_ID}
```


```bash
%%bash
rm -rf results && mkdir -p results
bacalhau get $JOB_ID --output-dir results
```

    Fetching results of job '22715ef6-759e-488a-9aa3-aaf2c8a79b08'...
    Results for job '22715ef6-759e-488a-9aa3-aaf2c8a79b08' have been written to...
    results


    2022/11/14 12:32:02 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.


Viewing the outputs


```bash
%%bash
cat results/combined_results/stdout
```

    final result: 1.000000

