---

sidebar_label: Network
---

# Network Specification

The `Network` object offers a method to specify the networking requirements of a `Task`. It defines the scope and constraints of the network connectivity based on the demands of the task.

## `Network` Parameters:

- **Type** `(string: "None")`: Indicates the network configuration's nature. There are several network modes available:
    - `None`: This mode implies that the task does not necessitate any networking capabilities.
    - `Full`: Specifies that the task mandates unrestricted, raw IP networking without any imposed filters.
    - `HTTP`: This mode constrains the task to only require HTTP networking with specific domains. In this model:
        - The job specifier puts forward a job, stipulating the domain(s) it intends to communicate with.
        - The compute provider assesses the inherent risk of the job based on these domains and bids accordingly.
        - At runtime, the network traffic remains strictly confined to the designated domain(s).

:::info
A typical command for this might resemble:
    ```
bacalhau docker run —network=http —domain=crates.io —domain=github.com -i ipfs://Qmy1234myd4t4,dst=/code rust/compile
    ```
:::

      The primary risks for the compute provider center around possible violations of its terms, its hosting provider's terms, or even prevailing laws in its jurisdiction. This encompasses issues such as unauthorized access or distribution of illicit content and potential cyber-attacks.

      Conversely, the job specifier's primary risk involves operating in a paid environment. External entities might seek to exploit this environment, for instance, through a compromised package download that initiates a cryptomining operation, depleting the allocated, prepaid job time. By limiting traffic strictly to the pre-specified domains, the potential for such cyber threats diminishes considerably.

      While a compute provider might impose its limits through other means, having domains declared upfront allows it to selectively bid on jobs that it can execute without issues, improving the user experience for job specifiers.

- **Domains** `(string[]: <optional>)`: A list of domain strings, relevant primarily when the `Type` is set to **HTTP**. It dictates the specific domains the task can communicate with over HTTP.

Understanding and utilizing these configurations aptly can ensure that tasks are executed in an environment that aligns with their networking requirements, bolstering efficiency and security.
