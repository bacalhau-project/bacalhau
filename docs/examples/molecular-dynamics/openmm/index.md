---
sidebar_label: "Simulation with OpenMM"
sidebar_position: 1
---
# Molecular Simulation with OpenMM

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/molecular-dynamics/openmm/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=molecular-dynamics/openmm/index.ipynb)

[OpenMM](https://github.com/openmm/openmm) is a toolkit for molecular simulation. Physic based libraries like OpenMM are then useful for refining the structure and exploring functional interactions with other molecules. It provides a combination of extreme flexibility (through custom forces and integrators), openness, and high performance (especially on recent GPUs) that make it truly unique among simulation codes.

References:

* https://github.com/openmm/openmm
* https://github.com/Openzyme/openzyme (Docker scaffolding to run OpenMM)


### Goal

The goal of this notebook is to showcase how to containerize an OpenMM workload so that it can be executed on the Bacalhau network and to take advantage of the distributed storage & compute resources.

### Prerequisites

This example requires Docker. If you don't have Docker installed, you can install it from [here](https://docs.docker.com/install/). Docker commands will not work on hosted notebooks like Google Colab, but the Bacalhau commands will.

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation)

## Protein data

We use a processed 2DRI dataset that represents the ribose binding protein in bacterial transport and chemotaxis. The source organism is the [Escherichia coli](https://en.wikipedia.org/wiki/Escherichia_coli) bacteria.
You can find more details on this protein at the related [RCSB Protein Data Bank page](https://www.rcsb.org/structure/2dri).

![image.png](./2dri-image.png)



Protein data can be stored in a `.pdb` file, this is a human readable format.
It provides for description and annotation of protein and nucleic acid structures including atomic coordinates, secondary structure assignments, as well as atomic connectivity.
Please find more info about PDB format in [this article](https://www.cgl.ucsf.edu/chimera/docs/UsersGuide/tutorials/pdbintro.html).

Let us sneak peak into the dataset by printing the first 10 lines of the file.
Among other things, we can see it contains a number of ATOM records. These describe the coordinates of the atoms that are part of the protein.


```bash
%%bash
head ./dataset/2dri-processed.pdb
```

    REMARK   1 CREATED WITH OPENMM 7.6, 2022-07-12
    CRYST1   81.309   81.309   81.309  90.00  90.00  90.00 P 1           1 
    ATOM      1  N   LYS A   1      64.731   9.461  59.430  1.00  0.00           N  
    ATOM      2  CA  LYS A   1      63.588  10.286  58.927  1.00  0.00           C  
    ATOM      3  HA  LYS A   1      62.707   9.486  59.038  1.00  0.00           H  
    ATOM      4  C   LYS A   1      63.790  10.671  57.468  1.00  0.00           C  
    ATOM      5  O   LYS A   1      64.887  11.089  57.078  1.00  0.00           O  
    ATOM      6  CB  LYS A   1      63.458  11.567  59.749  1.00  0.00           C  
    ATOM      7  HB2 LYS A   1      63.333  12.366  58.879  1.00  0.00           H  
    ATOM      8  HB3 LYS A   1      64.435  11.867  60.372  1.00  0.00           H  


## Prepare & Run the task


1. Upload the data to IPFS
1. Create a docker image with the code and dependencies
1. Run the docker image on the Bacalhau network using the IPFS data



```python
!(export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    Your system is darwin_arm64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.1 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.1/bacalhau_v0.3.1_darwin_arm64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.1/bacalhau_v0.3.1_darwin_arm64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into . successfully.
    Client Version: v0.3.1
    Server Version: v0.3.1
    env: PATH=./:/Users/phil/.pyenv/versions/3.9.7/bin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.gvm/bin:/opt/homebrew/opt/findutils/libexec/gnubin:/opt/homebrew/opt/coreutils/libexec/gnubin:/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/bin:/Users/phil/.pyenv/shims:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Library/TeX/texbin:/usr/local/MacGPG2/bin:/Users/phil/.nexustools


### Upload the Data to IPFS

The first step is to upload the data to IPFS. The simplest way to do this is to use a third party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this you need an account with a pinning service like [web3.storage](https://web3.storage/) or [Pinata](https://pinata.cloud/). Once registered you can use their UI or API or SDKs to upload files.

For the purposes of this example I pinned the `2dri-processed.pdb` file to IPFS via [web3.storage](https://web3.storage/).

This resulted in the IPFS CID of `bafybeig63whfqyuvwqqrp5456fl4anceju24ttyycexef3k5eurg5uvrq4`.

<!-- TODO: Add link to notebook showing people how to upload data to IPFS -->

### Create a Docker Image to Process the Data

Next we will create the docker image that will process the data. The docker image will contain the code and dependencies needed to perform the conversion. This code originated with [wesfloyd](https://github.com/wesfloyd/openmm-test). Thank you Wes!

:::tip
For more information about working with custom containers, see the [custom containers example](../../workload-onboarding/custom-containers/).
:::

The key thing to watch out for here is the paths to the data. I'm using the default bacalhau output directory `/outputs` to write my data to. And the input data is mounted to the `/inputs` directory. But as you will see in a moment, web3.storage has added another `input` directory that we need to account for.


```python
%%writefile run_openmm_simulation.py
import os
from openmm import *
from openmm.app import *
from openmm.unit import *

# Input Files
input_path = '/inputs/2dri-processed.pdb'
os.path.exists(input_path) # check if input file exists
pdb = PDBFile(input_path)
forcefield = ForceField('amber14-all.xml', 'amber14/tip3pfb.xml')

# Output
output_path = '/outputs/final_state.pdbx'
if not os.path.exists(os.path.dirname(output_path)): # check if ouput dir exists
    os.makedirs(os.path.dirname(output_path))

# System Configuration

nonbondedMethod = PME
nonbondedCutoff = 1.0*nanometers
ewaldErrorTolerance = 0.0005
constraints = HBonds
rigidWater = True
constraintTolerance = 0.000001
hydrogenMass = 1.5*amu

# Integration Options

dt = 0.002*picoseconds
temperature = 310*kelvin
friction = 1.0/picosecond
pressure = 1.0*atmospheres
barostatInterval = 25

# Simulation Options

steps = 10
equilibrationSteps = 0
#platform = Platform.getPlatformByName('CUDA')
platform = Platform.getPlatformByName('CPU')
#platformProperties = {'Precision': 'single'}
platformProperties = {}
dcdReporter = DCDReporter('trajectory.dcd', 1000)
dataReporter = StateDataReporter('log.txt', 1000, totalSteps=steps,
    step=True, time=True, speed=True, progress=True, elapsedTime=True, remainingTime=True, potentialEnergy=True, kineticEnergy=True, totalEnergy=True, temperature=True, volume=True, density=True, separator='\t')
checkpointReporter = CheckpointReporter('checkpoint.chk', 1000)

# Prepare the Simulation

print('Building system...')
topology = pdb.topology
positions = pdb.positions
system = forcefield.createSystem(topology, nonbondedMethod=nonbondedMethod, nonbondedCutoff=nonbondedCutoff,
    constraints=constraints, rigidWater=rigidWater, ewaldErrorTolerance=ewaldErrorTolerance, hydrogenMass=hydrogenMass)
system.addForce(MonteCarloBarostat(pressure, temperature, barostatInterval))
integrator = LangevinMiddleIntegrator(temperature, friction, dt)
integrator.setConstraintTolerance(constraintTolerance)
simulation = Simulation(topology, system, integrator, platform, platformProperties)
simulation.context.setPositions(positions)

# Minimize and Equilibrate

print('Performing energy minimization...')
simulation.minimizeEnergy()
print('Equilibrating...')
simulation.context.setVelocitiesToTemperature(temperature)
simulation.step(equilibrationSteps)

# Simulate

print('Simulating...')
simulation.reporters.append(dcdReporter)
simulation.reporters.append(dataReporter)
simulation.reporters.append(checkpointReporter)
simulation.currentStep = 0
simulation.step(steps)

# Write file with final simulation state

state = simulation.context.getState(getPositions=True, enforcePeriodicBox=system.usesPeriodicBoundaryConditions())
with open(output_path, mode="w+") as file:
    PDBxFile.writeFile(simulation.topology, state.getPositions(), file)
print('Simulation complete, file written to disk at: {}'.format(output_path))
```

    Overwriting run_openmm_simulation.py


To run the script above all we need is a Python environment with the OpenMM library installed.
We install that via the package manager [conda](https://docs.conda.io/projects/conda/en/latest/user-guide/index.html).
Below is the resulting Dockerfile; to keep this example concise we the Docker build command is commented out.


```python
%%writefile Dockerfile
FROM conda/miniconda3

RUN conda install -y -c conda-forge openmm

WORKDIR /project

COPY ./run_openmm_simulation.py /project

LABEL org.opencontainers.image.source https://github.com/bacalhau-project/examples

CMD ["python","run_openmm_simulation.py"]
```

    Overwriting Dockerfile



```bash
%%bash
#docker buildx build --platform linux/amd64 --push -t ghcr.io/bacalhau-project/examples/openmm:0.3 .
```

### Test the Container Locally

Before we upload the container to the Bacalhau network, we should test it locally to make sure it works.


```bash
%%bash
docker run \
    -v $(pwd)/dataset:/inputs/ \
    -v $(pwd)/output:/output \
    ghcr.io/bacalhau-project/examples/openmm:0.3
```

### Run a Bacalhau Job

Now that we have the data in IPFS and the docker image pushed, we can run a job on the Bacalhau network.

I find it useful to first run a simple test with a known working container to ensure the data is located in the place I expect, because some storage providers add their own opinions. E.g. web3.storage wraps the directory uploads in a top level directory.


```bash
%%bash
rm -rf stdout stderr volumes shards
bacalhau docker run \
        --inputs bafybeig63whfqyuvwqqrp5456fl4anceju24ttyycexef3k5eurg5uvrq4 \
        ubuntu -- ls /inputs
```

    Job successfully submitted. Job ID: 5836a70b-0ed1-4741-90fa-390c6a4f1137
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):
    
    	       Creating job for submission ... done ✅
    	       Finding node(s) for the job ... done ✅
    	             Node accepted the job ... done ✅
    	                                   ... done ✅
    	   Job finished, verifying results ... done ✅
    	      Results accepted, publishing ... done ✅
    	                                  
    Results CID: QmbVhcvWKmZbLd6ZKiDctUYY7DN5jQBKsbPsrTz5aGFY68
    Job Results By Node:
    Node QmXaXu9N:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    Node QmdZQ7Zb:
      Shard 0:
        Status: Completed
        Container Exit Code: 0
        Stdout:
          2dri-processed.pdb
        Stderr: <NONE>
    
    To download the results, execute:
      bacalhau get 5836a70b-0ed1-4741-90fa-390c6a4f1137
    
    To get more details about the run, execute:
      bacalhau describe 5836a70b-0ed1-4741-90fa-390c6a4f1137


Let's switch to our custom container image.


```bash
%%bash
rm -rf stdout stderr volumes shards
bacalhau docker run \
    --inputs bafybeig63whfqyuvwqqrp5456fl4anceju24ttyycexef3k5eurg5uvrq4 \
    ghcr.io/bacalhau-project/examples/openmm:0.3 -- ls -la /inputs/
```

    Job successfully submitted. Job ID: d9ca75a5-a766-42e1-aab5-b97a5ae1e7f1
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):
    
    	       Creating job for submission ... done ✅
    	       Finding node(s) for the job ... done ✅
    	             Node accepted the job ... done ✅
    	   Job finished, verifying results ... done ✅
    	      Results accepted, publishing ... done ✅
    	                                  
    Results CID: QmcVp5m7MngLa7QU9prwzZZHHgKgmJaW6wrvEyufUCwX9x
    Job Results By Node:
    Node QmYgxZiy:
      Shard 0:
        Status: Completed
        Container Exit Code: 0
        Stdout:
          total 4080
    drwxr-xr-x 2 root root    4096 Oct 10 12:04 .
    drwxr-xr-x 1 root root    4096 Oct 10 12:08 ..
    -rw-r--r-- 1 root root 4167654 Oct 10 12:04 2dri-processed.pdb
        Stderr: <NONE>
    
    To download the results, execute:
      bacalhau get d9ca75a5-a766-42e1-aab5-b97a5ae1e7f1
    
    To get more details about the run, execute:
      bacalhau describe d9ca75a5-a766-42e1-aab5-b97a5ae1e7f1


And finally let's run the full job. This time I will not download the data immediately, because the job takes a few minutes to complete. The commands are below, but you will need to wait until the job completes before they work.


```bash
%%bash --out job_id
bacalhau docker run \
    --inputs bafybeig63whfqyuvwqqrp5456fl4anceju24ttyycexef3k5eurg5uvrq4 \
    --wait \
    --id-only \
    ghcr.io/bacalhau-project/examples/openmm:0.3 -- python run_openmm_simulation.py
```


```python
%env JOB_ID={job_id}
```

    env: JOB_ID=10e11cba-3de2-4507-85f6-a8f2b53d110b



```bash
%%bash
bacalhau list --id-filter=${JOB_ID} --no-style
```

     CREATED   ID        JOB                      STATE      VERIFIED  PUBLISHED               
     12:08:16  10e11cba  Docker ghcr.io/bacal...  Completed            /ipfs/QmUpBj6Eacz5Y5... 


### Get Results

Now let's download and display the result from the results directory. We can use the `bacalhau get` command to download the results from the output data volume.


```bash
%%bash
rm -rf stdout stderr volumes shards
bacalhau get ${JOB_ID} # Download the results
```

    Fetching results of job '10e11cba-3de2-4507-85f6-a8f2b53d110b'...


```bash
%%bash
ls -l volumes/outputs
```

    total 6656
    -rw-r--r-- 1 phil staff 6578336 Oct 10 13:11 final_state.pdbx


That's all folks!
