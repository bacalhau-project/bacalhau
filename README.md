# Bacalhau - The Filecoin Distributed Computation Framework

## Background

A common need for using a large dataset is to do [embarrassingly parallel](https://en.wikipedia.org/wiki/Embarrassingly_parallel) compute jobs next to the data (e.g. data and compute are on the same physical/virtual device). Scenarios where this may need to occur include:

- Creating derivative datasets of the orginal stored data (e.g. a user only needs the first 10 lines from each file for a large set of files)
- Processing data and returning the results (e.g. a user needs to compute the statistics (mean, mode, std deviation, min, max, etc) for a each of a series of columns in a dataset that may be stored across a large number of files)
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

- Using a standard interface (e.g. CLI, SDK, API) to submit arbitrary an arbitrary job (e.g. code that executes inside a Docker container) to IPFS that can be sharded and run, in parallel, on many nodes.
- Miners who self-identify as having a component or all of the data execute the job.
- Return the results of the jobs back to IPFS or any arbitrary endpoint.

```bash
# PSEUDO CODE - returns all log file entries where GPS is within a range of a certain location

# GPS of Lisbon is lat 38.736946, longitude -9.142685
# Reducing decimals to 2 sig digits will be within 1.1km of this point

ifps job submit -e "sed /38.7[2-4]....,-9.1[3-5]....' -c bafy2bzacedcdedrghloawlwkntdhqnknqzxgh26ddwix7ld2a5ygagco3ngee
```

This will take advantage of the computing power and data locality the miners have already, running only on the data stored there, but in the context of the larger dataset.

Ideally we will also allow much more fine grained control, specifying location, machine type, etc. Examples:

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
  - Dana is an IT administrator for a weather sensor company. They have sensors all over the world that they use to collect datetime, GPS, temperature, barometer and rainfall data that they push to a central repository, once per second.
  - Every hour, the system exports the information to a log file, and once a day they write the file to IPFS. These files are aggregated behind a single CID (in some way) such that a single interaction could operate across all of them at once.
  - The total data changes about 10 GB per day across the system - they have about 100 TB of total data.
  - Michelle would like to query the data to get a subset of the information - to retrieve the past ten years of rainfall data within 10 km of Lisbon.
  - She writes a simple program to query for the information across the whole system, execute the filtering, and push the results into a single CID:
    - Behind the scenes, the "orchestration" miner farms out requests to "worker" miners that have shards of the top level data set.
    - Each miner pulls the data from their local file into memory runs the program (which filters for just the information necessary) and writes the resulting data structure back to the CID.
    - It then passes the intermediate CID back to the "orchestration" miner, which merges the results into a single unified CID, and lets Michelle know the job is done.
  - Michelle can now pull the resulting data down to her local machine for further analysis.
- How miners benefit:
  - Their nodes no longer just store data, they can process it - for which they should be compensated
  - Even if low compute (relatively), they have SOME compute and it's MOSTLY going unused
  - Payment channels are neat
    - Need some way of metering to know roughly how to agree on pricing apriori (charge as you go is kind of annoying for this sort of work)
    - Definitely dont need to have a perfect payment system setup from the get-go, but definitely something to think about
- Process isolation, security, etc
  - Docker might be good enough? Otherwise could use kubernetes to spin up whole VMs and such. Lots of prior art here, how do lambdas work?
  - Stake in the ground:
    - Use Firecracker to create a virtual environment in which a single container runs.
    - No local disk access is available (all writes must go back to IPFS)
  - No real need to trust miners, computation should be cheap enough, replicas abundant enough, that you can redundantly run the same computations across different miners with the same data and double check the results against each other
    - This is a key insight IMO: You can run every computation 3x, and it should be cheaper than doing this sort of work any other way *even* with that overhead.

- **SCENARIO 2** Process data before retrieval
  - Lochana is a data scientist building models based on satellite images.
  - The satellite data is often very large, much larger than she needs for her processing. On the order of 1GB per image and millions of pixels.
  - She needs data no bigger than 1 MB per image, gray scale, downscaled.
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
- The inputs to the job should be a 'program' and a cid. The miner should pull the cid requested into a car file (it should already be in this format for sectors that they have sealed) and pass that to the docker image (probably mounted somewhere to the image).
- This should run as a sidecar to miners, and should be fairly isolate so as not to mess with the miners primary operation.
- Need a payment system, payment estimator
- Need a dataset aggregator - where a single large dataset can describe many CIDs that may span sectors

## What's with the Name?


## Raw Notes / Questions

- Performance:
  - What performance task will we target (e.g. almost certainly not performant enough for ML requirements around weight transfers - but maybe?)
  - What performance/$ will be MVP for us?
  - What's the minimum size task where this is valuable?
  - What's the minimum time for the smallest possible task? e.g. 10 minutes? 1 hour?
  - What's the comparison for perf/$ vs. other platforms for equivalent job?
- Core functionality:
  - Need to encoding of data correctly so that it can be broken up
- Functionality extension:
  - Two level orchestration - a job that then fires off additional jobs.
    - Corollary - how does an orchestration job identify miners who have subsets of the data
    - Corollary - sub jobs will have to have their own job sucess criteria (e.g. potentially different timeouts, costs, etc), and push results back to the originally executor.
    - Corollary - sub jobs will be paid subsets of the original cost to run, but only if their sub-work has been accepted. This means we will have to think about how to allow "acceptance" of the top level job to trickle down to acceptance of subjobs and a way to divvy up the original payment to the amount of work each sub-miner did
      - To what extent are we dependent on a meta-deal-making apparatus to allow apparent pieces to go into multiple sectors
    - Corollary - Where do we store results (both intermediate results and end results) - push back onto the chain to start, but is that a good solution?
      - When we get to cross-miner (which will be required to deal with datasets larger than a single sector), we will have to understand how to share intermediate proofs.
      - Need to define a format for intermediate data plus the requisite state for continuing the computation.
    - Do we need to develop a structure for intermediate entities to be doing the sharding of the work (person A receives the job, uses an indexer to find the shards and does the work to split up and hand off) - they should extract some value for doing this (even though they didn't do any of the 'actual' work)
      - If they don't see anyone bidding on the work, theoretically they could do the work themselves and charge for that.
    - Eventual flow for multi-miner orchestration:
      - Job layout:
                request
                analysis
                distributed computation
                aggregation
                client payment
                contributor claim and reimbursement
      - Built in audit trail - signature for everyone who contributed to it (allowing everyone who did work to get paid after the top level job accepted - because then if you release the data AND it's used THEN you have proof that you used the data)
      - Also incentivizes folks to act quickly - beat other miners to the result - Because the 'winning fork' will be the one that gets paid.
        - If I calculate some results but don't release them fast enough, the final answer will be computed without relying on my contribution.
      - The trick is figuring out how to embed the attribution in the intermediate proofs such that it can be stripped out.
        - Will need to design the computations in such a way that a valid response must include this metadata.
  - "Data sets"
    - Need a first class way to address cross sector data
    - How do we force data to be distributed among many miners (otherwise, there will be less benefit to a system like this)- How do we allow selection of machine profile (I want this to execute on an accelerator/GPU)?
  - Compute
    - Possible structure: Miner can verify Lurk proofs, and for smart contracts able to minimally parse its data (which should ultimately be a subset of IPLD)
      - Put a contract on chain that says, I want to perform query Q on data D.
      - Q is a content-addressable expression corresponding to a Lurk program.
      - D is the root hash of some Lurk data (which happens to be stored in some number of sectors).
      - It could also be the case that this data exists outside of Filecoin. The Filecoin part might be cold storage for some other data.
      - But the client (and perhaps the chain itself) can know that some set of miners do have the data.
      - So at least that set of storage providers (miners) will be able to efficiently perform the query.
    - Sub miners claiming portions of payment
      - Setting aside the problem of 'delivery', anyone who can create a proof of correct response, can then claim the payment.
      - The delivery problem is that the proof will just certify that some CID representing the response is correct.
    - How do we verify data was transmitted after computation
      - So the important point is that Lurk provides a mechanism for a kind of 'function-evaluation-addressable data' which can actually be trusted.
      - So in the same way that I can verify a CID and know that the data someone gives me really corresponds to what was intended by the person who gave me the CID…
      - I can verify that the result of a computation really is correct, even though the person who specified the new-kind-of-CID didn't know the result (so couldn't hash it to produce a digest).
    - Could we build a global computational graph that would enable caching/storage of computation on Filecoin sectors - reducing redundant computations?
    - How do we leverages Filecoin more - e.g. Filecoin ses proof of replication for its intended purpose:
      - We could have two queries: certified and uncertified.
      - Uncertified means: here's the data I want to query. If you have it (all of it), go nuts!
      - Certified means:
        - This is a cross sector query, and I know it will only be answered if adversarial parties cooperate.
        - in addition to proving the data has the root you requested, I must also prove that I possess that data in a Filecoin sector.
  - Partnerships
    - Should we partner folks who want to develop a languages for certifiable data (IPLD) interpretation and transformation for use in decentralized systems.
    - How coupled should this effort be with other chains and/or FVM?
  - Would it be useful to deploy a local test network or would it be ok as a subset of the IPFS network
    - Both have separate use cases
    - We should think about targeting low-trust environment, but allow for a spectrum
  - What is the way to describe the higher level primitive that maps to the entire dataset
    - Imagine you had a compute job with many sequences along many data sets
    - Imagine shipping the entire compute job based on evaluating a condition of what data is here
  - Framework for thinking of this - invoke dynamic (from Java)
  - Should be executed as a layer 2 protocol
  - Make it work with Kubernetes (should have the ability to distribute to pods, not just nodes?)

  - Customer dataset built on-prem - clinical diagnostics
    - Works against existing data
    - USED TO:
      - Minio with blob store
      - Do puts into local host
      - Look up via hash
      - Docker container spawn to execute command
    - Company moved away from S3/Minio to IPFS because S3/Minio does not allow for lazy setup/changing of the cluster sizing
      - Lazy pulling of archived
  - Alternates: Containers or WASM running in the place (not just in the local experience)
  - Value of provability:
    - I'm not running a data center
    - I want to make sure the storage provider (who i don't trust) ran the binary that i handed them
    - HIPAA could encourage the use of zero knowledge proofs
    - Alternative to proof is to use trusted environment (SGX) to execute compute
    - Plausibility in order:
      - Running a docker container
      - Need to run arbitrary compute - WASM & FVM is not acceptable for now
  - Core seems to be disk throughput and locality issue
  - Don't use IPLD to manage cross sector node
  - Need to support WinCE - file lands on the WinCE node and then the compute gets executed
  - Node agent runs on every node and uses inotify (our job is the job that spun up the node sequencer and roughly in 30 hours to fire)

  - WASM on lurk
  - Have the proof of running
  - As a compute requester, I should be able to select the style of running - trusted, proveable, fast, etc
