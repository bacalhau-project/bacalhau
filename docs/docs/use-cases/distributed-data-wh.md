---
sidebar_label: 'Distributed Data Warehouse'
sidebar_position: 0
---

# Distributed Data Warehouse

## Traditional data warehousing challenges
Traditional data warehousing has certain challenges:
1. **Scalability Issues**: Centralized systems struggle to scale with the growing data and user demands, making it expensive and complex.
1. **Performance Bottlenecks**: Increased user and application access can lead to performance degradation, causing delays in data retrieval and analysis.
1. **High Data Engineering Costs**: Centralizing data requires significant data engineering efforts, especially when data comes from diverse sources.
1. **Maintenance**: Routine maintenance and upgrades can be resource-intensive and disruptive. Upkeep demands significant investments in hardware, software, and skilled personnel.
1. **Slow To Adapt**: Adapting to new data sources or schema changes is often slow and complex in centralized architectures.

With many organizations investigating Data Mesh architectures, where data is treated and managed like a product by the teams that generate it, there may be a better way than centralizing this data.

## Solution: Distributed Data Warehousing
As teams treat data as a product, perhaps under a federated governance model, querying data at its origin can bypass costly ETL processes. It’s often just as effective to send the computation to the data rather than the other way around, even outside a data mesh.

By positioning compute nodes close to where systems generate data, you can dispatch computational tasks or queries directly to the data, and then deliver results through the selected storage solution. The resulting computations are usually way smaller, making them easier to transmit than entire datasets, thereby enabling near real-time processing. Backing up required log  data is still done conventionally.

**Implementing a distributed data warehouse offers clear benefits**:

1. **Reduced Data Movement**: Directly querying data at its source drastically lowers the need to move large amounts of data to a central location, saving on costs and reducing the workload on data engineers. This setup allows users to pull concise results from their queries and still archive daily data for audit trails.
1. **Easier Scaling**: A distributed data warehouse can grow easily by adding more compute nodes in new areas. If you double the number of data sources, you won't need to more than double your central warehouse's capacity; just boost the infrastructure where the data is collected.
1. **Real-Time Querying**: Users don't have to wait for the day's data to be transferred and processed. They can run queries on the most recent data immediately, making decisions based on the latest information about the business.
1. **Avoid Vendor Lock-in**: Using one provider for a centralized data warehouse might not be the best long-term strategy. A single provider can become less effective over time due to limited capabilities or changes in service. The sunk costs associated with one provider can make it tough to switch, even when necessary.


## Bacalhau as a Distributed Data Warehouse Orchestrator
Bacalhau offers a way to tap into the benefits of a distributed data warehouse with minimal changes to existing processes. Many organizations already have the necessary data and compute resources for analytics, spread across various databases, servers, and edge locations.

With Bacalhau, these scattered resources can be harnessed to form a dynamic data warehouse. Installing its lightweight agents where the data already exists allows compute jobs to run on-site. This means there’s no need to move large datasets around or to majorly change ETL processes and data models.

### Flexible Compute Nodes
Bacalhau’s compatibility with Docker and WebAssembly allows for a wide variety of workloads to be run efficiently. Its compute nodes are not only open for custom execution engines but are also versatile enough to support everything from .NET applications to legacy IBM AS/400 systems.

### Diverse Data Access
To cut the high costs of data transfer, Bacalhau facilitates direct access to local data, reducing the need for extensive data movement. It supports S3-compatible storage, IPFS, and direct local storage, making data readily available for computation.

### Smart Job Allocation
With Bacalhau, you can manage how jobs are assigned to compute nodes with precision. Nodes can be targeted for specific jobs using labels that denote their characteristics, and the latest platform updates allow for even more nuanced selections based on these labels.

## Implementation Example
**Domain**: Retail Industry

### Scenario
Consider a retail chain with multiple stores spread across different regions. Each store has its own POS system collecting sales data. Traditionally, this data is batch-uploaded to a central data warehouse at the end of each day for processing and report generation.

**Bacalhau Deployment**

With Bacalhau, each store installs a compute node that processes data locally. A control plane node orchestrates tasks across the compute nodes, distributing work based on various selection criteria. Retailers can now query data in near real-time, gaining instant insights without the need for extensive data transfer or central processing infrastructure.

#### Benefits

1. **Real-time Insights**: Retailers can query data and generate reports in near real-time, enabling quicker decision-making.
1. **Reduced Network Bandwidth**: By processing data locally, Bacalhau significantly reduces the amount of data that needs to be transferred over the network.
1. **Enhanced Security**: Minimizing data transfer also reduces the risk of data incursions, ensuring better compliance with data governance regulations.
1. **Cost-Effectiveness**: Bacalhau eliminates the need for expensive central processing infrastructure, making it a cost-effective solution.

### Code Implementation

### Step 0: Prerequisites

Before you start, you’ll need:

1. Storage Solution: Have a storage provider or location ready for the job results.
1. Firewall Configuration: Adjust your firewall settings so your node can talk to the rest of the Bacalhau network.
1. Hosting Setup: Prepare a physical server, a virtual machine, or a cloud-based instance. Note that Bacalhau compute nodes should not be run inside a Docker container.
1. Bacalhau CLI: Install the Bacalhau CLI on your local machine, following the [instructions provided](../getting-started/installation.md)

### Step 1: Provisioning Hardware

To get started with setting up a distributed data warehouse, you’ll need the right infrastructure. If you’re not familiar with setting up a private node network, check out this guide. For our example, assume you’re starting from scratch. You’ll need:
1. **Control Plane Node**: This is your operation’s headquarters, coordinating tasks throughout your network. Recommended Specs:
    1. Instances: 1
    1. Disk Space: 25-100 GB
    1. CPU: 1-8 cores (vCPU)
    1. RAM: 4-16 GB
1. **Compute Node(s)**: These workhorses run your code and access data that’s local or close to it. Recommended Specs:
    1. Instances: 1-N (We’ll use 4 in this example, symbolizing 4 different locations)
    1. Disk Space: 32-500 GB
    1. CPU: 1-8 cores (vCPU)
    1. RAM: 4-16 GB

**Note**: It’s crucial that the Control Plane Node can communicate with the Compute Nodes. For guidance on this, you can follow this tutorial.

### Step 2: Installing the Compute and Requestor Node
Bacalhau uses a node called a requester node to orchestrate jobs in the network, communicating with the compute nodes to distribute work according to the various selection criteria. Once you have installed Bacalhau, you can run the requester node as follows.
```bash
bacalhau serve \
--node-type requester \
--private-internal-ipfs=false
```
After your requester node is operational, it will generate environment variables that must be configured on your compute nodes. Here’s what to do next:
1. **Set Environment Variables**: Record the environment variables from the requester node and apply them to each machine designated as a compute node.
1. **Install Compute Nodes**: At each store location with data, you’ll need to install a Bacalhau compute node on a machine with data access. Use the standard Bacalhau installation instructions, but for a private network, you’ll specify a unique –peer value.
1. **Job Distribution**: To distribute jobs to the compute nodes, you have two main strategies:
    1. **Job Selection Policies**: Implement custom logic to decide if a node should run a job. More details can be found in the [job selection policy documentation](../setting-up/jobs/job-selection.md).
    1. **Node Labels**: For a simpler approach, use node labels to target specific nodes or groups of nodes. This is the recommended method for this guide.
With your infrastructure and settings ready, you can initiate the compute nodes using the command line provided in the [Bacalhau quick-start guide](../getting-started/installation.md).

```bash
# We added a path to allow-listed-local-paths where we store our data, and
# a set of labels to allow us to target specific nodes
bacalhau serve \
--node-type compute \
--ipfs-connect $IPFS_CONNECT \
--private-internal-ipfs=false \
--labels "storeid=1,region=EU,country=FR,city=Paris"
--allow-listed-local-paths "/node/data"
--peer env
```
Note the addition of labels, which allow us to target specific nodes when we run our jobs. Here we add a store identifier, a region, a country and a city so that we can target queries in our warehouse to any of these labels. In reality, we may add more metadata here to provide even more flexibility in precisely targeting stores by one or more of these labels.

For this example, we’ll use nodes with the following labels:

| Node (ID) | Region       | Country | City                   |
|-----------|--------------|---------|------------------------|
| 1         | EU           | FR      | Australia              |
| 2         | NorthAmerica | CA      | Australia              |
| 3         | EU           | DE      | United States of America |
| 4         | NorthAmerica | US      | Canada                 |

By strategically applying labels, you can direct jobs to:

1. All stores, regardless of location.
1. A single store within a specific region.
1. All stores within a particular region.
1. A single store within a given country.
1. All stores across a specific country.
1. A particular set of stores identified by their IDs.

The `–allow-listed-local-path` option is used to mount a specified directory on each node, providing a way for the system’s users to feed local data directly into computations happening on that node. By using `-i src=file:/storedata,dst=/inputs/storedata`, any job processed on that node can access the data in `/storedata` through the path `/inputs/storedata`.


### Step 3: Running a query across the network

Before we can query our data, we need to know what shape it has, so we want to run a query against the transaction data on one of the compute nodes (it doesn’t matter which). As we know that each compute node has access to transaction data at `/node/data/transactions.csv` we can query for that using DuckDB. If the data was made available in a form that DuckDB does not understand, we can use any other tool that works with docker, or webassembly, or even implement our own pluggable executor to support specific use-cases.

We’ll need to set an environment variable to point to our Bacalhau cluster, in this case by specifying `BACALHAU_CLIENT_API_HOST` as this will remove the need to provide a command line flag to the Bacalhau program on each invocation. As each command we run will also need to access the transactions database, we’ll also store that in an environment variable to reduce the amount of typing necessary.

```bash
export BACALHAU_API_HOST="34.155.152.133"
export TRXN_DATA_INPUT="src=file:/node/data/transactions.csv,dst=/inputs/trxn.csv"
```
To find the shape of our transaction data, we can run the following command to query the database and print the results to the terminal:

```bash
bacalhau \
docker run \
-f -i $TRXN_DATA_INPUT \
expanso/duckdb-ddw:0.0.1 \
"DESCRIBE TABLE '/inputs/trxn.csv';"
```
Here: 
1. `docker run` is telling Bacalhau to run a docker container
1. `-f` tells it to log output to the terminal
1. `-i` sets up the input data
1. `expanso/duckdb-ddw:0.0.1` is the docker container to run
1. the final section `"DESCRIBE TABLE '/inputs/trxn.csv';"` is the query we want to run against the data. 

In this case, after a short delay, we should see the following output:

```bash
column_name,column_type,null,key,default,extra
Invoice,VARCHAR,YES,,,
StockCode,VARCHAR,YES,,,
Description,VARCHAR,YES,,,
Quantity,BIGINT,YES,,,
InvoiceDate,VARCHAR,YES,,,
Price,DOUBLE,YES,,,
"Customer ID",BIGINT,YES,,,
Country,VARCHAR,YES,,,
```

### Step 4: Getting the results

So far, we’ve only run queries that show output to the terminal using the `-f` flag. In practice, we’ll be running queries with more output, and potentially across multiple nodes. In this case we’ll want to publish the results, so that anything the compute task writes to the `/outputs` folder is made available to you in the terminal as a file (or files). To do this, we use the `-p` flag to specify a publisher.

As we want to store our output in S3 (or any S3-compatible storage), we have made sure that each of the compute nodes has credentials that allow it to connect to S3. Details on these credential requirements are available in the Bacalhau documentation. In our case, we want to store the output in an S3 bucket called “bacalhau-usecase-distributed-data-warehouse”. To avoid having to type this for each command, we’ll store the full publisher URL in an environment variable, showing we want to also include the job id, and the execution id in the output’s prefix.

```bash
export PUBLISHER=s3://bacalhau-usecase-distributed-data-warehouse/{jobID}/{executionID}
```
We can now use specify `-p $PUBLISHER` in our docker run commands to have the output written to that location.

### Step 5: Working with the data

Now that we’re all set up, we can query our data. For instance, we can use the selector flags (-s) to target specific nodes. For instance, to find the total of all transactions in the Paris store, we can run:

```bash
bacalhau \
docker run \
-s city=Paris \
-f -i $TRXN_DATA_INPUT \
expanso/duckdb-ddw:0.0.1 \
"SELECT ROUND(SUM(Price),2) as Total FROM '/inputs/trxn.csv';"
```

This displays the output below:
```bash
Total
1620674.31
```
At this point, we might want to get more data, perhaps a list of all the countries who buy products from our European stores. This time, we want the output to be stored in S3, and so we also specify `-p $PUBLISHER` so that if we write to `/outputs` then the data will be put into our bucket.

We now need to write out data to a specific location, and so we will do that with the following command. Note that we need to specify `–target=all` as we expect it to run on more than one compute node. Without this it will pick only a single node in that region.

```bash
bacalhau \
docker run \
-s region=EU \
--target=all \
-p $PUBLISHER \
-i $TRXN_DATA_INPUT \
expanso/duckdb-ddw:0.0.1 \
"COPY
(SELECT DISTINCT(Country) as Country FROM '/inputs/trxn.csv' ORDER BY(Country))
TO '/outputs/results.csv' (HEADER, DELIMITER ',');"
```

This time, we see different output as Bacalhau shows us a job ID (in this case, `073ab816-9b9e-4dfa-9e90-6c4498aa1de6`) and then shows progress as the job is happening. Once complete it tells us how we can get the details of the job, but running `bacalhau describe 073ab816-9b9e-4dfa-9e90-6c4498aa1de6`. Doing this shows lots of output, but the following cut-down snippet shows information on where the query was run, and where the outputs are stored.

```bash
State:
  CreateTime: "2023-10-17T12:28:03.88046717Z"
  Executions:
  - ComputeReference: e-7c942a16-420d-4736-809c-1d6676e13a1c
    CreateTime: "2023-10-17T12:28:03.902519479Z"
    JobID: 073ab816-9b9e-4dfa-9e90-6c4498aa1de6
    NodeId: QmfKmkipkbAQu3ddChL4sLdjjcqifWQzURCin2QKUzovex
    PublishedResults:
      S3:
Bucket: bacalhau-usecase-distributed-data-warehouse           Key:073ab816-9b9e-4dfa-9e90-6c4498aa1de6/e-7c942a16-420d-4736-809c-1d6676e13a1c/
      StorageSource: s3
    ...
    State: Completed
    UpdateTime: "2023-10-17T12:28:07.510247793Z"
    Version: 3
  - ComputeReference: e-7e346e49-d659-4188-ae04-cf5c28fd963b
    CreateTime: "2023-10-17T12:28:03.907178486Z"
    JobID: 073ab816-9b9e-4dfa-9e90-6c4498aa1de6
    NodeId: QmeD1rESiDtdVTDgekXAmDDqgN9ZdUHGGuMAC77krBGqSv
    PublishedResults:
      S3:
Bucket: bacalhau-usecase-distributed-data-warehouse
Key:073ab816-9b9e-4dfa-9e90-6c4498aa1de6/e-7e346e49-d659-4188-ae04-cf5c28fd963b/
      StorageSource: s3
    ...
    State: Completed
    UpdateTime: "2023-10-17T12:28:07.890888772Z"
    Version: 3
```

Here we can see the two executions performed on EU nodes, with the bucket and key containing the outputs from our execution. Using the standard Bacalhau structure for outputs, we know that we will find CSV files in our bucket at `073ab816-9b9e-4dfa-9e90-6c4498aa1de6/e-7c942a16-420d-4736-809c-1d6676e13a1c/outputs/results.csv` and `s3://bacalhau-usecase-distributed-data-warehouse/073ab816-9b9e-4dfa-9e90-6c4498aa1de6/e-7e346e49-d659-4188-ae04-cf5c28fd963b/outputs/results.csv`. To access this data requires that the user have AWS credentials, a tool to download the data, and a way to merge all of the results into one. Rather than burden the user with this work, we can wrap our command line invocations with something less complex.


### Step 6: Simplifying the interface

The previous sections of this tutorial have shown how to use, and specify, various Bacalhau features using the Bacalhau command line interface (CLI). While the interface is flexible and allows you to configure work in any way you wish, it does involve a lot of typing that might be overwhelming in an interactive scenario such as this.

Fortunately, Bacalhau provides an API, used by the command line interface, which means anything you can do in the CLI, you can do via its API. This provides even more flexibility in presentation, making it possible to build specialized interfaces for different use-cases. As an example of how you can use the Python SDK to build a specialized interface, you can take a look at the Distributed Data Warehouse Client which allows you to store commonly keyed information in a configuration file. This program of more than 200 lines of code let’s us move away from querying all regional stores like this rather lengthy command.

```bash
$ bacalhau \
docker run \
-s region=EU \
--target=all \
-p $PUBLISHER \
-i $TRXN_DATA_INPUT \
expanso/duckdb-ddw:0.0.1 \
"COPY
(SELECT DISTINCT(Country) as Country FROM '/inputs/trxn.csv' ORDER BY(Country))
TO '/outputs/results.csv' (HEADER, DELIMITER ',');"
```

We are able to move to querying like the following and get merged results written locally, ready for opening in a spreadsheet or further processing. So quite neat and way simpler.
```bash
$ poetry run ddw -a -s region=EU "SELECT DISTINCT(Country) as Country FROM '/inputs/transactions.csv' ORDER
BY(Country)"
Submitted job: d802752f-e0b1-417a-8b98-55381ce4f7fb
Output written to: output-d802752f.csv
```

**Note**: Only the first line is the input by us, the rest is the response from the system itself.

After checking you have the dependencies described in the repository you can install this client to try it out with

```bash
git clone <https://github.com/bacalhau-project/examples.git>
cd examples/distributed-datawarehouse
poetry install
```

Whilst this vastly reduces the complexity of the interface, not to mention the amount of typing, it is really just a starting point, beyond which it is possible to imagine a more complete user interface that allows you to recall previous queries and see results in different formats.


## Conclusion

Bacalhau’s distributed computing approach empowers retailers to overcome the challenges associated with centralized data processing. By deploying Bacalhau, retail chains can harness the full potential of their geographically dispersed data, enabling real-time insights, enhanced security, and cost savings.

Hopefully this tutorial has shown how we can take advantage of individual Bacalhau features to achieve our goal. It has shown how we can use labels and selectors to target single nodes or groups of nodes distributed across the globe. How changing the publisher that is responsible for disseminating the results makes it easy it is to switch from built in storage to using S3-compatible options instead. Finally it has shown how by taking advantage of Bacalhau’s powerful API and using the Python SDK, we are able to provide a different a experience with simple tools where interactivity is required.