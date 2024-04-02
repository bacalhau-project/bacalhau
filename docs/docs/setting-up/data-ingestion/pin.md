---
sidebar_label: "Pinning data"
sidebar_position: 2
description: "How to pin data to public storage"
---
# Pinning Data

If you have data that you want to make available to your Bacalhau jobs (or other people), you can pin it using a pinning service like Pinata, NFT.Storage, Thirdweb, etc. Pinning services store data on behalf of users. The pinning provider is essentially guaranteeing that your data will be available if someone knows the CID. Most pinning services offer you a free tier, so you can try them out without spending any money.


## Basic steps

To use a pinning service, you will almost always need to create an account. After registration, you get an API token, which is necessary to control and access the files. Then you need to upload files - usually services provide a web interface, CLI and code samples for integration into your application. Once you upload the files you will get its CID, which looks like this: `QmUyUg8en7G6RVL5uhyoLBxSWFgRMdMraCRWFcDdXKWEL9`. Now you can access pinned data from the jobs via this CID.

:::info
Data source can be specified via `--input` flag, see the [CLI Guide](../../dev/cli-reference/all-flags.md#docker-run) for more details
:::
