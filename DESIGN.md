# Bacalhau - The Filecoin Distributed Computation Framework

## Background

A common need for using a large dataset is to do [embarrassingly parallel](https://en.wikipedia.org/wiki/Embarrassingly_parallel) compute jobs next to the data (e.g. data and compute are on the same physical/virtual device). Scenarios where this may need to occur include:

- Creating derivative datasets of the original stored data (e.g. a user only needs the first 10 lines from each file for a large set of files)
- Processing data and returning the results (e.g. a user needs to compute the statistics (mean, mode, std deviation, min, max, etc) for each of a series of columns in a dataset that may be stored across a large number of files)
- Transforming the data in place or creating a new data set (e.g. a user needs to convert the encoding for each file for a large set of files)

One of the most popular implementations of this type of technology was [MapReduce](https://en.wikipedia.org/wiki/MapReduce) made popular by Google and, eventually, Hadoop. This has largely been surpassed by newer technologies

However, the existing solutions often can be challenging for the average data engineer:

- The most popular solutions for doing distributed data engineering jobs include [Spark](https://en.wikipedia.org/wiki/Apache_Spark) (which optimizes for in-memory processing for streams), [Hadoop/HDFS](https://hadoop.apache.org/) (which optimizes for batch, on-disk) and roll your own. There are also many hosted solutions provided by most hyperscale clouds (e.g. AWS EMR, Google DataProc, Azure HDInsights).
- However, running and maintaining such solutions requires a significant amount of investment:
  - The computation APIs for these platforms are often difficult to understand and restricted to specific platforms (non-portable) or in new languages.
  - Maintaining clusters can be very challenging and expensive.
  - The cost of executing each job can be very computationally expensive and bottlenecked via contention.

While the market has adopted these in large numbers (combined valuation of these companies is $100B+), there remains a large number of individuals who are underserved and/or overbilled for solutions.

## Using IPFS for Distributed Computation

IPFS is already optimized for storing large datasets; it hosts petabytes of public information today and is expected to grow quickly in the future. Adding the ability to execute distributed jobs will unlock much higher efficiency of use of these data sets through using each node's spare compute.

The flow of a submission will look like:

- Using a standard interface (e.g. CLI, SDK, API) to submit an arbitrary job (e.g. code that executes inside a Docker container) to IPFS that can be sharded and run, in parallel, on many nodes.
- Storage providers who self-identify as having a component or all of the data execute the job.
- Return the results of the jobs back to IPFS or any arbitrary endpoint.

```bash
# PSEUDO CODE

# -x - takes a CID where a function has been defined
# -c - the CID of the IPLD dataset

ipfs job submit -x bafye9d4c615a2d4f7c4413cw5aec9c2f3308a6d32ffa -c bafy2bzacedcdedrghloawlwkntdhqnknqzxgh26ddwix7ld2a5ygagco3ngee
```

One could imagine a veneer over the top of this that allows for submitting "linux compatible" things. For ex, the below would be wrapped in a 

```bash
# PSEUDO CODE - returns all log file entries where GPS is within a range of a certain location

# GPS of Lisbon is lat 38.736946, longitude -9.142685
# Reducing decimals to 2 sig digits will be within 1.1km of this point

ipfs job submit -e "sed /38.7[2-4]....,-9.1[3-5]...." -c bafy2bzacedcdedrghloawlwkntdhqnknqzxgh26ddwix7ld2a5ygagco3ngee
```

This will take advantage of the computing power and data locality the Storage Providers have already, running only on the data stored there, but in the context of the larger dataset.

Ideally, we will also allow much more fine-grained control, specifying location, machine type, etc. Examples:

- for each sector, run this indexing function over it and return me the index results
- for each of these sectors containing labeled image data, run this training function on your GPUs and return me the gradient

## Ideal User Benefits:

- **Data sets become more useful**: A very cheap solution that can execute any arbitrary code against a large data set.
- **Trade cost for speed**: It would not be nearly as fast, but for jobs that do not require the speed, it would be orders of magnitude cheaper than competitive solutions.
- **Drive maintenance to zero**: Further, the maintenance of a highly available data would be zero, since it would be hosted on IPFS.

## Sample Scenarios

- **SCENARIO 1** Downscale large images to smaller images before use
  - Sarah is a data scientist at Fooco, a hedge fund that analyzes satellite data for retail information (e.g. looks at agricultural growth patterns across regions)
  - She's built a model which takes the [NOAA](https://www.noaa.gov/) satellite data which amounts to 30 TB per day of data and builds models off it
  - However, because the image sizes are so large, it takes her more than a week to process each model
  - What she'd really like is every night, after the data for that day has been uploaded to IPFS by NOAA, she could downscale all the images from 10MB per frame to just 200k (by reducing bits and colors)
  - She writes a cron job to fire a program to do so every night, which runs against a CID, and reuploads the resulting downscaled data to a new entry, and returns the CID when done.
- Map/Reduce over many log files to create a "filtered" view for later use
  - Dana is an IT administrator for a weather sensor company. They have sensors all over the world that they use to collect datetime, GPS, temperature, barometer, and rainfall data that they push to a central repository, once per second.
  - Every hour, the system exports the information to a log file, and once a day they write the file to IPFS. These files are aggregated behind a single CID (in some way) such that a single interaction could operate across all of them at once.
  - The total data changes about 10 GB per day across the system - they have about 100 TB of total data.
  - Michelle would like to query the data to get a subset of the information - to retrieve the past ten years of rainfall data within 10 km of Lisbon.
  - She writes a simple program to query for the information across the whole system, execute the filtering, and push the results into a single CID:
    - Behind the scenes, the "orchestration" provider farms out requests to "worker" nodes that have shards of the top-level data set.
    - Each worker node pulls the data from their local file into memory runs the program (which filters for just the information necessary) and writes the resulting data structure back to the CID.
    - It then passes the intermediate CID back to the "orchestration" provider, which merges the results into a single unified CID, and lets Michelle know the job is done.
  - Michelle can now pull the resulting data down to her local machine for further analysis.
- How Storage Provider benefit:
  - Their nodes no longer just store data, they can process it - for which they should be compensated
  - Even if low compute (relatively), they have SOME compute and it's MOSTLY going unused
  - Payment channels are neat
    - Need some way of metering to know roughly how to agree on pricing apriori (charge as you go is kind of annoying for this sort of work)
    - Definitely, don't need to have a perfect payment system setup from the get-go, but definitely something to think about
- Process isolation, security, etc
  - Docker might be good enough? Otherwise could use kubernetes to spin up whole VMs and such. Lots of prior art here, how do lambdas work?
  - Stake in the ground:
    - Use Firecracker to create a virtual environment in which a single container runs.
    - No local disk access is available (all writes must go back to IPFS)
  - No real need to trust storage providers, computation should be cheap enough, replicas abundant enough, that you can redundantly run the same computations across different storage providers with the same data and double-check the results against each other
    - This is a key insight IMO: You can run every computation 3x, and it should be cheaper than doing this sort of work any other way *even* with that overhead.

- **SCENARIO 2** Process data before retrieval
  - Lochana is a data scientist building models based on satellite images.
  - The satellite data is often very large, much larger than she needs for her processing. On the order of 1GB per image and millions of pixels.
  - She needs data no bigger than 1 MB per image, grayscale, downscaled.
  - She already uses a python library which downscales per her needs.
  - She has a file `process.py` which includes the python code necessary to execute in a function called 'downscale()' which takes a file handle to local, processes it, and returns a bytestream.
  - She executes the following command:
```
ifps job submit -f process.py -r requirements.txt -c QmbWqxBEKC3P8tqsKc98xmWNzrzDtRLMiMPL8wBuTGsMnR
```
  - This runs the command in a local executor, first installing all the python packages necessary, and then executing them, on the subset of data available on that node.
  - Once complete, the system returns the CID of the updated dataset that she can download.

- **SCENARIO 3** Want to burst to cloud but cannot move entire dataset in short time
  - DHASH CAN YOU HELP FLESH OUT
  - **PUSH COMPUTE INTO GENE SEQUENCER**
  - **PIPE TO S3**

## Components to Build

- Build an application that listens for jobs over libp2p, receives payment somehow, runs the job in {kuberenetes, docker, idk}, and returns the result to the use (ideally the 'result' is in the form of an ipfs object and we can just return the hash).
- The inputs to the job should be a 'program' and a CID. The node should pull the CID requested into a car file (it should already be in this format for sectors that they have sealed) and pass that to the docker image (probably mounted somewhere to the image).
- This should run as a sidecar to lotus nodes, and should be fairly isolate so as not to mess with the node's primary operation.
- Need a payment system, payment estimator
- Need a dataset aggregator - where a single large dataset can describe many CIDs that may span sectors

## What's with the Name?
Bacalhau means cod (the fish) in Portuguese (where several folks were brainstorming this topic). 

Compute-Over-Data == Cod == Bacalhau

## Prior Art / Parallel Projects
* IPFS-FAN - distributed serverless - https://research.protocol.ai/publications/ipfs-fan-a-function-addressable-computation-network/delarocha2021a.pdf
* IPLS : A Framework for Decentralized Federated Learning- https://arxiv.org/pdf/2101.01901v1.pdf
* Interplanetary Distributed Computing (2018) - https://github.com/yenkuanlee/IPDC
* IPTF - IPFS + TensorFlow (2018) - https://github.com/tesserai/iptf
* Lurk -> Run queries over Filecoin Sealed Data (no public paper yet)
* Radix - Nomad based scheduler for  IPFS cluster (only) - high level spec doc https://docs.google.com/document/d/18hdYBmDlvusEOQ-iSNIO_IAEOvJVFL1MyAU_B8hON9Q/edit?usp=sharing
* Bringing Arbitrary Compute to Authoritative Data https://queue.acm.org/detail.cfm?id=2645649
* Manta: a scalable, distributed object store https://github.com/joyent/manta
