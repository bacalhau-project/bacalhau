---
sidebar_label: Stable-Diffusion-Keras-GPU
sidebar_position: 2
---
# Stable Diffusion Keras [GPU]

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/model-inference/gpu-keras-stable-diffusion/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=model-inference/gpu-keras-stable-diffusion/index.ipynb)

#### **Prompt**


```
Speed of Light
```



#### **Output**
![](https://i.imgur.com/cCuXiWe.jpg)

In this example we will be running stable diffusion using on a GPU using bacalhau

The source of this example is this [notebook](https://colab.research.google.com/drive/1zVTa4mLeM_w44WaFwl7utTaa6JcaH1zK?usp=sharing)

# Keras Stable Diffusion: GPU starter example

## Install GPU requirements


```bash
pip install git+https://github.com/fchollet/stable-diffusion-tensorflow --upgrade --quiet
pip install tensorflow tensorflow_addons ftfy --upgrade --quiet
pip install tqdm
apt install --allow-change-held-packages libcudnn8=8.1.0.77-1+cuda11.2
```

      Building wheel for stable-diffusion-tf (setup.py) ... [?25l[?25hdone
    [K     |████████████████████████████████| 578.0 MB 17 kB/s 
    [K     |████████████████████████████████| 1.1 MB 15.4 MB/s 
    [K     |████████████████████████████████| 53 kB 1.9 MB/s 
    [K     |████████████████████████████████| 5.9 MB 56.0 MB/s 
    [K     |████████████████████████████████| 1.7 MB 59.2 MB/s 
    [K     |████████████████████████████████| 438 kB 70.1 MB/s 
    Reading package lists... Done
    Building dependency tree       
    Reading state information... Done
    The following package was automatically installed and is no longer required:
      libnvidia-common-460
    Use 'apt autoremove' to remove it.
    The following packages will be REMOVED:
      libcudnn8-dev
    The following held packages will be changed:
      libcudnn8
    The following packages will be upgraded:
      libcudnn8
    1 upgraded, 0 newly installed, 1 to remove and 18 not upgraded.
    Need to get 430 MB of archives.
    After this operation, 3,139 MB disk space will be freed.
    Get:1 https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64  libcudnn8 8.1.0.77-1+cuda11.2 [430 MB]
    Fetched 430 MB in 13s (33.8 MB/s)
    (Reading database ... 159447 files and directories currently installed.)
    Removing libcudnn8-dev (8.0.5.39-1+cuda11.1) ...
    (Reading database ... 159425 files and directories currently installed.)
    Preparing to unpack .../libcudnn8_8.1.0.77-1+cuda11.2_amd64.deb ...
    Unpacking libcudnn8 (8.1.0.77-1+cuda11.2) over (8.0.5.39-1+cuda11.1) ...
    Setting up libcudnn8 (8.1.0.77-1+cuda11.2) ...


## Let's instantiate a Text2Image generator and make a first image

The first run has a bit of extra compilation overhead.


```python
from stable_diffusion_tf.stable_diffusion import Text2Image
from PIL import Image

generator = Text2Image( 
    img_height=512,
    img_width=512,
    jit_compile=False,  # You can try True as well (different performance profile)
)
img = generator.generate(
    "DSLR photograph of an astronaut riding a horse",
    num_steps=50,
    unconditional_guidance_scale=7.5,
    temperature=1,
    batch_size=1,
)
pil_img = Image.fromarray(img[0])
display(pil_img)
```

    Downloading data from https://github.com/openai/CLIP/blob/main/clip/bpe_simple_vocab_16e6.txt.gz?raw=true
    1356917/1356917 [==============================] - 0s 0us/step
    Downloading data from https://huggingface.co/fchollet/stable-diffusion/resolve/main/text_encoder.h5
    492456896/492456896 [==============================] - 6s 0us/step
    Downloading data from https://huggingface.co/fchollet/stable-diffusion/resolve/main/diffusion_model.h5
    3439035312/3439035312 [==============================] - 48s 0us/step
    Downloading data from https://huggingface.co/fchollet/stable-diffusion/resolve/main/decoder.h5
    198152112/198152112 [==============================] - 1s 0us/step


      0   1: 100%|██████████| 50/50 [01:10<00:00,  1.41s/it]



    
![png](index_files/index_7_2.png)
    



```bash
pip install numba
```


```python
# clearing the GPU memory 
from numba import cuda 
device = cuda.get_current_device()
device.reset()
```

    Looking in indexes: https://pypi.org/simple, https://us-python.pkg.dev/colab-wheels/public/simple/
    Requirement already satisfied: numba in /usr/local/lib/python3.7/dist-packages (0.56.2)
    Requirement already satisfied: importlib-metadata in /usr/local/lib/python3.7/dist-packages (from numba) (4.12.0)
    Requirement already satisfied: llvmlite<0.40,>=0.39.0dev0 in /usr/local/lib/python3.7/dist-packages (from numba) (0.39.1)
    Requirement already satisfied: numpy<1.24,>=1.18 in /usr/local/lib/python3.7/dist-packages (from numba) (1.21.6)
    Requirement already satisfied: setuptools<60 in /usr/local/lib/python3.7/dist-packages (from numba) (57.4.0)
    Requirement already satisfied: zipp>=0.5 in /usr/local/lib/python3.7/dist-packages (from importlib-metadata->numba) (3.8.1)
    Requirement already satisfied: typing-extensions>=3.6.4 in /usr/local/lib/python3.7/dist-packages (from importlib-metadata->numba) (4.1.1)



We will write a script that can take arguments and pass it to the stable diffusion generator

That generates images and save the outputs as images




```python
%%writefile stable-diffusion.py
import argparse
from stable_diffusion_tf.stable_diffusion import Text2Image
from PIL import Image
import os
parser = argparse.ArgumentParser(description="Stable Diffusion")
parser.add_argument("--h",dest="height", type=int,help="height of the image",default=512)
parser.add_argument("--w",dest="width", type=int,help="width of the image",default=512)
parser.add_argument("--p",dest="prompt", type=str,help="Description of the image you want to generate",default="cat")
parser.add_argument("--n",dest="numSteps", type=int,help="Number of Steps",default=50)
parser.add_argument("--u",dest="unconditionalGuidanceScale", type=float,help="Number of Steps",default=7.5)
parser.add_argument("--t",dest="temperature", type=int,help="Number of Steps",default=1)
parser.add_argument("--b",dest="batchSize", type=int,help="Number of Images",default=1)
parser.add_argument("--o",dest="output", type=str,help="Output Folder where to store the Image",default="./")

args=parser.parse_args()
height=args.height
width=args.width
prompt=args.prompt
numSteps=args.numSteps
unconditionalGuidanceScale=args.unconditionalGuidanceScale
temperature=args.temperature
batchSize=args.batchSize
output=args.output

generator = Text2Image(
    img_height=height,
    img_width=width,
    jit_compile=False,  # You can try True as well (different performance profile)
)

img = generator.generate(
    prompt,
    num_steps=numSteps,
    unconditional_guidance_scale=unconditionalGuidanceScale,
    temperature=temperature,
    batch_size=batchSize,
)
for i in range(0,batchSize):
  pil_img = Image.fromarray(img[i])
  image = pil_img.save(f"{output}/image{i}.png")

```

    Overwriting stable-diffusion.py



```bash
python stable-diffusion.py
```

    2022-09-29 15:57:32.473158: E tensorflow/stream_executor/cuda/cuda_blas.cc:2981] Unable to register cuBLAS factory: Attempting to register factory for plugin cuBLAS when one has already been registered
    2022-09-29 15:57:33.475937: W tensorflow/stream_executor/platform/default/dso_loader.cc:64] Could not load dynamic library 'libnvinfer.so.7'; dlerror: libnvinfer.so.7: cannot open shared object file: No such file or directory; LD_LIBRARY_PATH: /usr/lib64-nvidia
    2022-09-29 15:57:33.476158: W tensorflow/stream_executor/platform/default/dso_loader.cc:64] Could not load dynamic library 'libnvinfer_plugin.so.7'; dlerror: libnvinfer_plugin.so.7: cannot open shared object file: No such file or directory; LD_LIBRARY_PATH: /usr/lib64-nvidia
    2022-09-29 15:57:33.476182: W tensorflow/compiler/tf2tensorrt/utils/py_utils.cc:38] TF-TRT Warning: Cannot dlopen some TensorRT libraries. If you would like to use Nvidia GPU with TensorRT, please make sure the missing libraries mentioned above are installed properly.
    2022-09-29 15:57:37.010234: W tensorflow/core/common_runtime/gpu/gpu_bfc_allocator.cc:42] Overriding orig_value setting because the TF_FORCE_GPU_ALLOW_GROWTH environment variable is set. Original config value was 0.
     49 981:   0% 0/50 [00:00<?, ?it/s]2022-09-29 15:58:19.893237: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.48GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:19.893362: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.48GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.270785: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.69GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.270859: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.69GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.386111: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.386176: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.493172: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.493253: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.581118: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
    2022-09-29 15:58:20.581185: W tensorflow/core/common_runtime/bfc_allocator.cc:290] Allocator (GPU_0_bfc) ran out of memory trying to allocate 6.67GiB with freed_by_count=0. The caller indicates that this is not a failure, but this may mean that there could be performance gains if more memory were available.
      0   1: 100% 50/50 [01:05<00:00,  1.30s/it]


Viewing the outputted image


```python
import IPython.display as display
display.Image("image0.png")
```




    
![png](index_files/index_14_0.png)
    




```
optional arguments:
  -h, --help            show this help message and exit
  --h HEIGHT            height of the image
  --w WIDTH             width of the image
  --p PROMPT            Description of the image you want to generate
  --n NUMSTEPS          Number of Steps
  --u UNCONDITIONALGUIDANCESCALE
                        UNCONDITIONALGUIDANCESCALE
  --t TEMPERATURE       Temparature
  --b BATCHSIZE         Number of Images to generate
  --o OUTPUT            Output Folder where to store the Image
  ```



### Running the script with arguments

#### Prompt
```python stable-diffusion.py --p "cat with three eyes"```

#### Number of iterations
```
python stable-diffusion.py --p "cat with three eyes" --n 100
```
#### Batch Size (No of images to generate)
```
python stable-diffusion.py --p "cat with three eyes" --b 2
```



After that we will write a DOCKERFILE to containernize this script and then run it on bacalhau


## **Building and Running on docker**

In this step you will create a  `Dockerfile` to create your Docker deployment. The `Dockerfile` is a text document that contains the commands used to assemble the image.

First, create the `Dockerfile`.

Next, add your desired configuration to the `Dockerfile`. These commands specify how the image will be built, and what extra requirements will be included.

Dockerfile


```
FROM tensorflow/tensorflow:latest-gpu

RUN apt-get -y update

RUN apt-get -y install git

RUN python3 -m pip install --upgrade pip

RUN python -m pip install regex tqdm Pillow

RUN pip install git+https://github.com/fchollet/stable-diffusion-tensorflow --upgrade --quiet

RUN pip install tensorflow tensorflow_addons ftfy --upgrade --quiet

RUN apt install --allow-change-held-packages libcudnn8=8.1.0.77-1+cuda11.2

ADD stable-diffusion.py stable-diffusion.py

RUN python stable-diffusion.py --n 5
```


In the dockerfile we will be using the tensorflow GPU image and then installing dependencies like git and other python python 

To Build the docker container run the docker build command


```
docker build -t <hub-user>/<repo-name>:<tag> .
```


Please replace

&lt;hub-user> with your docker hub username, If you don’t have a docker hub account [Follow these instructions to create docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

&lt;repo-name> This is the name of the container, you can name it anything you want

&lt;tag> This is not required but you can use the latest tag

After you have build the container, the next step is to test it locally and then push it docker hub

Now you can push this repository to the registry designated by its name or tag.


```
 docker push <hub-user>/<repo-name>:<tag>
```


After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

## **Running the container on bacalhau**

After the repo image has been pushed to docker hub, we can now use the container for running on bacalhau

We use the --gpu flag to denote the no of GPU we are going to use


```
bacalhau docker run \
--gpu 1 \
jsacex/stable-diffusion-keras \
-- python stable-diffusion.py --o ./outputs
```


Insalling bacalhau


```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    
    BACALHAU CLI is detected:
    Client Version: v0.2.5
    Server Version: v0.2.5
    Reinstalling BACALHAU CLI - /usr/local/bin/bacalhau...
    Getting the latest BACALHAU CLI...
    Installing v0.2.5 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.5/bacalhau_v0.2.5_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.5
    Server Version: v0.2.5



```bash
echo $(bacalhau docker run --id-only --wait --wait-timeout-secs 1000 --gpu 1 jsacex/stable-diffusion-keras -- python stable-diffusion.py --o ./outputs) > job_id.txt
cat job_id.txt
```

    4f758052-0543-40b5-bd86-6ab41e77389a



```bash
bacalhau list --id-filter $(cat job_id.txt)
```

    [92;100m CREATED  [0m[92;100m ID       [0m[92;100m JOB                     [0m[92;100m STATE     [0m[92;100m VERIFIED [0m[92;100m PUBLISHED               [0m
    [97;40m 17:33:46 [0m[97;40m 4f758052 [0m[97;40m Docker jsacex/stable... [0m[97;40m Completed [0m[97;40m          [0m[97;40m /ipfs/QmcQEQPg934Pow... [0m



Where it says "`Completed `", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe $(cat job_id.txt)
```

Since there is no error we can’t see any error instead we see the state of our job to be complete, that means 
we can download the results!
we create a temporary directory to save our results


```bash
mkdir results
```

To Download the results of your job, run 

---

the following command:


```bash
bacalhau get  $(cat job_id.txt)  --output-dir results
```

    [90m17:38:25.343 |[0m [32mINF[0m [1mbacalhau/get.go:67[0m[36m >[0m Fetching results of job '4f758052-0543-40b5-bd86-6ab41e77389a'...
    2022/09/29 17:38:25 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
    [90m17:38:35.851 |[0m [32mINF[0m [1mipfs/downloader.go:115[0m[36m >[0m Found 1 result shards, downloading to temporary folder.
    [90m17:38:37.1 |[0m [32mINF[0m [1mipfs/downloader.go:195[0m[36m >[0m Combining shard from output volume 'outputs' to final location: '/content/results'


After the download has finished you should 
see the following contents in results directory


```bash
ls results/
```

    shards	stderr	stdout	volumes



By Inspecting the Downloaded Results

We can find that our generated image is located in /volumes/outputs/mars.png


```
.
├── shards
│   └── job-2c281c1b-1a3e-4863-830f-8c48d117f6ea-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
│       ├── exitCode
│       ├── stderr
│       └── stdout
├── stderr
├── stdout
└── volumes
    └── outputs
        └── cat.png
```


```python
import IPython.display as display
display.Image("results/volumes/outputs/image0.png")
```




    
![png](index_files/index_32_0.png)
    




```bash
bacalhau describe $(cat job_id.txt) --spec > job.yaml
```


```bash
cat job.yaml
```
