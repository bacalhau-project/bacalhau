# Bacalhau Python SDK

This is the official Python SDK for Bacalhau, named `bacalhau-sdk`.
It is a **high-level** SDK that ships all the client-side logic (e.g. signing requests) needed to query the endpoints. 
Please take a look at [the examples](./examples) for snippets to create, list and inspect jobs. 
Under the hood, this uses [bacalhau-apiclient](../clients/README.md) to call the API.

Please use this library in projects instead of the low-level bacalhau-apiclient.


## Install

Clone the public repository:

``` console
$ git clone https://github.com/filecoin-project/bacalhau/
```

Once you have a copy of the source, you can install it with:

``` console
$ cd python/
$ pip install .
```

## Devstack

You can set the environment variables `BACALHAU_API_HOST` and `BACALHAU_API_PORT` to point this SDK to your Bacalhau API (e.g. local devstack).

