# Bacalhau Master Plan Roadmap

## MAY

1. *(basic)* Build a system for unreliably running a single deterministic program where a single 10MB piece of data is on IPFS, assuming everyone participating is trustworthy, assuming only 10 nodes in the network.*Example:* Run cloud detection on a single Landsat image file. Get the result back. Verify it by eye.
    1. **STATUS** Complete.


## JUNE

1. ***(reliability)* Extend that system to work 99% of the time. Submit 10,000 jobs and show that at most 100 of them fail. It might take several minutes to resolve each job.**
    1. **Example:** By the end of this phase, job execution will be significantly more reliable. We’ll generally be able to submit 50 jobs and have them all succeed.
    2. **Status:** Final benchmarking cluster in-flight.
2. *(scale-1)* Extend that system to work when 100 nodes and have access to 100TB of data. Still, the error rate might be high.
    1. **Example:** 90 more nodes join the network and at first things break, but by the end of the milestone the network is working well again, albeit slowly and with some error rate

## JULY

1. *(multi-file)* Extend that system to work when jobs consist of many (thousands) of files, rather than a single file, and we want to distribute the work across the network and run it in parallel.
    1. **Example:** A user can submit a cloud detection on 10,000 Landsat images at once and have the work be parallelised automatically on the network, still according to data locality where possible.
2. *(scale-2)* Extend the system to work when 1000 nodes are participating, over 1PB of data. Resolving jobs now may take a very long time (10s of minutes).
    1. **Example:** Many users can run landsat data in parallel, along with use cases on public biomedical images and 9 other use cases without the network failing.

## AUGUST

1. *(performance-1)* Get the resolution of jobs down to seconds, even in large networks where 1000s of nodes are participating with hundreds of job submissions per second.
    1. **Example:** As the network is dealing with a multitude of use cases (landsat, biomedical, SETI@home and protein folding has migrated over to use Bacalhau, etc) and the network is processing hundreds of job executions per second, it's now started to slow down a lot. This phase is all about getting it speedy again.
2. *(filecoin)* Add support for reading datasets from Filecoin so that data in that network becomes accessible to IPCS workloads
    1. **Example:** A big data provider has put petabytes of public data onto Filecoin. Bacalhau users can consume it by attaching a Filecoin wallet to their Bacalhau node and giving it a spending limit.

## SEPTEMBER

1. *(byzantine-1)* Extend that system to work when up to 10% of the nodes are malicious.
    1. **Example:** Even when a small minority of nodes are trying to mess up the results, a user can still run cloud detection on 10,000 files in IPFS with no errors or incorrect results.

1. *(dag)* Extend that system to support jobs that are described in terms of pipelines: the output of one job feeding into the input of the next.
    1. **Example:** Cloud removal in the Landsat job is actually a pipeline which first detects images with clouds, then only for those images, forwards them to a pipeline which removes the clouds.

## OCTOBER (TBD)

1. *(byzantine-2)* Extend that system to work when up to 49% of the nodes are malicious.
    1. **Example:** A larger attack happens on the network (>10%, <50%). Before this phase, this attack would bring down the network. After this phase, the network would carry on operating (although potentially degraded, higher latencies etc).
1. *(nondeterminism)* Extend that system to work with execution runtimes that are non-deterministic, e.g. arbitrary user-provided container images, to support workloads such ML training. In particular, this would prove out that the system is pluggable in terms of verification strategies, this lays the groundwork for future support for other strategies in the triad of trustless compute, such as cryptographic verifiability, secrecy and optimistic verifiability.
    1. **Example:** Nondeterministic workloads, or ones that can’t be expressed as deterministic WASM binaries, can now be run on the network, for example training ML models.

To be continued in [master plan part two]([url](https://hackmd.io/i-UdANDVSwycXtVacIPgEg))
