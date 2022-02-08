# Bacalhau - The Filecoin Distributed Computation Framework

## demo

<https://user-images.githubusercontent.com/264658/152514573-b7b115ce-4123-486c-983a-8e26acf4b86d.mp4>

## running locally

### requirements

* linux
* go >= 1.16
* [ignite](https://ignite.readthedocs.io/en/stable/installation/)
* A machine that will support KVM. This could be a bare metal instance, or one from the [following list](https://github.com/weaveworks/ignite/blob/main/docs/cloudprovider.md):

  * AWS Bare Metal Instances (<https://eu-west-1.console.aws.amazon.com/ec2/v2/home?region=eu-west-1#InstanceTypes>:) - where "Bare Metal" is true. **NOTE** These can be quite costly (>$2k/mo). Launch with [Ubuntu Focal](https://eu-west-1.console.aws.amazon.com/ec2/v2/home?region=eu-west-1#LaunchInstanceWizard:ami=ami-08ca3fed11864d6bb), ARM64 architecture, r6g.metal instance type on Kernal >=4.14 appears to be the cheapest available. 
  * GCP - Need both special instance types and licensing approval. [Read more](https://blog.kubernauts.io/ignite-on-google-cloud-5d5228a5ffec)
  * Azure - No special configuration required for machine types D4s_v3, D8s_v3, E4s_v3, E8s_v3

### installation notes

Following the instructions [here](https://github.com/weaveworks/ignite/blob/e2a0f39b614177f6fd3b84817ac7f34a00c1e288/docs/installation.md) is fairly straightforward.

Several errors you may run into:

* Make sure you have a version of go later than 1.16 installed and it's for the correct architecture. You can check this with: `go version` and `uname -a`. `arm64` and `aarch64` need `arm64` go.
* If you get the message `/usr/local/bin/ignite: cannot execute binary file: Exec format error`, you have installed go for the wrong architecture, and ignite has installed based on the wrong GOARCH variable. Update your go version and reinstall ignite using the script

Run the following commands to start ignite daemon and create a VM:

Terminal 1:
```bash
sudo su
ignite image import weaveworks/ignite-ubuntu
ignite create weaveworks/ignite-ubuntu --name bac-vm-1 --cpus 1 --memory 1GB  --size 1GB --ssh \
&& ignite create weaveworks/ignite-ubuntu --name bac-vm-2 --cpus 1 --memory 1GB  --size 1GB --ssh \
&& ignite create weaveworks/ignite-ubuntu --name bac-vm-3 --cpus 1 --memory 1GB  --size 1GB --ssh

# Clone the repo and build bacalhau
gh repo clone filecoin-project/bacalhau
make build

# Copy the binary to all the VMs
ignite cp bin/bacalhau bac-vm-1:bacalhau \
&& ignite cp bin/bacalhau bac-vm-2:bacalhau \
&& ignite cp bin/bacalhau bac-vm-3:bacalhau
```

Terminal 2:

```bash
ignite start bac-vm-1 -i
```

Terminal 3:

```bash
ignite start bac-vm-2 -i
```

Terminal 4:

```bash
ignite start bac-vm-3 -i
```
 

### start compute nodes

Have a few terminal windows.

This starts the first compute node listening on port 8080 so we can connect to a known port.

Terminal 1:

```bash
./bin/bacalhau serve --port 8080 --dev
```

It will print out the command to run in other terminal windows to connect to this first node (start at least one more so now we have a cluster)

For example:

```bash
go run . serve --peer /ip4/127.0.0.1/tcp/8080/p2p/<peerid> --jsonrpc-port <randomport>
```

Execute this in terminals 2, 3 and 4.

### submit a job with the CLI

Now we submit a job to the network:

Terminal 5:

```bash
./bin/bacalhau submit
```

This should start an ignite VM in each of the compute nodes we have running.

It will also print back the path to the results folder where each compute node has written its output.

The output folder path has this pattern `outputs/<job_id>/<node_id>`

So if 2 nodes both complete job `123` - you will see 3 folders in `outputs/123` one for each node that completed it.

To submit any other job, just execute it inside of quotes. E.g.

```bash
./bin/bacalhau submit --job "unzip 5m-Sales-Records.zip; for X in {1..10}; do bash -c \"sed 's/Office Supplies/Booze/' '5m Sales Records.csv' -i\"; sleep 2; done"
```

### Clean up

To delete the vms, execute the following two commands:

```bash
ignite stop bac-vm-1 bac-vm-2 bac-vm-3 
ignite rm bac-vm-1 bac-vm-2 bac-vm-3
```
