#!/bin/bash

# Define the arrays for Executor, StorageSources, and Publishers
# add "WASM" later when templates are ready
executors=("docker")
storageSources=("" "urlDownload" "repoClone" "repoCloneLFS" "s3" "ipfs")
publishers=("" "ipfs" "s3")

# Begin the JSON array
echo "[" > permutations.json

# Initialize a boolean to handle the comma placement in JSON array
first=true

# Loop through each combination of Executor, StorageSources, and Publishers
for count in 1 2; do
    for executor in "${executors[@]}"; do
        for storage in "${storageSources[@]}"; do
            for publisher in "${publishers[@]}"; do
                # Construct the JSON object
                if [ "$first" = true ]; then
                    first=false
                else
                    echo "," >> permutations.json
                fi
                printf '{"Engine":"%s", "StorageSources":"%s", "Publishers":"%s", "Count":%d}\n' \
                "$executor" "$storage" "$publisher" $count >> permutations.json
            done
        done
    done
done

# Close the JSON array
echo "]" >> permutations.json