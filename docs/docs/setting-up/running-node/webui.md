---
sidebar_label: 'WebUI'
sidebar_position: 160
description: How to run the WebUI.
---

# Bacalhau WebUI

## Overview

The Bacalhau WebUI offers an intuitive interface for interacting with the Bacalhau network. This guide provides comprehensive instructions for setting up, deploying, and utilizing the WebUI.

For contributing to the WebUI's development, please refer to the [Bacalhau WebUI GitHub Repository](https://github.com/bacalhau-project/bacalhau/tree/main/webui).

## Spinning Up the WebUI Locally

### Prerequisites

- Ensure you have a Bacalhau v1.1.7 or later installed.

### Running the WebUI

To launch the WebUI locally, execute the following command:

```bash
bacalhau serve --node-type=requester,compute --web-ui
```

This command initializes a requester and compute node, configured to listen on `HOST=0.0.0.0` and `PORT=1234`.

### Accessing the Local WebUI

Once started, the WebUI is accessible at [http://127.0.0.1/](http://127.0.0.1/). This local instance allows you to interact with your local Bacalhau network setup.

## Accessing the WebUI from the Browser

For observational purposes, a development version of the WebUI is available at [bootstrap.development.bacalhau.org](http://bootstrap.development.bacalhau.org). This instance displays jobs from the development server.

N.b.
The development version of the WebUI is for observation only and may not reflect the latest changes or features available in the local setup.
