---
sidebar_label: "Oceanography - Data Conversion"
sidebar_position: 5
description: "Oceanography data conversion with Bacalhau"
---
# Oceanography - Data Conversion


The Surface Ocean CO₂ Atlas (SOCAT) contains measurements of the [fugacity](https://en.wikipedia.org/wiki/Fugacity) of CO₂ in seawater around the globe. But to calculate how much carbon the ocean is taking up from the atmosphere, these measurements need to be converted to the partial pressure of CO₂. We will convert the units by combining measurements of the surface temperature and fugacity.  Python libraries (xarray, pandas, numpy) and the pyseaflux package facilitate this process.

In this example tutorial, our focus will be on running the oceanography dataset with Bacalhau, where we will investigate the data and convert the workload. This will enable the execution on the Bacalhau network, allowing us to leverage its distributed storage and compute resources.

## Prerequisites

To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md)

## Running Locally

### Downloading the dataset

For the purposes of this example we will use the [SOCATv2022](https://www.socat.info/index.php/version-2022/) dataset in the "Gridded" format from the [SOCAT website](https://www.socat.info/) and long-term global sea surface temperature data from [NOAA](https://downloads.psl.noaa.gov/Datasets/noaa.oisst.v2/sst.mnmean.nc) - information about that dataset can be found [here](https://psl.noaa.gov/data/gridded/data.noaa.oisst.v2.highres.html).


```bash
%%bash
mkdir -p inputs
curl -L --output ./inputs/SOCATv2022_tracks_gridded_monthly.nc.zip https://www.socat.info/socat_files/v2022/SOCATv2022_tracks_gridded_monthly.nc.zip
curl --output ./inputs/sst.mnmean.nc https://downloads.psl.noaa.gov/Datasets/noaa.oisst.v2/sst.mnmean.nc
```

### Installing dependencies

Next let's write the `requirements.txt`. This file will also be used by the Dockerfile to install the dependencies.


```python
%%writefile requirements.txt
Bottleneck==1.3.5
dask==2022.2.0
fsspec==2022.5.0
netCDF4==1.6.0
numpy==1.21.6
pandas==1.3.5
pip==22.1.2
pyseaflux==2.2.1
scipy==1.7.3
xarray==0.20.2
zarr>=2.0.0
```


```bash
%%bash
pip install -r requirements.txt > /dev/null
```

### Reading and Viewing Data


```python
import fsspec # for reading remote files
import xarray as xr

# Open the zip archive using fsspec and load the data into xarray.Dataset
with fsspec.open("./inputs/SOCATv2022_tracks_gridded_monthly.nc.zip", compression='zip') as fp:
    ds = xr.open_dataset(fp)

# Display information about the dataset    
ds.info()
```


```python
time_slice = slice("2010", "2020") # select a decade
res = ds['sst_ave_unwtd'].sel(tmnth=time_slice).mean(dim='tmnth') # compute the mean for this period
res.plot() # plot the result

```

We can see that the dataset contains latitude-longitude coordinates, the date, and a series of seawater measurements. Below is a plot of the average sea surface temperature (SST) between 2010 and 2020, where data have been collected by buoys and vessels.


![image](../../../../static/img/examples/Average-SST.png 'Average-SST')


### Data Conversion

To convert the data from fugacity of CO2 (fCO2) to partial pressure of CO2 (pCO2) we will combine the measurements of the surface temperature and fugacity. The conversion is performed by the [pyseaflux](https://seaflux.readthedocs.io/en/latest/api.html?highlight=fCO2_to_pCO2#pyseaflux.fco2_pco2_conversion.fCO2_to_pCO2) package.


### Writing the Script

Let's create a new file called `main.py` and paste the following script in it:

```python
%%writefile main.py
import fsspec
import xarray as xr
import pandas as pd
import numpy as np
import pyseaflux


def lon_360_to_180(ds=None, lonVar=None):
    lonVar = "lon" if lonVar is None else lonVar
    return (ds.assign_coords({lonVar: (((ds[lonVar] + 180) % 360) - 180)})
            .sortby(lonVar)
            .astype(dtype='float32', order='C'))


def center_dates(ds):
    # start and end date
    start_date = str(ds.time[0].dt.strftime('%Y-%m').values)
    end_date = str(ds.time[-1].dt.strftime('%Y-%m').values)

    # monthly dates centered on 15th of each month
    dates = pd.date_range(start=f'{start_date}-01T00:00:00.000000000',
                          end=f'{end_date}-01T00:00:00.000000000',
                          freq='MS') + np.timedelta64(14, 'D')

    return ds.assign(time=dates)


def get_and_process_sst(url=None):
    # get noaa sst
    if url is None:
        url = ("/inputs/sst.mnmean.nc")

    with fsspec.open(url) as fp:
        ds = xr.open_dataset(fp)
        ds = lon_360_to_180(ds)
        ds = center_dates(ds)
        return ds


def get_and_process_socat(url=None):
    if url is None:
        url = ("/inputs/SOCATv2022_tracks_gridded_monthly.nc.zip")

    with fsspec.open(url, compression='zip') as fp:
        ds = xr.open_dataset(fp)
        ds = ds.rename({"xlon": "lon", "ylat": "lat", "tmnth": "time"})
        ds = center_dates(ds)
        return ds


def main():
    print("Load SST and SOCAT data")
    ds_sst = get_and_process_sst()
    ds_socat = get_and_process_socat()

    print("Merge datasets together")
    time_slice = slice("1981-12", "2022-05")
    ds_out = xr.merge([ds_sst['sst'].sel(time=time_slice),
                       ds_socat['fco2_ave_unwtd'].sel(time=time_slice)])

    print("Calculate pco2 from fco2")
    ds_out['pco2_ave_unwtd'] = xr.apply_ufunc(
        pyseaflux.fCO2_to_pCO2,
        ds_out['fco2_ave_unwtd'],
        ds_out['sst'])

    print("Add metadata")
    ds_out['pco2_ave_unwtd'].attrs['units'] = 'uatm'
    ds_out['pco2_ave_unwtd'].attrs['notes'] = ("calculated using" +
                                               "NOAA OI SST V2" +
                                               "and pyseaflux package")

    print("Save data")
    ds_out.to_zarr("/processed.zarr")
    import shutil
    shutil.make_archive("/outputs/processed.zarr", 'zip', "/processed.zarr")
    print("Zarr file written to disk, job completed successfully")

if __name__ == "__main__":
    main()
```
This code loads and processes SST and SOCAT data, combines them, computes pCO2, and saves the results for further use.

## Upload the Data to IPFS

The simplest way to upload the data to IPFS is to use a third-party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this you need an account with a pinning service like [NFT.storage](https://nft.storage/) or [Pinata](https://pinata.cloud/). Once registered you can use their UI or API or SDKs to upload files.

This resulted in the IPFS CID of `bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y`.


## Setting up Docker Container

We will create a  `Dockerfile` and add the desired configuration to the file. These commands specify how the image will be built, and what extra requirements will be included.


```python
%%writefile Dockerfile
FROM python:slim

RUN apt-get update && apt-get -y upgrade \
    && apt-get install -y --no-install-recommends \
    g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /project

COPY ./requirements.txt /project

RUN pip3 install -r requirements.txt

COPY ./main.py /project

CMD ["python","main.py"]
```

### Build the container

We will run `docker build` command to build the container:

```
docker build -t <hub-user>/<repo-name>:<tag> .
```

Before running the command replace;

**`hub-user`** with your docker hub username, If you don’t have a docker hub account [follow these instructions to create a Docker account](https://docs.docker.com/docker-id/), and use the username of the account you created

**`repo-name`** with the name of the container, you can name it anything you want

**`tag`** this is not required but you can use the latest tag

### Push the container

Now you can push this repository to the registry designated by its name or tag.

```
docker push <hub-user>/<repo-name>:<tag>
```

:::tip
For more information about working with custom containers, see the [custom containers example](../../../setting-up/workload-onboarding/custom-containers/).
:::

## Running a Bacalhau Job

Now that we have the data in IPFS and the Docker image pushed, next is to run a job using the `bacalhau docker run` command


```bash
%%bash  --out job_id
bacalhau docker run \
    --input ipfs://bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y \
    --id-only \
    --wait \
    ghcr.io/bacalhau-project/examples/socat:0.0.11 \
    -- python main.py
```

### Structure of the command

Let's look closely at the command above:

`bacalhau docker run`: call to Bacalhau

`--input ipfs://bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y`: CIDs to use on the job. Mounts them at '/inputs' in the execution.

`ghcr.io/bacalhau-project/examples/socat:0.0.11`: the name and the tag of the image we are using

`python main.py`: execute the script


When a job is submitted, Bacalhau prints out the related `job_id`. We store that in an environment variable so that we can reuse it later on.


```python
%env JOB_ID={job_id}
```

## Checking the State of your Jobs

**Job status**: You can check the status of the job using `bacalhau list`.


```bash
%%bash
bacalhau list --id-filter ${JOB_ID}
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
rm -rf results
mkdir -p ./results # Temporary directory to store the results
bacalhau get ${JOB_ID} --output-dir ./results # Download the results
```

## Viewing your Job Output

To view the file, run the following command:


```bash
%%bash
cat results/stdout
```

## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).