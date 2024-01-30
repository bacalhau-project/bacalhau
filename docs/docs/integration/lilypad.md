---
sidebar_label: 'Lilypad for Web3'
sidebar_position: 3
description: Lilypad is a distributed compute network for web3 based on Bacalhau. It currently enables the running of Bacalhau jobs from smart contracts.
---

# 🍃 What is Lilypad?

## Full Documentation

[Lilypad Docs](https://docs.lilypadnetwork.org)

## Vision

[Lilypad](https://blog.lilypadnetwork.org/) is aiming to build an internet-scale trustless distributed compute network for web3. Creating the infrastructure for use cases like AI inference, ML training, DeSci and more.

<iframe src="https://platform.twitter.com/embed/Tweet.html?dnt=true&embedId=twitter-widget-0&features=eyJ0ZndfdGltZWxpbmVfbGlzdCI6eyJidWNrZXQiOltdLCJ2ZXJzaW9uIjpudWxsfSwidGZ3X2ZvbGxvd2VyX2NvdW50X3N1bnNldCI6eyJidWNrZXQiOnRydWUsInZlcnNpb24iOm51bGx9LCJ0ZndfdHdlZXRfZWRpdF9iYWNrZW5kIjp7ImJ1Y2tldCI6Im9uIiwidmVyc2lvbiI6bnVsbH0sInRmd19yZWZzcmNfc2Vzc2lvbiI6eyJidWNrZXQiOiJvbiIsInZlcnNpb24iOm51bGx9LCJ0ZndfZm9zbnJfc29mdF9pbnRlcnZlbnRpb25zX2VuYWJsZWQiOnsiYnVja2V0Ijoib24iLCJ2ZXJzaW9uIjpudWxsfSwidGZ3X21peGVkX21lZGlhXzE1ODk3Ijp7ImJ1Y2tldCI6InRyZWF0bWVudCIsInZlcnNpb24iOm51bGx9LCJ0ZndfZXhwZXJpbWVudHNfY29va2llX2V4cGlyYXRpb24iOnsiYnVja2V0IjoxMjA5NjAwLCJ2ZXJzaW9uIjpudWxsfSwidGZ3X3Nob3dfYmlyZHdhdGNoX3Bpdm90c19lbmFibGVkIjp7ImJ1Y2tldCI6Im9uIiwidmVyc2lvbiI6bnVsbH0sInRmd19kdXBsaWNhdGVfc2NyaWJlc190b19zZXR0aW5ncyI6eyJidWNrZXQiOiJvbiIsInZlcnNpb24iOm51bGx9LCJ0ZndfdXNlX3Byb2ZpbGVfaW1hZ2Vfc2hhcGVfZW5hYmxlZCI6eyJidWNrZXQiOiJvbiIsInZlcnNpb24iOm51bGx9LCJ0ZndfdmlkZW9faGxzX2R5bmFtaWNfbWFuaWZlc3RzXzE1MDgyIjp7ImJ1Y2tldCI6InRydWVfYml0cmF0ZSIsInZlcnNpb24iOm51bGx9LCJ0ZndfbGVnYWN5X3RpbWVsaW5lX3N1bnNldCI6eyJidWNrZXQiOnRydWUsInZlcnNpb24iOm51bGx9LCJ0ZndfdHdlZXRfZWRpdF9mcm9udGVuZCI6eyJidWNrZXQiOiJvbiIsInZlcnNpb24iOm51bGx9fQ%3D%3D&frame=false&hideCard=false&hideThread=false&id=1667164746599002114&lang=en&origin=https%3A%2F%2Fcdn.iframe.ly%2F6kNS5bO%3Fapp%3D1&sessionId=a65b4df788ce5d41bbb126f645b7d4046f3c62bb&theme=light&widgetsVersion=aaf4084522e3a%3A1674595607486&width=850px" style={{width: '100%', height: '500px'}}></iframe>

## Overview

Lilypad (v0) currently enables users to access verifiable, distributed off-chain compute directly from smart contracts.\
\
Lilypad is at v0 and is a Proof of Concept project operating as an integration layer between Bacalhau compute jobs and solidity smart contracts. This integration enables users to access verifiable off-chain decentralised compute from DApps and smart contract projects, enabling interactions and innovations between on-chain and off-chain compute. \
\
Lilypad v0 does not charge for compute jobs, outside of network running fees (ie. the cost of transactions on the blockchain network it is deployed to). It operates on the [Bacalhau](https://www.docs.bacalhau.org) public compute network (which is free to use), though it is worth noting that there are no reliability guarantees given for this network (which is something future versions of this protocol will be working to improve\

<iframe width="560" height="315" src="https://www.youtube.com/embed/9lF7omNEK-c" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Lilypad Roadmap

- v0: **September 2022** - Lilypad Bridge POC for triggering and returning Bacalhau compute jobs from a smart contract
- v1: **July 2023** - A [modicum](https://dl.acm.org/doi/pdf/10.1145/3401025.3401737)-based minimal testnet (EVM-based). See [github](https://github.com/bacalhau-project/lilypad)
- v2: **September 2023** - A more robust trustless distributed testnet
- v3: tbd - Lilypad Mainnet

<figure><img src="/img/lilypad/Lilypad%20Roadmap%20June.png" alt="" /><figcaption><p>Lilypad Roadmap</p></figcaption></figure>

# Lilypad v0 Reference

## Architecture

### Overview

Lilypad is a ‘bridge’ to enable computation jobs from smart contracts. The aim of Lilypad v0 was to create an integration for users to call Bacalhau jobs directly from their solidity smart contracts and hence enable interactions and innovations between on-chain and off-chain compute.\
\
Lilypad v0 is a proof of concept bridge which runs off the public (free to use) Bacalhau compute network. As such, the reliability of jobs on this network are not guaranteed.

> If you have a need for reliable compute based on this infrastructure - get in touch with us.

<figure><img src="/img/lilypad/Lilypad Architecture.png" alt="" /><figcaption><p>Lilypad v0 on the FVM Network</p></figcaption></figure>

A user contract implements the LilypadCaller interface and to call a job, they make a function call to the deployed LilypadEvents contract.

This contract emits an event which the Lilypad bridge daemon listens for and then forwards on to the Bacalhau network for processing.

Once the job is complete, the results are returned back to the originating user contract from the bridge code.

<figure><img src="https://user-images.githubusercontent.com/12529822/224299570-366bde1c-1f48-4af9-9d7c-0d4f8a0fc1fc.png" alt="" /><figcaption><p>Note: runBacalhauJob() is now runLilypadJob()</p></figcaption></figure>

See more about how Bacalhau & Lilypad are related below:

- [Bacalhau Notion page](https://www.notion.so/7-Introduction-to-Bacalhau-Decentralised-Compute-over-Data-AI-ML-DeSci-fbef1ef73b4e479a9b209be8d29cb58f)

- FVM Hackerbase Video
<iframe width="560" height="315" src="https://www.youtube.com/embed/drwcj2kk6bA" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Lilypad v0 Quick Start

## Prefer Video?

:::info
Note: Since this video was released some changes have been made to the underlying code, but the process and general architecture remains the same.
:::

<iframe width="560" height="315" src="https://www.youtube.com/embed/B0l0gFYxADY" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## Quick Start Guide

:::info
The Lilypad Contracts are not currently importable via npm (though this is in progress), so to import them to you own project, you'll need to use their github links
:::

Using Lilypad in your own solidity smart contract requires the following steps

1.  Create a contract that implements the [LilypadCaller](https://github.com/bacalhau-project/lilypad-v0/blob/main/hardhat/contracts/LilypadCallerInterface.sol) interface.&#x20;

    As part of this interface you need to implement 2 functions:

    - `lilypadFulfilled` - a callback function that will be called when the job completes successfully
    - `lilypadCancelled` - a callback function that will be called when the job fails

2.  Provide a public [Docker Spec compatible for use on Bacalhau](https://docs.bacalhau.org/getting-started/docker-workload-onboarding) in JSON format to the contract.
3.  To trigger a job from your contract, you need to call the `LilypadEvents` contract which the Lilypad bridge is listening to and which connects to the Bacalhau public network. Create an instance of [`LilypadEvents`](https://github.com/bacalhau-project/lilypad-v0/blob/main/hardhat/contracts/LilypadEvents.sol) by passing the public contract address on the network you are using (see [deployed network details](https://docs.lilypadnetwork.org/lilypad-v0-reference/deployed-network-details)) to the `LilypadEvents` constructor.
4.  Call the [LilypadEvents](https://github.com/bacalhau-project/lilypad-v0/blob/main/hardhat/contracts/LilypadEvents.sol) contract function `runLilypadJob()` passing in the following parameters.&#x20;

|     Name      |                                                              Type                                                               |                                                                                                             Purpose                                                                                                              |
| :-----------: | :-----------------------------------------------------------------------------------------------------------------------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
|    `_from`    |                                                            `address`                                                            |                                         The address of the calling contract, to which success or failure will be passed back. You should probably use address(this) from your contract.                                          |
|    `_spec`    |                                                            `string`                                                             |                                                                    A Bacalhau job spec in JSON format. See below for more information on creating a job spec.                                                                    |
| `_resultType` | [`LilypadResultType`](https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadCallerInterface.sol#L4-L9) | The type of result that you want to be returned. If you specify CID, the result tree will come back as a retrievable IPFS CID. If you specify StdOut, StdErr or ExitCode, those raw values output from the job will be returned. |

## Implement the LilypadCaller Interface in your contract

Create a contract that implements [`LilypadCallerInterface`](https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadCallerInterface.sol). As part of this interface you need to implement 2 functions:

- `lilypadFulfilled` - a callback function that will be called when the job completes successfully
- `lilypadCancelled` - a callback function that will be called when the job fails

```solidity
  /** === LilypadCaller Interface === **/
  pragma solidity >=0.8.4;
  import 'https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadCallerInterface.sol' //Location of file link

  /** === User Contract Example === **/
  contract MyContract is LilypadCallerInterface {

      function lilypadFulfilled(address _from, uint _jobId,
          LilypadResultType _resultType, string calldata _result)
          external override {
          // Do something when the LilypadEvents contract returns
          // results successfully
      }

      function lilypadCancelled(address _from, uint _jobId, string
          calldata _errorMsg) external override {
          // Do something if there's an error returned by the
          // LilypadEvents contract
      }
  }

```

## Add a Spec compatible with [Bacalhau](https://www.docs.bacalhau.org)

:::info
There are several public examples you can try out without needing to know anything about Docker or WASM specification jobs -> see the [Bacalhau Docs](https://www.docs.bacalhau.org). The full specification for Bacalhau jobs can be [seen here](https://docs.bacalhau.org/all-flags).
:::

Bacalhau operates by executing jobs within containers. This means it is able to run any arbitrary Docker jobs or WASM images

We'll use the public Stable Diffusion Docker Container[ located here](https://github.com/bacalhau-project/examples/pkgs/container/examples%2Fstable-diffusion-gpu) for this example.

Here's an example JSON job specification for the Stable Diffusion job:

```json
{
  "Engine": "docker",
  "Verifier": "noop",
  "PublisherSpec": { "Type": "ipfs" },
  "Docker": {
    "Image": "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
    "Entrypoint": ["python"],
    "Parameters": [
      "main.py",
      "--o",
      "./outputs",
      "--p",
      "A User Prompt Goes here"
    ]
  },
  "Resources": { "GPU": "1" },
  "Outputs": [{ "Name": "outputs", "Path": "/outputs" }],
  "Deal": { "Concurrency": 1 }
}
```

Here's an example of using this JSON specification in solidity:

Note that since we need to be able to add the user prompt input to the spec, it's been split into 2 parts.

```solidity
string constant specStart = '{'
    '"Engine": "docker",'
    '"Verifier": "noop",'
    '"PublisherSpec": {"Type": "ipfs"},'
    '"Docker": {'
    '"Image": "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",'
    '"Entrypoint": ["python"],
    '"Parameters": ["main.py", "--o", "./outputs", "--p", "';

string constant specEnd =
    '"]},'
    '"Resources": {"GPU": "1"},'
    '"Outputs": [{"Name": "outputs", "Path": "/outputs"}],'
    '"Deal": {"Concurrency": 1}'
    '}';


//Example of use:
string memory spec = string.concat(specStart, _prompt, specEnd);
```

:::info
See more about how to [onboard your Docker Workloads for Bacalhau](https://docs.bacalhau.org/getting-started/docker-workload-onboarding/), [Onboard WebAssembly Workloads](https://docs.bacalhau.org/getting-started/wasm-workload-onboarding) or [Work with Custom Containers](https://docs.bacalhau.org/examples/workload-onboarding/custom-containers/) in the Bacalhau Docs.
:::

## Add the Lilypad Events Address & Network Fee

You can do this by either passing it into your constructor or setting it as a variable

```solidity
// SPDX-License-Identifier: MIT
pragma solidity >=0.8.4;
import "https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadEventsUpgradeable.sol";
import "https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadCallerInterface.sol";

/** === User Contract Example === **/
contract MyContract is LilypadCallerInterface {
  address public bridgeAddress; // LilypadEvents contract address for interacting with the deployed LilypadEvents contract
  LilypadEventsUpgradeable bridge; // Instance of the LilypadEvents Contract to interact with
  uint256 public lilypadFee; //=30000000000000000 on FVM;

  constructor(address _bridgeContractAddress) {
    bridgeAddress = _bridgeContractAddress; //the LilypadEvents contract address for your network
    bridge = LilypadEventsUpgradeable(_bridgeContractAddress); //create an instance of the Events Contract to interact with
    uint fee = bridge.getLilypadFee(); // you can fetch the fee amount required for the contract to run also
    lilypadFee = fee;
  }

  function lilypadFulfilled(address _from, uint _jobId,
    LilypadResultType _resultType, string calldata _result)
    external override {
    // Do something when the LilypadEvents contract returns
    // results successfully
  }

  function lilypadCancelled(address _from, uint _jobId, string
    calldata _errorMsg) external override {
    // Do something if there's an error returned by the Lilypad Job
  }
}
```

## Call the LilypadEvents runLilypadJob function

Using the LilypadEvents Instance, we can now send jobs to the Bacalhau Network via our contract using the `runLilypadJob()` function.

In this example we'll use the Stable Diffusion Spec shown above in [#add-a-spec-compatible-with-bacalhau](https://docs.lilypadnetwork.org/lilypad-v0-reference/lilypad-v0-quick-start#add-a-spec-compatible-with-bacalhau)

:::info
Note that calling the runLilypadJob() function requires a network fee. While the Bacalhau public Network is currently free to use, gas fees are still needed to return the results of the job performed. This is the payable fee in the contract.
:::

```solidity
// SPDX-License-Identifier: MIT
pragma solidity >=0.8.4;
import "https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadEventsUpgradeable.sol";
import "https://github.com/bacalhau-project/lilypad/blob/main/hardhat/contracts/LilypadCallerInterface.sol";

/** === User Contract Example === **/
contract MyContract is LilypadCallerInterface {
  address public bridgeAddress; // Variable for interacting with the deployed LilypadEvents contract
  LilypadEventsUpgradeable bridge;
  uint256 public lilypadFee; //=30000000000000000;

  constructor(address _bridgeContractAddress) {
    bridgeAddress = _bridgeContractAddress;
    bridge = LilypadEventsUpgradeable(_bridgeContractAddress);
    uint fee = bridge.getLilypadFee(); // you can fetch the fee amount required for the contract to run also
    lilypadFee = fee;
  }

  //** Define the Bacalhau Specification */
  string constant specStart = '{'
      '"Engine": "docker",'
      '"Verifier": "noop",'
      '"PublisherSpec": {"Type": "ipfs"},'
      '"Docker": {'
      '"Image": "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",'
      '"Entrypoint": ["python"],
      '"Parameters": ["main.py", "--o", "./outputs", "--p", "';

  string constant specEnd =
      '"]},'
      '"Resources": {"GPU": "1"},'
      '"Outputs": [{"Name": "outputs", "Path": "/outputs"}],'
      '"Deal": {"Concurrency": 1}'
      '}';


  /** Call the runLilypadJob() to generate a stable diffusion image from a text prompt*/
  function StableDiffusion(string calldata _prompt) external payable {
      require(msg.value >= lilypadFee, "Not enough to run Lilypad job");
      // TODO: spec -> do proper json encoding, look out for quotes in _prompt
      string memory spec = string.concat(specStart, _prompt, specEnd);
      uint id = bridge.runLilypadJob{value: lilypadFee}(address(this), spec, uint8(LilypadResultType.CID));
      require(id > 0, "job didn't return a value");
      prompts[id] = _prompt;
  }

  /** LilypadCaller Interface Implementation */
  function lilypadFulfilled(address _from, uint _jobId,
    LilypadResultType _resultType, string calldata _result)
    external override {
    // Do something when the LilypadEvents contract returns
    // results successfully
  }

  function lilypadCancelled(address _from, uint _jobId, string
    calldata _errorMsg) external override {
    // Do something if there's an error returned by the
    // LilypadEvents contract
  }
}
```

---

## description: Lilypad v0 Integrated Networks

# Deployed Network Details

:::info
If you have a use case for another network - please get in touch with us!
:::

The Lilypad Events contract - used for triggering compute jobs on Bacalhau, is currently integrated to the following networks on the address specified:

## Lilypad v0 Deployed Networks

|                                        |                                            |                                                                                                                                                                                                                                                                                    |                     |                                                                                                                                                                                                                            |                                                                                                                                                                                                              |
| -------------------------------------- | ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Chain Name**                         | **LilypadEvents Contract Address**         | **RPC**                                                                                                                                                                                                                                                                            | **ChainID**         | **BlockExplorer**                                                                                                                                                                                                          | **Faucet**                                                                                                                                                                                                   |
| Filecoin Calibration Net (**testnet**) | 0xdC7612fa94F098F1d7BB40E0f4F4db8fF0bC8820 | [https://api.calibration.node.glif.io/rpc/v0](https://api.calibration.node.glif.io/rpc/v0)                                                                                                                                                                                         | 314159 (0x4cb2f)    | [https://calibration.filscan.io/](https://calibration.filscan.io/),                                                                                                                                                        | [https://faucet.calibration.fildev.network/](https://faucet.calibration.fildev.network/)                                                                                                                     |
| Filecoin Mainnet                       | 0xc18879C0a781DdFa0258302467687413AaD5a4E6 | [https://api.node.glif.io/rpc/v1](https://api.node.glif.io/rpc/v1), [https://filecoin-mainnet.chainstacklabs.com/rpc/v1](https://filecoin-mainnet.chainstacklabs.com/rpc/v1), [https://rpc.ankr.com/filecoin](https://rpc.ankr.com/filecoin)                                       | 314 (0x13a)         | [https://fvm.starboard.ventures/](https://fvm.starboard.ventures/), [https://explorer.glif.io/](https://explorer.glif.io/), [https://beryx.zondax.ch/](https://beryx.zondax.ch/), [https://filfox.io/](https://filfox.io/) | Requires Filecoin token [See docs](https://docs.filecoin.io/basics/assets/get-fil/)                                                                                                                          |
| Mantle Testnet                         | 0xdC7612fa94F098F1d7BB40E0f4F4db8fF0bC8820 | [https://rpc.testnet.mantle.xyz](https://rpc.testnet.mantle.xyz)                                                                                                                                                                                                                   | 5001 (0x1389)       | [https://explorer.testnet.mantle.xyz/](https://explorer.testnet.mantle.xyz/)                                                                                                                                               | [https://faucet.testnet.mantle.xyz/](https://faucet.testnet.mantle.xyz/)                                                                                                                                     |
| Sepolia Testnet                        | 0xdC7612fa94F098F1d7BB40E0f4F4db8fF0bC8820 | [https://rpc2.sepolia.org](https://rpc2.sepolia.org), [https://eth-sepolia.g.alchemy.com/v2/demo](https://eth-sepolia.g.alchemy.com/v2/demo), [https://rpc.sepolia.org](https://rpc.sepolia.org), see [https://chainlist.org/chain/11155111](https://chainlist.org/chain/11155111) | 11155111 (0xaa36a7) | [https://sepolia.etherscan.io/](https://sepolia.etherscan.io/)                                                                                                                                                             | [https://www.infura.io/faucet/sepolia](https://www.infura.io/faucet/sepolia), [https://sepoliafaucet.com/](https://sepoliafaucet.com/), [https://sepolia-faucet.pk910.de/](https://sepolia-faucet.pk910.de/) |
| Polygon Mumbai                         | 0xdC7612fa94F098F1d7BB40E0f4F4db8fF0bC8820 | see [https://chainlist.org/chain/80001](https://chainlist.org/chain/80001)                                                                                                                                                                                                         | 80001 (0x13881)     | [https://mumbai.polygonscan.com/](https://mumbai.polygonscan.com/)                                                                                                                                                         | [https://faucet.polygon.technology/](https://faucet.polygon.technology/), [https://mumbaifaucet.com/](https://mumbaifaucet.com/)                                                                             |
| Polygon Mainnet (coming soon)          |                                            | see [https://chainlist.org/chain/137](https://chainlist.org/chain/137)                                                                                                                                                                                                             | 137 (0x89)          | [https://polygonscan.com/](https://polygonscan.com/)                                                                                                                                                                       | Requires MATIC tokens                                                                                                                                                                                        |
| Optimism (coming soon)                 |                                            | see [https://chainlist.org/chain/10](https://chainlist.org/chain/10)                                                                                                                                                                                                               | 10(0xa)             |                                                                                                                                                                                                                            | Requires OP tokens                                                                                                                                                                                           |
| Arbitrum One (coming soon)             |                                            | see [https://chainlist.org/chain/42161](https://chainlist.org/chain/42161)                                                                                                                                                                                                         | 42161 (0xa4b1)      |                                                                                                                                                                                                                            | Requires ARB tokens                                                                                                                                                                                          |
