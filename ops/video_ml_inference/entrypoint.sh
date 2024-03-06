#!/bin/bash

function print_header() {
    echo
    echo "==================== $1 ===================="
    echo
}

#
# setup for node parallel processing
# nodes figure out their rank and number of nodes in the network from these file
#

# Check if /local_data/node_rank exists and is not empty, else default to 1
if [ -f /local_data/node_rank ] && [ -s /local_data/node_rank ]; then
    NODE_RANK=$(cat /local_data/node_rank)
    print_header "Node Configuration"
    echo "  /local_data/node_rank found. NodeRank is set to: ${NODE_RANK}"
else
    NODE_RANK=1
    print_header "Node Configuration"
    echo "  /local_data/node_rank NOT found. Defaulting NodeRank to: ${NODE_RANK}"
fi

# Check if /local_data/node_count exists and is not empty, else default to 1
if [ -f /local_data/node_count ] && [ -s /local_data/node_count ]; then
    NODE_COUNT=$(cat /local_data/node_count)
    echo "  /local_data/node_count found. NodeCount is set to: ${NODE_COUNT}"
else
    NODE_COUNT=1
    echo "  /local_data/node_count NOT found. Defaulting NodeCount to: ${NODE_COUNT}"
fi

#
# Fetch the files from the bucket
#
print_header "Download Configuration"
echo "  Bucket Name: ${VIDEO_BUCKET_NAME}"
echo "  Download Destination: ${VIDEO_DOWNLOAD_DIR}"
echo
python3 /scripts/fetch_script.py --node_id "${NODE_RANK}" --total_nodes "${NODE_COUNT}" --bucket_name "${VIDEO_BUCKET_NAME}" --destination_dir "${VIDEO_DOWNLOAD_DIR}"
echo
echo "  Download process completed."

#
# YOLO video inference over the downloaded videos
#
cmd="python3 detect.py --save-csv --weights ${YOLO_WEIGHTS_PATH} --source ${VIDEO_DOWNLOAD_DIR} --project ${YOLO_PROJECT_DIR} --conf-thres=${YOLO_CONF_THRES}"
print_header "YOLO Video Inference Configuration"
echo "  Source: ${VIDEO_DOWNLOAD_DIR}"
echo "  Destination: ${YOLO_PROJECT_DIR}"
echo "  Weights Path: ${YOLO_WEIGHTS_PATH}"
echo "  Confidence Threshold: ${YOLO_CONF_THRES}"

# Check if $CLASSES is set and not empty
if [[ -n "${YOLO_CLASSES}" ]]; then
    # Append --class flag with its value to the command
    cmd="${cmd} --class ${YOLO_CLASSES}"
    echo "  Classes: ${YOLO_CLASSES}"
else
    echo "  Classes: Not specified, capturing all recognized types"
fi

echo "  Executing YOLO video inference..."
echo
eval "${cmd}"
echo
echo "  YOLO video inference completed."

# TODO: Add python script to process contents out /outputs which contains labeled video crops and move them to a vector database.
# As an example, the structure of outputs is shown below.
#   The .mp4 files are videos that have been analyzed with bounding boxes of objects.
#   The directories in 'crop' like bench, chair, etc. all contain crops of the extracted videos as jpegs.
# outputs/
  #├── exp
  #│   ├── --3ouPhoy2A_000020_000030.mp4
  #│   ├── --4-0ihtnBU_000058_000068.mp4
  #│   ├── --56QUhyDQM_000185_000195.mp4
  #│   ├── --6q_33gNew_000132_000142.mp4
  #│   ├── crops
  #│       ├── bench
  #│       ├── chair
  #│       ├── knife
  #│       ├── person
  #│       ├── sports ball
  #│       ├── tennis racket
  #│       ├── tie
  #│       └── wine glass

# to add the scrip add a script:
# - add it in the directory of the Dockerfile
# - modify the dockerfile to copy the script into the container
# - modify the dockerfile to install any dependencies of the script
# - call the scrip here
