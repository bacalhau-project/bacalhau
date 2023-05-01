---
sidebar_label: 'Running a Private Cluster'
sidebar_position: 6
---

# Deploy a private cluster

A private cluster is a network of Bacalhau nodes completely isolated from any public node.
That means you can safely process private jobs and data on your cloud or on-premise hosts!

Good news. Spinning up a private cluster is really a piece of cake :cake::

1. Install Bacalhau `curl -sL https://get.bacalhau.org/install.sh | bash` on every host
1. Run `bacalhau serve` only on one host, this will be our "bootstrap" machine
1. Copy and paste the command it outputs under the "*To connect another node to this private one, run the following command in your shell...*" line to the **other hosts**
1. Copy and paste the env vars it outputs under the "*To use this requester node from the client, run the following commands in your shell...*" line to a **client machine**
1. Run `bacalhau docker run ubuntu echo hello` on the client machine
1. That's all folks! :tada:

Optionally, set up [systemd](https://en.wikipedia.org/wiki/Systemd) units make Bacalhau daemons permanent , here's an example [systemd service file](https://github.com/bacalhau-project/bacalhau/blob/main/ops/terraform/remote_files/configs/bacalhau.service).

Please contact us on [Slack](https://join.slack.com/t/bacalhauproject/shared_invite/zt-1sihp4vxf-TjkbXz6JRQpg2AhetPzYYQ/) `#bacalhau` channel for questions and feedback!