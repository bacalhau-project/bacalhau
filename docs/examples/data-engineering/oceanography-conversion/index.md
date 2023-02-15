---
sidebar_label: "Oceanography - Data Conversion"
sidebar_position: 10
description: "Oceanography data conversion with Bacalhau"
---
# Oceanography - Data Conversion

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/data-engineering/oceanography-conversion/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=data-engineering/oceanography-conversion/index.ipynb)

The Surface Ocean CO₂ Atlas (SOCAT) contains measurements of the [fugacity](https://en.wikipedia.org/wiki/Fugacity) of CO2 in seawater around the globe. But to calculate how much carbon the ocean is taking up from the atmosphere, these measurements need to be converted to the partial pressure of CO2. We will convert the units by combining measurements of the surface temperature and fugacity.  Python libraries (xarray, pandas, numpy) and the pyseaflux package facilitate this process.

The goal of this notebook is to investigate the data and convert dockerize the workload so that it can be executed on the Bacalhau network, to take advantage of the distributed storage and compute resources.

## Prerequisites

This example requires Docker. If you don't have Docker installed, you can install it from [here](https://docs.docker.com/install/). Docker commands will not work on hosted notebooks like Google Colab, but the Bacalhau commands will.

Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation)

## The Data

The raw data is available on the [SOCAT website](https://www.socat.info/). We will use the [SOCATv2021](https://www.socat.info/index.php/version-2021/) dataset in the "Gridded" format to perform this calculation. First, let's take a quick look at some data:


```python
!command -v docker >/dev/null 2>&1 || { echo >&2 "I require docker but it's not installed.  Aborting."; exit 1; }
```


```python
!(export BACALHAU_INSTALL_DIR=.; curl -sL https://get.bacalhau.org/install.sh | bash)
path=!echo $PATH
%env PATH=./:{path[0]}
```

    Your system is linux_amd64
    No BACALHAU detected. Installing fresh BACALHAU CLI...
    Getting the latest BACALHAU CLI...
    Installing v0.3.15 BACALHAU CLI...
    Downloading https://github.com/filecoin-project/bacalhau/releases/download/v0.3.15/bacalhau_v0.3.15_linux_amd64.tar.gz ...
    Downloading sig file https://github.com/filecoin-project/bacalhau/releases/download/v0.3.15/bacalhau_v0.3.15_linux_amd64.tar.gz.signature.sha256 ...
    Verified OK
    Extracting tarball ...
    NOT verifying Bin
    bacalhau installed into . successfully.
    Client Version: v0.3.15
    Server Version: v0.3.15
    env: PATH=./:/home/gitpod/.pyenv/versions/3.8.13/bin:/home/gitpod/.pyenv/libexec:/home/gitpod/.pyenv/plugins/python-build/bin:/home/gitpod/.pyenv/plugins/pyenv-virtualenv/bin:/home/gitpod/.pyenv/plugins/pyenv-update/bin:/home/gitpod/.pyenv/plugins/pyenv-installer/bin:/home/gitpod/.pyenv/plugins/pyenv-doctor/bin:/home/gitpod/.pyenv/shims:/ide/bin/remote-cli:/home/gitpod/.nix-profile/bin:/home/gitpod/.local/bin:/home/gitpod/.sdkman/candidates/maven/current/bin:/home/gitpod/.sdkman/candidates/java/current/bin:/home/gitpod/.sdkman/candidates/gradle/current/bin:/workspace/.cargo/bin:/home/gitpod/.rvm/gems/ruby-3.1.2/bin:/home/gitpod/.rvm/gems/ruby-3.1.2@global/bin:/home/gitpod/.rvm/rubies/ruby-3.1.2/bin:/home/gitpod/.pyenv/plugins/pyenv-virtualenv/shims:/home/gitpod/.pyenv/shims:/workspace/go/bin:/home/gitpod/.nix-profile/bin:/ide/bin/remote-cli:/home/gitpod/go/bin:/home/gitpod/go-packages/bin:/home/gitpod/.nvm/versions/node/v16.18.1/bin:/home/gitpod/.yarn/bin:/home/gitpod/.pnpm:/home/gitpod/.pyenv/bin:/workspace/.rvm/bin:/home/gitpod/.cargo/bin:/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin/:/home/gitpod/.local/bin:/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/gitpod/.nvm/versions/node/v16.18.1/bin:/home/gitpod/.rvm/bin



```bash
%%bash
mkdir -p inputs
curl --output ./inputs/SOCATv2022_tracks_gridded_monthly.nc.zip https://www.socat.info/socat_files/v2022/SOCATv2022_tracks_gridded_monthly.nc.zip
curl --output ./inputs/sst.mnmean.nc https://downloads.psl.noaa.gov/Datasets/noaa.oisst.v2/sst.mnmean.nc
```

      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                     Dload  Upload   Total   Spent    Left  Speed
    100 35.5M  100 35.5M    0     0  34.6M      0  0:00:01  0:00:01 --:--:-- 34.6M
      % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                     Dload  Upload   Total   Spent    Left  Speed
    100 60.8M  100 60.8M    0     0  14.6M      0  0:00:04  0:00:04 --:--:-- 14.6M


Next let's write the requirements.txt and install the dependencies. This file will also be used by the Dockerfile to install the dependencies.


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

    Writing requirements.txt



```bash
%%bash
pip install -r requirements.txt > /dev/null
```


```python
import fsspec # for reading remote files
import xarray as xr
with fsspec.open("./inputs/SOCATv2022_tracks_gridded_monthly.nc.zip", compression='zip') as fp:
    ds = xr.open_dataset(fp)
ds.info()
```

    xarray.Dataset {
    dimensions:
    	xlon = 360 ;
    	ylat = 180 ;
    	tmnth = 624 ;
    	bnds = 2 ;
    
    variables:
    	float64 xlon(xlon) ;
    		xlon:units = degrees_east ;
    		xlon:point_spacing = even ;
    		xlon:axis = X ;
    		xlon:modulo = 360.0 ;
    		xlon:standard_name = longitude ;
    	float64 ylat(ylat) ;
    		ylat:units = degrees_north ;
    		ylat:point_spacing = even ;
    		ylat:axis = Y ;
    		ylat:standard_name = latitude ;
    	datetime64[ns] tmnth(tmnth) ;
    		tmnth:axis = T ;
    		tmnth:bounds = tmnth_bnds ;
    		tmnth:time_origin = 01-JAN-1970 ;
    		tmnth:standard_name = time ;
    	datetime64[ns] tmnth_bnds(tmnth, bnds) ;
    	float64 count_ncruise(tmnth, ylat, xlon) ;
    		count_ncruise:long_name = Number of cruises ;
    		count_ncruise:units = count ;
    		count_ncruise:history = From SOCAT_ABCD_data_for_gridding ;
    		count_ncruise:summary = Number of datasets containing observations in the grid cell ;
    	float64 fco2_count_nobs(tmnth, ylat, xlon) ;
    		fco2_count_nobs:long_name = Number of fco2 obs ;
    		fco2_count_nobs:units = count ;
    		fco2_count_nobs:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_count_nobs:summary = Total number of observations in the grid cell. ;
    	float32 fco2_ave_weighted(tmnth, ylat, xlon) ;
    		fco2_ave_weighted:long_name = fCO2 mean - per cruise weighted ;
    		fco2_ave_weighted:units = uatm ;
    		fco2_ave_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_ave_weighted:summary = Mean of fco2 recomputed computed by calculating the arithmetic mean value for each cruise passing through the cell and then averaging these datasets. ;
    	float32 fco2_ave_unwtd(tmnth, ylat, xlon) ;
    		fco2_ave_unwtd:long_name = fCO2 mean - unweighted all obs ;
    		fco2_ave_unwtd:units = uatm ;
    		fco2_ave_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_ave_unwtd:summary = Arithmetic mean of all fco2 recomputed values found in the grid cell. ;
    	float32 fco2_min_unwtd(tmnth, ylat, xlon) ;
    		fco2_min_unwtd:long_name = fCO2 min ;
    		fco2_min_unwtd:units = uatm ;
    		fco2_min_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_min_unwtd:summary = Minimum value of fco2 recomputed observed in the grid cell. ;
    	float32 fco2_max_unwtd(tmnth, ylat, xlon) ;
    		fco2_max_unwtd:long_name = fCO2 max ;
    		fco2_max_unwtd:units = uatm ;
    		fco2_max_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    	float32 fco2_std_weighted(tmnth, ylat, xlon) ;
    		fco2_std_weighted:long_name = fCO2 std dev - per cruise weighted ;
    		fco2_std_weighted:units = uatm ;
    		fco2_std_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_std_weighted:summary = A weighted standard deviation of fco2 recomputed computed to account for the differing 
    variance estimates for each cruise passing through the cell. The statistical technique is 
    described at See http://wapedia.mobi/en/Weighted_mean#7. ;
    	float32 fco2_std_unwtd(tmnth, ylat, xlon) ;
    		fco2_std_unwtd:long_name = fCO2 std dev - unweighted all obs ;
    		fco2_std_unwtd:units = uatm ;
    		fco2_std_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		fco2_std_unwtd:summary = The standard deviation of fco2 recomputed computed from the unweighted mean. ;
    	float64 sst_count_nobs(tmnth, ylat, xlon) ;
    		sst_count_nobs:long_name = Number of valid sst obs ;
    		sst_count_nobs:units = count ;
    		sst_count_nobs:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_count_nobs:summary = Total number of observations in the grid cell. ;
    	float32 sst_ave_weighted(tmnth, ylat, xlon) ;
    		sst_ave_weighted:long_name = SST mean - per cruise weighted ;
    		sst_ave_weighted:units = degrees C ;
    		sst_ave_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_ave_weighted:summary = Mean of sst computed by calculating the arithmetic mean value for each cruise passing through the cell and then averaging these datasets. ;
    	float32 sst_ave_unwtd(tmnth, ylat, xlon) ;
    		sst_ave_unwtd:long_name = SST mean - unweighted all obs ;
    		sst_ave_unwtd:units = degrees C ;
    		sst_ave_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_ave_unwtd:summary = Arithmetic mean of all sst values found in the grid cell. ;
    	float32 sst_min_unwtd(tmnth, ylat, xlon) ;
    		sst_min_unwtd:long_name = SST min ;
    		sst_min_unwtd:units = degrees C ;
    		sst_min_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_min_unwtd:summary = Minimum value of sst observed in the grid cell. ;
    	float32 sst_max_unwtd(tmnth, ylat, xlon) ;
    		sst_max_unwtd:long_name = SST max ;
    		sst_max_unwtd:units = degrees C ;
    		sst_max_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    	float32 sst_std_weighted(tmnth, ylat, xlon) ;
    		sst_std_weighted:long_name = SST std dev - per cruise weighted ;
    		sst_std_weighted:units = degrees C ;
    		sst_std_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_std_weighted:summary = A weighted standard deviation of sst computed to account for the differing 
    variance estimates for each cruise passing through the cell. The statistical technique is 
    described at See http://wapedia.mobi/en/Weighted_mean#7. ;
    	float32 sst_std_unwtd(tmnth, ylat, xlon) ;
    		sst_std_unwtd:long_name = SST std dev - unweighted all obs ;
    		sst_std_unwtd:units = degrees C ;
    		sst_std_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		sst_std_unwtd:summary = The standard deviation of sst computed from the unweighted mean. ;
    	float64 salinity_count_nobs(tmnth, ylat, xlon) ;
    		salinity_count_nobs:long_name = Number of valid salinity obs ;
    		salinity_count_nobs:units = count ;
    		salinity_count_nobs:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_count_nobs:summary = Total number of observations in the grid cell. ;
    	float32 salinity_ave_weighted(tmnth, ylat, xlon) ;
    		salinity_ave_weighted:long_name = Salinity mean - per cruise weighted ;
    		salinity_ave_weighted:units = PSU ;
    		salinity_ave_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_ave_weighted:summary = Mean of salinity computed by calculating the arithmetic mean value for each cruise passing through the cell and then averaging these datasets. ;
    	float32 salinity_ave_unwtd(tmnth, ylat, xlon) ;
    		salinity_ave_unwtd:long_name = Salinity mean - unweighted all obs ;
    		salinity_ave_unwtd:units = PSU ;
    		salinity_ave_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_ave_unwtd:summary = Arithmetic mean of all salinity values found in the grid cell. ;
    	float32 salinity_min_unwtd(tmnth, ylat, xlon) ;
    		salinity_min_unwtd:long_name = Salinity min ;
    		salinity_min_unwtd:units = PSU ;
    		salinity_min_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_min_unwtd:summary = Minimum value of salinity observed in the grid cell. ;
    	float32 salinity_max_unwtd(tmnth, ylat, xlon) ;
    		salinity_max_unwtd:long_name = Salinity max ;
    		salinity_max_unwtd:units = PSU ;
    		salinity_max_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    	float32 salinity_std_weighted(tmnth, ylat, xlon) ;
    		salinity_std_weighted:long_name = Salinity std dev - per cruise weighted ;
    		salinity_std_weighted:units = PSU ;
    		salinity_std_weighted:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_std_weighted:summary = A weighted standard deviation of salinity computed to account for the differing 
    variance estimates for each cruise passing through the cell. The statistical technique is 
    described at See http://wapedia.mobi/en/Weighted_mean#7. ;
    	float32 salinity_std_unwtd(tmnth, ylat, xlon) ;
    		salinity_std_unwtd:long_name = Salinity std dev - unweighted all obs ;
    		salinity_std_unwtd:units = PSU ;
    		salinity_std_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		salinity_std_unwtd:summary = The standard deviation of salinity computed from the unweighted mean. ;
    	float32 lat_offset_unwtd(tmnth, ylat, xlon) ;
    		lat_offset_unwtd:long_name = Latitude average offset from cell center ;
    		lat_offset_unwtd:units = Deg N ;
    		lat_offset_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		lat_offset_unwtd:summary = The arithmetic average of latitude offsets from the grid cell center for all observations in 
    the grid cell. The value of this offset can vary from -0.5 to 0.5. A value of zero indicates 
    that the computed fco2 mean values are representative of the grid cell center position. ;
    	float32 lon_offset_unwtd(tmnth, ylat, xlon) ;
    		lon_offset_unwtd:long_name = Longitude average offset from cell center ;
    		lon_offset_unwtd:units = Deg E ;
    		lon_offset_unwtd:history = From SOCAT_ABCD_data_for_gridding ;
    		lon_offset_unwtd:summary = The arithmetic average of longitude offsets from the grid cell center for all observations in 
    the grid cell. The value of this offset can vary from -0.5 to 0.5. A value of zero indicates 
    that the computed fco2 mean values are representative of the grid cell center position. ;
    
    // global attributes:
    	:history = PyFerret V7.63 (optimized) 31-May-22 ;
    	:Conventions = CF-1.6 ;
    	:SOCAT_Notes = SOCAT gridded v2022 26-May-2022 ;
    	:title = SOCAT gridded v2022 Monthly 1x1 degree gridded dataset ;
    	:summary = Surface Ocean Carbon Atlas (SOCAT) Gridded (binned) SOCAT observations, with a spatial 
    grid of 1x1 degree and monthly in time. The gridded fields are computed using only SOCAT 
    datasets with QC flags of A through D and SOCAT data points flagged with WOCE flag values of 2. ;
    	:references = http://www.socat.info/ ;
    	:caution = NO INTERPOLATION WAS PERFORMED. SIGNIFICANT BIASES ARE PRESENT IN THESE GRIDDED RESULTS DUE TO THE 
    ARBITRARY AND SPARSE LOCATIONS OF DATA VALUES IN BOTH SPACE AND TIME. ;
    }


```python
time_slice = slice("2010", "2020") # select a decade
res = ds['sst_ave_unwtd'].sel(tmnth=time_slice).mean(dim='tmnth') # average over time
res.plot() # plot the result

```


    ---------------------------------------------------------------------------

    AttributeError                            Traceback (most recent call last)

    Cell In [8], line 3
          1 time_slice = slice("2010", "2020") # select a decade
          2 res = ds['sst_ave_unwtd'].sel(tmnth=time_slice).mean(dim='tmnth') # average over time
    ----> 3 res.plot()


    File ~/.pyenv/versions/3.8.13/lib/python3.8/site-packages/xarray/plot/plot.py:862, in _PlotMethods.__call__(self, **kwargs)
        861 def __call__(self, **kwargs):
    --> 862     return plot(self._da, **kwargs)


    File ~/.pyenv/versions/3.8.13/lib/python3.8/site-packages/xarray/plot/plot.py:330, in plot(darray, row, col, col_wrap, ax, hue, rtol, subplot_kws, **kwargs)
        326     plotfunc = hist
        328 kwargs["ax"] = ax
    --> 330 return plotfunc(darray, **kwargs)


    File ~/.pyenv/versions/3.8.13/lib/python3.8/site-packages/xarray/plot/plot.py:1174, in _plot2d.<locals>.newplotfunc(darray, x, y, figsize, size, aspect, ax, row, col, col_wrap, xincrease, yincrease, add_colorbar, add_labels, vmin, vmax, cmap, center, robust, extend, levels, infer_intervals, colors, subplot_kws, cbar_ax, cbar_kwargs, xscale, yscale, xticks, yticks, xlim, ylim, norm, **kwargs)
       1170 yplt, ylab_extra = _resolve_intervals_2dplot(yval, plotfunc.__name__)
       1172 _ensure_plottable(xplt, yplt, zval)
    -> 1174 cmap_params, cbar_kwargs = _process_cmap_cbar_kwargs(
       1175     plotfunc,
       1176     zval.data,
       1177     **locals(),
       1178     _is_facetgrid=kwargs.pop("_is_facetgrid", False),
       1179 )
       1181 if "contour" in plotfunc.__name__:
       1182     # extend is a keyword argument only for contour and contourf, but
       1183     # passing it to the colorbar is sufficient for imshow and
       1184     # pcolormesh
       1185     kwargs["extend"] = cmap_params["extend"]


    File ~/.pyenv/versions/3.8.13/lib/python3.8/site-packages/xarray/plot/utils.py:905, in _process_cmap_cbar_kwargs(func, data, cmap, colors, cbar_kwargs, levels, _is_facetgrid, **kwargs)
        903 cmap_kwargs.update((a, kwargs[a]) for a in cmap_args if a in kwargs)
        904 if not _is_facetgrid:
    --> 905     cmap_params = _determine_cmap_params(**cmap_kwargs)
        906 else:
        907     cmap_params = {
        908         k: cmap_kwargs[k]
        909         for k in ["vmin", "vmax", "cmap", "extend", "levels", "norm"]
        910     }


    File ~/.pyenv/versions/3.8.13/lib/python3.8/site-packages/xarray/plot/utils.py:185, in _determine_cmap_params(plot_data, vmin, vmax, cmap, center, robust, extend, levels, filled, norm, _is_facetgrid)
        159 def _determine_cmap_params(
        160     plot_data,
        161     vmin=None,
       (...)
        170     _is_facetgrid=False,
        171 ):
        172     """
        173     Use some heuristics to set good defaults for colorbar and range.
        174 
       (...)
        183         Use depends on the type of the plotting function
        184     """
    --> 185     mpl = plt.matplotlib
        187     if isinstance(levels, Iterable):
        188         levels = sorted(levels)


    AttributeError: 'NoneType' object has no attribute 'matplotlib'


We can see that the dataset contains lat-long coordinates, the date, and a series of seawater measurements. Above you can see a plot of the average surface sea temperature (sst) between 2010-2020, where recording buoys and boats have travelled.

## The Task - Large Scale Data Conversion

The goal of this notebook is to convert the data from fugacity of CO2 (fCO2) to partial pressure of CO2 (pCO2). This is a common task in oceanography, and is performed by combining the measurements of the surface temperature and fugacity. The conversion is performed by the [pyseaflux](https://seaflux.readthedocs.io/en/latest/api.html?highlight=fCO2_to_pCO2#pyseaflux.fco2_pco2_conversion.fCO2_to_pCO2) package.

To execute this workload on the Bacalhau network we need to perform three steps:

1. Upload the data to IPFS
2. Create a docker image with the code and dependencies
3. Run the docker image on the Bacalhau network using the IPFS data

### Upload the Data to IPFS

The first step is to upload the data to IPFS. The simplest way to do this is to use a third party service to "pin" data to the IPFS network, to ensure that the data exists and is available. To do this you need an account with a pinning service like [web3.storage](https://web3.storage/) or [Pinata](https://pinata.cloud/). Once registered you can use their UI or API or SDKs to upload files.

For the purposes of this example I:
1. Downloaded the latest monthly data from the [SOCAT website](https://www.socat.info/)
2. Downloaded the latest long-term global sea surface temperature data from [NOAA](https://downloads.psl.noaa.gov/Datasets/noaa.oisst.v2/sst.mnmean.nc) - information about that dataset can be found [here](https://psl.noaa.gov/data/gridded/data.noaa.oisst.v2.highres.html).
3. Pinned the data to IPFS

This resulted in the IPFS CID of `bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y`.

<!-- TODO: Add link to notebook showing people how to upload data to IPFS -->

### Create a Docker Image to Process the Data

Next we will create the docker image that will process the data. The docker image will contain the code and dependencies needed to perform the conversion. This code originated with [lgloege](https://github.com/lgloege/bacalhau_socat_test) via [wesfloyd](https://github.com/wesfloyd/bacalhau_socat_test/). Thank you! 🤗

:::tip
For more information about working with custom containers, see the [custom containers example](../../workload-onboarding/custom-containers/).
:::

The key thing to watch out for here is the paths to the data. I'm using the default bacalhau output directory `/outputs` to write my data to. And the input data is mounted to the `/inputs` directory. But as you will see in a moment, web3.storage has added another `input` directory that we need to account for.


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


```bash
%%bash
# docker buildx build --platform linux/amd64,linux/arm64 --push -t ghcr.io/bacalhau-project/examples/socat:0.0.11 .
```

### Test the Container Locally

Before we upload the container to the Bacalhau network, we should test it locally to make sure it works.


```bash
%%bash
docker run \
	-v $(pwd)/inputs:/inputs \
	-v $(pwd)/outputs:/outputs \
	ghcr.io/bacalhau-project/examples/socat:0.0.11
```

### Run a Bacalhau Job

Now that we have the data in IPFS and the docker image pushed, we can run a job on the Bacalhau network.

I find it useful to first run a simple test with a known working container to ensure the data is located in the place I expect, because some storage providers add their own opinions. E.g. web3.storage wraps the directory uploads in a top level directory.


```bash
%%bash
rm -rf stdout stderr volumes shards
bacalhau docker run \
        --download \
        --inputs bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y \
        ubuntu -- ls /inputs
```

Then I like to run a simple test with my custom container ...


```bash
%%bash
rm -rf stdout stderr volumes shards
bacalhau docker run \
	--inputs bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y \
	--download \
	ghcr.io/bacalhau-project/examples/socat:0.0.11 -- ls -la /inputs/
```

And finally let's run the full job. This time I will not download the data immediately, because the job takes around 100s. And it takes another few minutes to download the results. The commands are below, but you will need to wait until the job completes before they work.


```bash
%%bash  --out job_id
bacalhau docker run \
        --inputs bafybeidunikexxu5qtuwc7eosjpuw6a75lxo7j5ezf3zurv52vbrmqwf6y \
        --id-only \
        --wait \
        ghcr.io/bacalhau-project/examples/socat:0.0.11 -- python main.py
```


```python
%env JOB_ID={job_id}
```

## Get Results

Now let's download and display the result from the results directory. We can use the `bacalhau get` command to download the results from the output data volume. The `--output-dir` argument specifies the directory to download the results to.


```bash
%%bash
rm -rf results
mkdir -p ./results # Temporary directory to store the results
bacalhau get --output-dir ./results ${JOB_ID} # Download the results
```


```bash
%%bash
cat ./results/stdout
```


```bash
%%bash
ls ./results/volumes/outputs
```


```python
import shutil
shutil.unpack_archive("./results/volumes/outputs/processed.zarr.zip", "./results/volumes/outputs/")
```


```python
import xarray as xr
ds = xr.open_dataset("./results/volumes/outputs/", engine='zarr')
ds.info()
```


```python
ds['pco2_ave_unwtd'].mean(dim='time').plot()
```


## References

- https://www.socat.info/
- https://seaflux.readthedocs.io/en/latest/api.html?highlight=fCO2_to_pCO2#pyseaflux.fco2_pco2_conversion.fCO2_to_pCO2
- https://github.com/lgloege/bacalhau_socat_test/blob/main/main.py
- https://github.com/wesfloyd/bacalhau_socat_test
