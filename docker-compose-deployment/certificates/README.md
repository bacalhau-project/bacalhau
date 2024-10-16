# Certificate Generation

The script in the folder allows you to generate certificates that are signed by a root CA, and provide the
CN and SAN for these leaf certs. The generated certs will be added to the `generated_assets` directory.

Usage: `./generate_leaf_certs.sh <CN_and_SAN>`
```shell
./generate_leaf_certs.sh my-bacalhau-requester-node
```
