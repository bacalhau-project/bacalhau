---
sidebar_label: YOLO-Object-Detection
sidebar_position: 3
---
# YOLOv5 (Object detection) on bacalhau


[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/model-inference/object-detection-yolo5/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=model-inference/object-detection-yolo5/index.ipynb)

## **Introduction**



Identification and localization of objects in photos is a computer vision task called ‘object detection, several algorithms have emerged in the past few years to tackle the problem. One of the most popular algorithms to date for real-time object detection is [YOLO (You Only Look Once)](https://towardsdatascience.com/yolo-you-only-look-once-real-time-object-detection-explained-492dc9230006), initially proposed by Redmond et. al [[1]](https://arxiv.org/abs/1506.02640).

Unfortunately, many of these models require enormous amounts of training materials to get high-quality results out of the model. For many organizations looking to run object detection, they may not have access to well-labeled training data, limiting the utility of these models. However, the advent of sharing pre-trained models to do inference, without training the model, offers the best of both worlds: better results with less or no training.

In this tutorial you will perform an end-to-end object detection inference,

using the [YOLOv5 Docker Image developed by Ultralytics.](https://github.com/ultralytics/yolov5/wiki/Docker-Quickstart)




Identification and localization of objects in photos is a computer vision task called ‘object detection, several algorithms have emerged in the past few years to tackle the problem. One of the most popular algorithms to date for real-time object detection is [YOLO (You Only Look Once)](https://towardsdatascience.com/yolo-you-only-look-once-real-time-object-detection-explained-492dc9230006), initially proposed by Redmond et. al [[1]](https://arxiv.org/abs/1506.02640).

Unfortunately, many of these models require enormous amounts of training materials to get high-quality results out of the model. For many organizations looking to run object detection, they may not have access to well-labeled training data, limiting the utility of these models. However, the advent of sharing pre-trained models to do inference, without training the model, offers the best of both worlds: better results with less or no training.

In this tutorial you will perform an end-to-end object detection inference,

using the [YOLOv5 Docker Image developed by Ultralytics.](https://github.com/ultralytics/yolov5/wiki/Docker-Quickstart)




## **Advantages of using bacalhau**

Using Bacalhau you can do the Object Detection inference on GPUs, and you don’t need to have all the images stored on your local machine they can be stored on IPFS 

Since the outputs are stored on IPFS you don’t need to download them on your local machine



Insalling bacalhau


```bash
curl -sL https://get.bacalhau.org/install.sh | bash
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.2.3 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.2.3/bacalhau_v0.2.3_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into /usr/local/bin successfully.
    Client Version: v0.2.3
    Server Version: v0.2.3



## **Running the job**

To get started we run object detection on images already present inside the container

If you want to use your custom images as an input please refer [Using custom Images as an input](#Uploading-Images-to-IPFS)


Command:

```
 bacalhau docker run \
--gpu 1   \
-u https://github.com/ultralytics/yolov5/releases/download/v6.2/yolov5s.pt:/usr/src/app/yolov5s.pt \
ultralytics/yolov5:latest \
-- /bin/bash -c 'python detect.py --weights yolov5s.pt --source $(pwd)/data/images --project ../../../outputs'
```

Structure of the command:



* Specify the command` bacalhau docker run` which is equivalent to docker run
* --gpu 1 specify the number of GPUs
* `-u` you can select the weights that you want from here [yolov5 release page](https://github.com/ultralytics/yolov5/releases)

the model requires weights for it to run, so it downloads the weights from github but since bacalhau doesn’t have networking enabled, you need to mount the weights and mount them to the pwd which in this case is /usr/src/app, so we specify the mount path /usr/src/app/yolov5s.pt


You can also provide your own weights, 

* `ultralytics/yolov5:latest` specify the container image you want to use
* `-- /bin/bash -c` here we specify the command we want to execute

we will run the script detect.py which is for object detection


Specify the path to the weights and source of the images 


`--project` here specify the output volume which you want to save to bacalhau   mounts an output volume called ‘outputs’ so we save the outputs there, for more flags refer [yolov5/detect.py at master](https://github.com/ultralytics/yolov5/blob/master/detect.py#L3-#L25) 



```bash
bacalhau docker run \
--gpu 1 \
--wait \
--wait-timeout-secs 1000 \
--id-only \
-u https://github.com/ultralytics/yolov5/releases/download/v6.2/yolov5s.pt \
ultralytics/yolov5:latest \
-- /bin/bash -c 'python detect.py --weights ../../../inputs/yolov5s.pt --source $(pwd)/data/images --project ../../../outputs'
```


```python
%env JOB_ID={job_id}
```


This should output a UUID (like `59c59bfb-4ef8-45ac-9f4b-f0e9afd26e70`). This is the ID of the job that was created. You can check the status of the job with the following command:



```bash
bacalhau list --id-filter ${JOB_ID}
```


Where it says "`Completed`", that means the job is done, and we can get the results.

To find out more information about your job, run the following command:


```bash
bacalhau describe ${JOB_ID}
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
bacalhau get  ${JOB_ID} --output-dir results
```

After the download has finished you should 
see the following contents in results directory


```bash
ls results/
```

    shards	stderr	stdout	volumes




The structure of the files and directories will look like this:


```
.
├── shards
│   └── job-59c59bfb-4ef8-45ac-9f4b-f0e9afd26e70-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
│   ├── exitCode
│   ├── stderr
│   └── stdout
├── stderr
├── stdout
└── volumes
└── outputs
├── bus.jpg
└── zidane.jpg
```


The Folder were our labelled images are stored is /volumes/outputs


the outputs of your job will be downloaded in volumes/outputs/


Viewing the results image bus.jpg
![](https://i.imgur.com/0Zk3zNz.jpg)






# Using custom Images as an input

To run Object Detection with your own images firstly you need to already have Images stored on IPFS/Filecoin or upload the Images to IPFS/FIlecoin

In this command we are using the [Cyclist Dataset for Object Detection | Kaggle](https://www.kaggle.com/datasets/f445f341fc5e3ab58757efa983a38d6dc709de82abd1444c8817785ecd42a1ac) dataset 

The weights can be downloaded from this link [YOLOv5s](https://github.com/ultralytics/yolov5/releases/download/v6.2/yolov5s.pt) or you can choose your own weights from [yolov5 release page](https://github.com/ultralytics/yolov5/releases) or even upload your own custom weights


### **Uploading Images to IPFS**

To test whether if our script works we upload 10 Images of the whole dataset along with weights

The directory structure of our dataset should look like


```
datasets
├── 008710.jpg
├── 008711.jpg
├── 008712.jpg
├── 008713.jpg
├── 008714.jpg
├── 008715.jpg
├── 008716.jpg
├── 008717.jpg
├── 008718.jpg
├── 008719.jpg
└── yolov5s.pt
```


Uploading datasets directory to IPFS using the ipfs cli (Not recommended)

we run the following command for that


```
$ ipfs add -r images-10/
added QmaoAngi85Rr3na1xSUdbC4F9Qv3CsT75KZUs7mVfdqQRX images-10/008710.jpg
added QmbTP6J9eAXpDvxLopH6GmNmgjR7WsJHdvhdQQ4iKmZdXh images-10/008711.jpg
added QmedfJnxJkZp2wQ5XNr7fbo4T7upr2eNTSJSrVceMU7JtY images-10/008712.jpg
added QmYHsXG1aR46PCPtgjZ1zQuFs3Mv8sFDcgFg73LhrHUKnf images-10/008713.jpg
added QmawStm5oDn4VPKsn9CAorAzJRfZKqsJoTe4n5TyosifAq images-10/008714.jpg
added QmU3Qim3o7BhTsUee68EyzxrTDjJzgQUUTgBZ6SQfuSBgL images-10/008715.jpg
added Qmbyhz2HTgiwo4dA5w4p2GcksTtxVmpcfGzcTCGz4mBBce images-10/008716.jpg
added QmQqimRW8Ng1z2dsMsKH1KBHy637d2H88QAzBXecavwT4a images-10/008717.jpg
added QmYsdTXKYsSriVG9a5Khv3qyt5oZog8Jdc82tvNunyybsa images-10/008718.jpg
added QmXHpTMvxARbEufWPDxu6xfSAC2QZGqW6xx492xCsL5Vob images-10/008719.jpg
added QmTv8e1W4q19CvX46fBxeit3SqaSB4ERcjcJUR4UnHyDoX images-10/yolov5s.pt
added QmVkqsJySdytkY75zdQGNQHqJc4naXtMDAVTD5gwZShAmd images-10
 15.77 MiB / 15.77 MiB [=======================================================================================] 100.00%
```


Since the data Uploaded To IPFS isn’t pinned or will be garbage collected

The Data needs to be Pinned, Pinning is the mechanism that allows you to tell IPFS to always keep a given object somewhere, the default being your local node, though this can be different if you use a third-party remote pinning service.

There a different pinning services available you can you any one of them


## [Pinata](https://app.pinata.cloud/)

Click on the upload folder button

![](https://i.imgur.com/crnkrwy.png)

After the Upload has finished copy the CID

![](https://i.imgur.com/2Zs884R.png)


### [NFT.Storage](https://nft.storage/) (Recommneded Option)

[Upload files and directories with NFTUp](https://nft.storage/docs/how-to/nftup/) 

To upload your dataset using NFTup just drag and drop your directory it will upload it to IPFS

![](https://i.imgur.com/g3VM2Kp.png)


Copy the CID in this case it is bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm

You can view you uploaded dataset by clicking on the Gateway URL

[https://bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm.ipfs.nftstorage.link/](https://bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm.ipfs.nftstorage.link/)


### **Running the command**

What the -v flag does

-v flag is used to mount your IPFS CIDs to the container

So if you want to mount your own CID 

-v &lt;THE-CID-YOU-COPIED>:/&lt;PATH-OF-DIRECTORY-IN-WHICH-YOU-WANT-TO-MOUNT-THE-DATASET>

In this case it will look like where we mount the CID to /datasets folder

-v bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm:/datasets


```
bacalhau docker run \
--gpu 1 \
-v bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm:/datasets \
ultralytics/yolov5:latest \
-- /bin/bash -c 'python detect.py --weights ../../../datasets/yolov5s.pt --source ../../../datasets --project  ../../../outputs'
```



```bash
bacalhau docker run \
--gpu 1 \
--wait \
--wait-timeout-secs 1000 \
--id-only \
-v bafybeicyuddgg4iliqzkx57twgshjluo2jtmlovovlx5lmgp5uoh3zrvpm:/datasets \
ultralytics/yolov5:latest \
-- python detect.py --weights ../../../datasets/yolov5s.pt --source ../../../datasets --project  ../../../outputs
```

    dbdf569f-7eec-4acd-b469-bd0a1a8005da



```python
%env JOB_ID={job_id}
```


This should output a UUID (like `1f113734-cb05-4331-b049-b9b5b102259a` ). This is the ID of the job that was created. You can check the status of the job with the following command:


```bash
bacalhau list --id-filter ${JOB_ID}
```


To Download the results of your job, run the following command:

we create a temporary directory to save our results


```bash
mkdir custom-results
```


```bash
bacalhau get ${JOB_ID} --output-dir custom-results
```


The structure of the files and directories will look like this:


```
├── shards
│   └── job-1f113734-cb05-4331-b049-b9b5b102259a-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL
│   ├── exitCode
│   ├── stderr
│   └── stdout
├── stderr
├── stdout
└── volumes
└── outputs
├── 008710.jpg
├── 008711.jpg
├── 008712.jpg
├── 008713.jpg
├── 008714.jpg
├── 008715.jpg
├── 008716.jpg
├── 008717.jpg
├── 008718.jpg
└── 008719.jpg
```


The labeled images are at volumes/outputs

If you don’t want to download the results but still view them using a IPFS gateway 

[https://cloudflare-ipfs.com/ipfs/bafybeiai3v7svitueeipqcohvelpxpcex5jtuep4wnezfqjknphuewwkoq](https://cloudflare-ipfs.com/ipfs/bafybeiai3v7svitueeipqcohvelpxpcex5jtuep4wnezfqjknphuewwkoq)

 Just replace the CID with the CID that you got as a result from bacalhau list

Running the same job on the whole dataset (13674 Images!)CID from the previous job

Upload the whole dataset which contains 13674 images along with the weight file

You can choose the methods mentioned above to upload your dataset directory

And copy the CID bafybeifvpl2clsdy4rc72oi4iqlyyt347ms64kmmuqwuai5j2waurnsk5e

Uploaded Dataset link: [https://bafybeifvpl2clsdy4rc72oi4iqlyyt347ms64kmmuqwuai5j2waurnsk5e.ipfs.nftstorage.link/](https://bafybeifvpl2clsdy4rc72oi4iqlyyt347ms64kmmuqwuai5j2waurnsk5e.ipfs.nftstorage.link/)

To run on the whole dataset we just need to replace the input CID in the -v flag with the CID of the whole dataset


```
 bacalhau docker run \
--gpu 1 \
-v bafybeifvpl2clsdy4rc72oi4iqlyyt347ms64kmmuqwuai5j2waurnsk5e:/datasets \
ultralytics/yolov5:latest \
-- /bin/bash -c 'python detect.py --weights ../../../datasets/yolov5s.pt --source ../../../datasets  --project  ../../../outputs'
```



---

**165000 Images (27GB)**

Dataset link:

https://bafybeiekic3o3tuefajvlqeiyvvbq5kkr2g27qivuawpbqva7frv42radm.ipfs.nftstorage.link/


```
bacalhau docker run \
--gpu 1 \
-v bafybeiekic3o3tuefajvlqeiyvvbq5kkr2g27qivuawpbqva7frv42radm:/datasets \
ultralytics/yolov5:latest \
-- /bin/bash -c 'python detect.py --weights ../../../datasets/yolov5s.pt --source ../../../datasets  --project  ../../../outputs'
```


