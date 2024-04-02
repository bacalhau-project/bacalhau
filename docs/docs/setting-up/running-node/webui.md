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

- Ensure you have a Bacalhau v1.1.7 or later installed, preferably use the latest version.

### Running the WebUI

To launch the WebUI locally, add the `--web-ui` flag to the `bacalhau serve` command:

```bash
bacalhau serve --web-ui
```

This command initializes a node with enabled WebUI. By default it is listening 8483 port, which can be changed via `--web-ui-port` flag or `node.webui.port` config parameter.

### Accessing the Local WebUI

Once started, the WebUI is accessible at [http://127.0.0.1:8483](http://127.0.0.1:8483). This local instance allows you to interact with your local Bacalhau network setup.

## Accessing the demo network WebUI

For observational purposes, a development version of the WebUI is available at [bootstrap.development.bacalhau.org](http://bootstrap.development.bacalhau.org). This instance displays jobs from the demo development server.

<<<<<<< HEAD
:::info
The demo version of the WebUI is for observation only and may not reflect the latest changes or features available in the local setup.
:::
=======
N.b.
The development version of the WebUI is for observation only and may not reflect the latest changes or features available in the local setup.
>>>>>>> main
