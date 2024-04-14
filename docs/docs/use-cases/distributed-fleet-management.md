---
sidebar_label: 'Distributed Fleet Management'
sidebar_position: 2
---

# Managing Your Global Device Fleet

## Introduction

As the world rapidly shifts towards the cloud, it is often easy to forget that our data is not just floating around in some ethereal plane. The less exciting reality is that data is physically stored somewhere, whether it is in a cloud provider’s huge data centers or on a tiny, embedded flash drive. In the age of the Internet of Things (IoT), businesses are now tasked with overseeing thousands of virtual and physical assets. The cloud is just one of them.

Much like a naval fleet is composed of a range of sea vessels, a company’s fleet can encompass a range of devices, from security cameras to satellites, each with its own specific job, configuration, and role within the organization’s operations. Monitoring and maintaining a fleet is challenging, time-consuming, and costly due to the sheer number and diversity of devices.

## Challenges With Traditional Fleet Management
As fleets expand, businesses are faced with mounting security risks and an increasingly complex web of operational assets. Every single device requires consistent maintenance, rigorous security protocols, and regular software updates, whether it’s an on-premise server or a meteorological sensor in the middle of the Pacific ocean. Without a comprehensive distributed fleet management framework in place, businesses lack the visibility that is vital to maintaining their assets and identifying potential vulnerabilities.

1. **Manual Tracking**: Engineers check configurations and perform spot-checks using one-off scripts and manual processes. In smaller fleets, this might be manageable, but as fleets grow in size and complexity, these methods become inefficient, error-prone, and unsustainable, especially in distributed networks. In short, manual methods just don’t scale.

1. **Limited Visibility**: Basic system-level monitoring offers essential metrics but lacks comprehensive visibility and the capability to query fleet data.  Further, without distributed metrics, any signal and necessary actions will be delayed.

1. **Costs In Centralizing Telemetry**: Centralizing logs about machines provides valuable insights, but it involves moving large amounts of data, which is costly, time-consuming, and risky – especially just for observing fairly small bits of information about the machines. This can lead to increased storage costs, potential network bottlenecks, longer data retrieval times, and heightened security concerns due to the concentration of sensitive information in one location.

1. **Challenges With Adopting Cloud First Solutions**: While cloud providers offer a variety of fleet management tools, they often only solve half of the problem. Many businesses cannot migrate their entire infrastructure to the cloud for a variety of reasons. Sometimes it is simply impractical, but often it is due to security concerns, regulatory requirements, economical motives, or a cluster of edge/IoT devices that are inherently off-cloud.

## Solution: Unified and Simplified Fleet Management Using Bacalhau
### Implementing a Unified Fleet Management Platform Benefits

The Bacalhau platform makes distributed fleet management easy, because it was designed for the distributed world. You can seamlessly integrate our lightweight agent into your existing infrastructure and gain real-time visibility of every single operational asset in your fleet without worrying about compatibility issues, device architecture or physical location. You are essentially wrapping all of the above solutions into one!

1. **Simplified Installation**: Joining your fleet together is just one command to install the Bacalhau agent, which can be done by following the instructions on [setting up a private cluster](../setting-up/networking-instructions/private-cluster.md). After that, you can run any application you want: Docker, WASM, or any binary of your choosing. Your architecture, your security, your fleet!
1. **Internal Control**: Bacalhau orchestrates jobs to run where your data lives. This means that fleet engineers can perform immediate diagnostics and real-time analytics of devices without having to go through a central server. Not only is this faster and cheaper, but it mitigates the security risks associated with transmitting data across open networks. This flexibility and granularity allows you to make pivotal decisions based on tailored insights and streamlines your workflow by decluttering your data landscape.
1. **Unified Automation and Security Scanning**: The Bacalhau agent can host open-source tools like osquery or ossec, which enable you to automate any of your fleet management tasks, such as:
    1. File integrity monitoring and anomaly detection
    1. Device configuration and provisioning
    1. Performance metrics
    1. Host intrusion detection
    1. Asset Tracking
    1. Compliance management
You can schedule these tasks to routinely send logs to your preferred logging service (see [log processing](./log-processing.md)) or selectively capture events when certain conditions are met. For example, you might want to set your edge cluster to transmit logs only if device temperatures rise above the standard range or configure your cloud resources to alert you when asset utilization deviates from predicted patterns. This is the strength of Bacalhau - you choose what you want to see.
1. **Real-Time Querying**: Automation is key to efficient and effective fleet management, but ad-hoc queries remain a vital component for engineers who need to quickly investigate critical issues flagged by the automated system. Bacalhau makes this process easy by offering robust command-line tools that empower engineers to run precise, on-the-fly searches and obtain immediate results.
You can query your entire fleet with just a few lines of code:

```SQL
./query-cluster \
  "SELECT system_info.hostname, uid, username, path,
          encrypted, key_type
     FROM system_info, users CROSS JOIN user_ssh_keys USING (uid)
    WHERE encrypted=0 OR key_type<>'ec
```

In this example we used a SQL-like query to check that all users have encrypted their SSH keys. This will return any keys that don’t meet the policy, giving the hostname, UID, username, and the path to the key file - along with the encryption and key type of that key, so you can see how they fail.

## Summary: Deploying and Managing Your Fleet with Bacalhau is Cheaper, Faster, and More Reliable

Bacalhau gives you the **visibility**, **unity** and **control** needed for comprehensive distributed fleet management. Once you have installed Bacalhau on all of your devices, you are ready to take full advantage of the network improvements:

1. **Cheap** – Only move the data that matters, reduce engineering workloads and gain invaluable insights from a system tailored to your business needs.
1. **Fast** – Get immediate results from automated logging or ad-hoc queries.
1. **Reliable** – Our distributed network ensures that workloads are spread across multiple devices, so you won’t experience any downtime.

Our platform is a fully open-source playground for all things distributed. Fleet management is just one of the many features that Bacalhau supports. Take a look at our other use-cases (links here) to see what else we can do for your business.

For a more in-depth tutorial and set-up guide, please read our blog on distributed fleet security.